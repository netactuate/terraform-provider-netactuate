package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCFloatingIPs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCFloatingIPsRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VPC ID whose floating IPs will be listed",
			},
			"floating_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of floating IPs attached to the VPC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"floating_ip_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Floating IP ID",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Floating IP address",
						},
						"ip_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "IP version for the floating IP",
						},
						"ptr": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "PTR record value for the floating IP",
						},
						"is_primary": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the floating IP is the VPC primary address",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCFloatingIPsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	vpcID := d.Get("vpc_id").(int)

	floatingIPs, err := c.ListVPCFloatingIPs(vpcID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	floatingIPsList := make([]map[string]interface{}, len(floatingIPs))
	for i, floatingIP := range floatingIPs {
		floatingIPsList[i] = map[string]interface{}{
			"floating_ip_id": floatingIP.FloatingIPID,
			"address":        floatingIP.Address,
			"ip_version":     floatingIP.IPVersion,
			"ptr":            floatingIP.PTR,
			"is_primary":     floatingIP.IsPrimary,
		}
	}

	setValue("floating_ips", floatingIPsList, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-floating-ips", vpcID))

	return diags
}
