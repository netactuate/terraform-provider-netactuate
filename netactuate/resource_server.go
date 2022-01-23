package netactuate

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	tries       = 60
	intervalSec = 10
)

var credentialKeys = []string{"password", "ssh_key_id"}

func resourceServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerCreate,
		ReadContext:   resourceServerRead,
		DeleteContext: resourceServerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"plan": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"location_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"image_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"password": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Sensitive:    true,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
			},
			"ssh_key_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
			},
			"cloud_config": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	s, err := c.CreateServer(d.Get("hostname").(string), d.Get("plan").(string), d.Get("location_id").(int),
		d.Get("image_id").(int),
		&gona.ServerOptions{SSHKeyID: d.Get("ssh_key_id").(int), Password: d.Get("password").(string),
			CloudConfig: d.Get("cloud_config").(string)})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(s.ID))

	return wait4Status(s.ID, "RUNNING", c)
}

func resourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := c.GetServer(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("hostname", server.Name, d, &diags)
	setValue("plan", server.Package, d, &diags)
	setValue("location_id", server.LocationID, d, &diags)
	setValue("image_id", server.OSID, d, &diags)

	return diags
}

func resourceServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteServer(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return wait4Status(id, "TERMINATED", c)
}

func wait4Status(serverId int, status string, client *gona.Client) diag.Diagnostics {
	for i := 0; i < tries; i++ {
		s, err := client.GetServer(serverId)
		if err != nil {
			return diag.FromErr(err)
		} else if s.ServerStatus == status {
			return nil
		}

		time.Sleep(intervalSec * time.Second)
	}

	return diag.Errorf("Timeout of waiting the server to obtain %q status", status)
}
