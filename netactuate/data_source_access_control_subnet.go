package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAccessControlSubnet() *schema.Resource {
	fields := accessControlSubnetFieldsSchema(false)
	fields["access_control_subnet_id"].Computed = false
	fields["access_control_subnet_id"].Required = true

	return &schema.Resource{
		Description: accessControlSubnetWarning,
		ReadContext: dataSourceAccessControlSubnetRead,
		Schema:      fields,
	}
}

func dataSourceAccessControlSubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	id := d.Get("access_control_subnet_id").(string)

	subnet, err := c.GetAccessControlSubnet(id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)
	var diags diag.Diagnostics
	setAccessControlSubnetState(subnet, d, &diags)
	return diags
}
