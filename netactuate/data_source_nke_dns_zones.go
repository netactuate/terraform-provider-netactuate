package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNKEDNSZones() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKEDNSZonesRead,
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The NKE cluster ID",
			},
			"zones": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "DNS zones attached to the NKE cluster",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dns_zone_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "NKE DNS zone ID",
						},
						"cluster_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "NKE cluster ID",
						},
						"zone": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNS zone name",
						},
						"mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNS zone mode",
						},
						"failure_reason": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Failure reason",
						},
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNS zone state",
						},
						"health": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw health object JSON",
						},
						"timestamps": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw timestamps object JSON",
						},
					},
				},
			},
		},
	}
}

func dataSourceNKEDNSZonesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	zones, err := c.ListClusterDNSZones(clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	zoneList := make([]map[string]interface{}, 0, len(zones))
	for _, zone := range zones {
		zoneList = append(zoneList, map[string]interface{}{
			"dns_zone_id":    zone.DNSZoneID,
			"cluster_id":     zone.ClusterID,
			"zone":           zone.Zone,
			"mode":           zone.Mode,
			"failure_reason": zone.FailureReason,
			"state":          zone.State,
			"health":         string(zone.Health),
			"timestamps":     string(zone.Timestamps),
		})
	}

	var diags diag.Diagnostics
	setValue("zones", zoneList, d, &diags)
	d.SetId(fmt.Sprintf("nke-dns-zones-%d", clusterID))
	return diags
}
