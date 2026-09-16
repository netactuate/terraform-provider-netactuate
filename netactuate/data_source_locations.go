package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLocations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLocationsRead,
		Schema: map[string]*schema.Schema{
			"locations": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available deployment locations",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Location ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Location name",
						},
						"iata_code": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Location IATA code",
						},
						"continent": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Location continent",
						},
						"flag": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Location flag value",
						},
						"disabled": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Location disabled flag as returned by the API, usually 0 or 1",
						},
					},
				},
			},
		},
	}
}

func dataSourceLocationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	locations, err := c.GetLocations()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	locationsList := make([]map[string]interface{}, len(locations))
	for i, location := range locations {
		locationsList[i] = map[string]interface{}{
			"id":        location.ID,
			"name":      location.Name,
			"iata_code": location.IATACode,
			"continent": location.Continent,
			"flag":      location.Flag,
			"disabled":  location.Disabled,
		}
	}

	setValue("locations", locationsList, d, &diags)
	d.SetId("locations")

	return diags
}
