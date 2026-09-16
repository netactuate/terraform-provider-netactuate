package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceDedicatedServerPowerStatus() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDedicatedServerPowerStatusRead,
		ReadContext:   resourceDedicatedServerPowerStatusRead,
		UpdateContext: resourceDedicatedServerPowerStatusRead,
		DeleteContext: resourceDedicatedServerPowerStatusDelete,
		Description:   "Reads dedicated server power status through the API status action. Deleting this resource removes it from Terraform state only.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Dedicated server package ID.",
			},
			"force": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Force flag sent to the status action only when configured.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password sent to the status action only when configured.",
			},
			"status_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compact JSON power status payload returned by the API.",
			},
		},
	}
}

func resourceDedicatedServerPowerStatusRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbPkgID := d.Get("mbpkgid").(int)
	if mbPkgID == 0 && d.Id() != "" {
		parsed, err := strconv.Atoi(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		mbPkgID = parsed
	}
	if mbPkgID == 0 {
		return diag.Errorf("mbpkgid is required")
	}

	status, err := c.GetDedicatedServerPowerStatus(mbPkgID, dedicatedServerActionRequest(d))
	if err != nil {
		return diag.FromErr(err)
	}
	statusJSON, err := rawMapString(status)
	if err != nil {
		return diag.FromErr(fmt.Errorf("encode dedicated server power status: %w", err))
	}

	var diags diag.Diagnostics
	setValue("mbpkgid", mbPkgID, d, &diags)
	setValue("status_json", statusJSON, d, &diags)
	d.SetId(strconv.Itoa(mbPkgID))
	return diags
}

func resourceDedicatedServerPowerStatusDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Dedicated server power status is read only",
		Detail:   "The NetActuate API status action does not delete or change dedicated server power state. Terraform removed status probe " + id + " from state only.",
	}}
}

func dedicatedServerActionRequest(d *schema.ResourceData) *gona.DedicatedServerActionRequest {
	req := &gona.DedicatedServerActionRequest{}
	if v, ok := d.GetOkExists("force"); ok {
		value := v.(bool)
		req.Force = &value
	}
	if v, ok := d.GetOk("password"); ok {
		value := v.(string)
		req.Password = &value
	}
	if req.Force == nil && req.Password == nil {
		return nil
	}
	return req
}
