package netactuate

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	tries       = 60
	intervalSec = 10
)

var (
	credentialKeys = []string{"password", "ssh_key_id"}
	locationKeys   = []string{"location", "location_id"}
	imageKeys      = []string{"image", "image_id"}

	hostnameRegex = fmt.Sprintf("(%[1]s\\.)*%[1]s$", fmt.Sprintf("(%[1]s|%[1]s%[2]s*%[1]s)", "[a-zA-Z0-9]", "[a-zA-Z0-9\\-]"))
)

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
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					var diags diag.Diagnostics

					match, err := regexp.MatchString(hostnameRegex, i.(string))
					if err != nil {
						diags = diag.FromErr(err)
					} else if !match {
						diags = diag.Errorf("%q is not a valid hostname", i)
					}

					return diags
				},
			},
			"plan": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"package_billing": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"package_billing_contract_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"location": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				ExactlyOneOf: locationKeys,
			},
			"location_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				ExactlyOneOf: locationKeys,
			},
			"image": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				ExactlyOneOf: imageKeys,
			},
			"image_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				ExactlyOneOf: imageKeys,
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
			"user_data_base64": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	locationId, imageId, diags := getParams(d, c)
	if diags != nil {
		return diags
	}

	server := &gona.Server{
		Name:                     d.Get("hostname").(string),
		Plan:                     d.Get("plan").(string),
		LocationID:               locationId,
		OSID:                     imageId,
		PackageBilling:           d.Get("package_billing").(string),
		PackageBillingContractId: d.Get("package_billing_contract_id").(string),
	}

	options := &gona.ServerOptions{
		SSHKeyID:    d.Get("ssh_key_id").(int),
		Password:    d.Get("password").(string),
		CloudConfig: d.Get("cloud_config").(string),
		UserData64:  d.Get("user_data_base64").(string),
		UserData:    d.Get("user_data").(string),
	}

	s, err := c.CreateServer(server, options)
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
	updateValue("location_id", server.LocationID, d, &diags)
	updateValue("location", server.Location, d, &diags)
	updateValue("image_id", server.OSID, d, &diags)
	updateValue("image", server.OS, d, &diags)

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

	// await termination
	ret := wait4Status(id, "TERMINATED", c)
	if ret != nil {
		return ret;
	}

	// send a cancel after the VM has been been deleted
	err = c.CancelServer(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
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

func getParams(d *schema.ResourceData, client *gona.Client) (int, int, diag.Diagnostics) {
	var diags diag.Diagnostics

	locationId, exists := d.GetOk("location_id")
	if !exists {
		location, d := getLocationByName(d.Get("location").(string), client)
		if d != nil {
			diags = append(diags, *d)
		} else {
			locationId = location.ID
		}
	}

	imageId, exists := d.GetOk("image_id")
	if !exists {
		image, d := getImageByName(d.Get("image").(string), client)
		if d != nil {
			diags = append(diags, *d)
		} else {
			imageId = image.ID
		}
	}

	return locationId.(int), imageId.(int), diags
}

func getLocationByName(name string, client *gona.Client) (*gona.Location, *diag.Diagnostic) {
	locations, err := client.GetLocations()
	if err != nil {
		return nil, &diag.FromErr(err)[0]
	}

	for _, location := range locations {
		if location.Name == name {
			return &location, nil
		}
	}

	return nil, &diag.Errorf("Provided location %q doesn't exist", name)[0]
}

func getImageByName(name string, client *gona.Client) (*gona.OS, *diag.Diagnostic) {
	oss, err := client.GetOSs()
	if err != nil {
		return nil, &diag.FromErr(err)[0]
	}

	for _, os := range oss {
		if os.Os == name {
			return &os, nil
		}
	}

	return nil, &diag.Errorf("Provided image %q doesn't exist", name)[0]
}
