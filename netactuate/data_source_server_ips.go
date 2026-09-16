package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceServerIPs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerIPsRead,
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Server package ID whose IP addresses will be listed",
			},
			"ipv4": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IPv4 addresses for the server",
				Elem: &schema.Resource{
					Schema: ipDataSourceSchema(),
				},
			},
			"ipv6": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IPv6 addresses for the server",
				Elem: &schema.Resource{
					Schema: ipDataSourceSchema(),
				},
			},
		},
	}
}

func ipDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "IP address ID",
		},
		"primary": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Primary flag as returned by the API",
		},
		"reverse": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Reverse DNS value",
		},
		"ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP address",
		},
		"gateway": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP gateway address",
		},
		"netmask": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP netmask",
		},
		"broadcast": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP broadcast address",
		},
	}
}

func dataSourceServerIPsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbPkgID := d.Get("mbpkgid").(int)

	ips, err := c.GetIPs(mbPkgID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ipv4", flattenIPs(ips.IPv4), d, &diags)
	setValue("ipv6", flattenIPs(ips.IPv6), d, &diags)
	d.SetId(fmt.Sprintf("server-%d-ips", mbPkgID))

	return diags
}

func flattenIPs(ips []gona.IP) []map[string]interface{} {
	out := make([]map[string]interface{}, len(ips))
	for i, ip := range ips {
		out[i] = map[string]interface{}{
			"id":        ip.ID,
			"primary":   ip.Primary,
			"reverse":   ip.Reverse,
			"ip":        ip.IP,
			"gateway":   ip.Gateway,
			"netmask":   ip.Netmask,
			"broadcast": ip.Broadcast,
		}
	}
	return out
}
