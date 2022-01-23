package netactuate

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceSshKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSshKeyCreate,
		ReadContext:   resourceSshKeyRead,
		UpdateContext: resourceSshKeyUpdate,
		DeleteContext: resourceSshKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"key": {
				Type:     schema.TypeString,
				Required: true,
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

func resourceSshKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.UpdateSSHKey(id, d.Get("name").(string), d.Get("key").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("last_updated", time.Now().Format(time.RFC850))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceSshKeyRead(ctx, d, m)
}

func resourceSshKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteSSHKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
