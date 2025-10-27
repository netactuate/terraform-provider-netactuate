package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceBGPSessions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBGPSessionCreate,
		ReadContext:   resourceBGPSessionRead,
		DeleteContext: resourceBGPSessionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"group_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"ipv6": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Default:  true,
				Optional: true,
			},
			"redundant": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func resourceBGPSessionCreate(_ context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	c := m.(*gona.Client)

	_, err := c.CreateBGPSessions(d.Get("mbpkgid").(int), d.Get("group_id").(int), d.Get("ipv6").(bool),
		d.Get("redundant").(bool))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(d.Get("mbpkgid").(int)))

	return nil
}

func resourceBGPSessionRead(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	// Do nothing
	return nil
}

func resourceBGPSessionDelete(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	// Do nothing
	return nil
}
