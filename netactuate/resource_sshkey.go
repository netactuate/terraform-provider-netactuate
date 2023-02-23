package netactuate

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceSshKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSshKeyCreate,
		ReadContext:   resourceSshKeyRead,
		DeleteContext: resourceSshKeyDelete,
		UpdateContext: resourceSshKeyUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val any) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"last_updated": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceSshKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	sshKey, err := c.CreateSSHKey(d.Get("name").(string), d.Get("key").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(sshKey.ID))

	return nil
}

func resourceSshKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sshKey, err := c.GetSSHKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("name", sshKey.Name, d, &diags)
	setValue("key", sshKey.Key, d, &diags)

	return diags
}

func resourceSshKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if id == 0 {
		return nil
	}

	err = c.DeleteSSHKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSshKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	// Delete the first Key
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if id == 0 {
		return nil
	}

	err = c.DeleteSSHKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	// Sleep 3 seconds.
	time.Sleep(3 * time.Second)

	// Create the second key
	sshKey, err := c.CreateSSHKey(d.Get("name").(string), d.Get("key").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(sshKey.ID))

	return nil
}
