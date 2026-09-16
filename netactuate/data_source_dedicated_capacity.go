package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDedicatedCapacity() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedCapacityRead,
		Schema: map[string]*schema.Schema{
			"dedicated_capacity": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of dedicated server capacity rows.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Dedicated device ID.",
						},
						"location_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Location ID.",
						},
						"looking_glass": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Looking glass identifier.",
						},
						"mbpkgid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package ID associated with the dedicated capacity row.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Location name.",
						},
						"nps_enabled": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "NPS enabled flag as returned by the API. The API uses an integer as a boolean.",
						},
						"pub_description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Public description exactly as returned by the API.",
						},
					},
				},
			},
		},
	}
}

func dataSourceDedicatedCapacityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	capacity, err := c.GetDedicatedCapacity()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	capacityList := make([]map[string]interface{}, len(capacity))
	for i, row := range capacity {
		capacityList[i] = map[string]interface{}{
			"device_id":       row.DeviceID,
			"location_id":     row.LocationID,
			"looking_glass":   row.LookingGlass,
			"mbpkgid":         row.MBPkgID,
			"name":            row.Name,
			"nps_enabled":     row.NPSEnabled,
			"pub_description": row.PubDescription,
		}
	}

	setValue("dedicated_capacity", capacityList, d, &diags)
	d.SetId("dedicated-capacity")

	return diags
}
