package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStorageLocations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageLocationsRead,
		Schema: map[string]*schema.Schema{
			"locations": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available storage locations",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The location ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The location name",
						},
					},
				},
			},
		},
	}
}

func dataSourceStorageLocationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	locations, err := c.ListStorageLocations()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	locationsList := make([]map[string]interface{}, len(locations))
	for i, loc := range locations {
		locationsList[i] = map[string]interface{}{
			"id":   loc.Location.ID,
			"name": loc.Location.Name,
		}
	}

	setValue("locations", locationsList, d, &diags)
	d.SetId("storage-locations")

	return diags
}
