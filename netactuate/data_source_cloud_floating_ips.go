package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudFloatingIPs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudFloatingIPsRead,
		Schema: map[string]*schema.Schema{
			"floating_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of account-level cloud floating IPv4 addresses.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"floating_ipv4_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Floating IPv4 ID.",
						},
						"assigned_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Floating IPv4 assignment timestamp. The API field is spelled AssignedOn.",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Floating IPv4 address.",
						},
						"vlan_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "VLAN ID associated with the floating IPv4 address.",
						},
						"ptr_domain": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "PTR domain value, or an empty string when the API returns null.",
						},
						"location": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Location associated with the floating IPv4 address.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Location ID.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location name.",
									},
									"flag": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location flag value.",
									},
									"latitude": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location latitude as returned by the API. The API returns this value as a string.",
									},
									"longitude": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location longitude as returned by the API. The API returns this value as a string.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceCloudFloatingIPsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	floatingIPs, err := c.ListCloudFloatingIPv4()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	floatingIPList := make([]map[string]interface{}, len(floatingIPs))
	for i, floatingIP := range floatingIPs {
		ptrDomain := ""
		if floatingIP.PTRDomain != nil {
			ptrDomain = *floatingIP.PTRDomain
		}

		var locations []map[string]interface{}
		if floatingIP.Location != nil {
			locations = []map[string]interface{}{
				{
					"id":        floatingIP.Location.ID,
					"name":      floatingIP.Location.Name,
					"flag":      floatingIP.Location.Flag,
					"latitude":  floatingIP.Location.Latitude,
					"longitude": floatingIP.Location.Longitude,
				},
			}
		}

		floatingIPList[i] = map[string]interface{}{
			"floating_ipv4_id": floatingIP.FloatingIPv4ID,
			"assigned_on":      floatingIP.AssignedOn,
			"address":          floatingIP.Address,
			"vlan_id":          floatingIP.VLANID,
			"ptr_domain":       ptrDomain,
			"location":         locations,
		}
	}

	setValue("floating_ips", floatingIPList, d, &diags)
	d.SetId("cloud-floating-ips-ipv4")

	return diags
}
