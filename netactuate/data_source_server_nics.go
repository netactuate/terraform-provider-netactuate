package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceServerNICs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerNICsRead,
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Server package ID whose network interfaces will be listed.",
			},
			"nics": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of extra server network interfaces. The element shape is intentionally empty because no server on the test account had an extra NIC when this data source was added.",
				Elem: &schema.Resource{
					// Shape unverified: every server checked on the test account returned an empty list.
					Schema: map[string]*schema.Schema{},
				},
			},
		},
	}
}

func dataSourceServerNICsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgID := d.Get("mbpkgid").(int)

	nics, err := c.GetServerNICs(mbpkgID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	nicList := make([]map[string]interface{}, len(nics))
	for i := range nics {
		nicList[i] = map[string]interface{}{}
	}

	setValue("nics", nicList, d, &diags)
	d.SetId(fmt.Sprintf("server-%d-nics", mbpkgID))

	return diags
}
