package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAccessControlSubnets() *schema.Resource {
	return &schema.Resource{
		Description: accessControlSubnetWarning,
		ReadContext: dataSourceAccessControlSubnetsRead,
		Schema: map[string]*schema.Schema{
			"subnets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "User access control subnets configured on the account.",
				Elem: &schema.Resource{
					Schema: accessControlSubnetFieldsSchema(false),
				},
			},
		},
	}
}

func dataSourceAccessControlSubnetsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	subnets, err := c.GetAccessControlSubnets()
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := make([]map[string]interface{}, len(subnets))
	for i, subnet := range subnets {
		flattened[i] = map[string]interface{}{
			"access_control_subnet_id": subnet.ID.String(),
			"label":                    subnet.Label,
			"subnet":                   subnet.Subnet,
		}
	}

	var diags diag.Diagnostics
	setValue("subnets", flattened, d, &diags)
	d.SetId("access-control-subnets")
	return diags
}
