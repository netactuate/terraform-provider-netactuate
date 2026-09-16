package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVLANs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVLANsRead,
		Schema: map[string]*schema.Schema{
			"vlans": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of account VLANs.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "VLAN ID.",
						},
						"mbid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Account ID for the VLAN.",
						},
						"private": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Private flag as returned by the API. The API uses an integer as a boolean.",
						},
						"allow_sriov": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "SR-IOV flag as returned by the API. The API uses an integer as a boolean.",
						},
						"display_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VLAN display name.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VLAN description.",
						},
						"last_updated": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VLAN last update timestamp.",
						},
						"created": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VLAN creation timestamp.",
						},
						"provisioned_locations": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Locations where the VLAN has provisioning status.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"provisioned": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether the VLAN is provisioned in this location.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location name.",
									},
									"location_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Location ID.",
									},
									"flag": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location flag value.",
									},
									"iata_code": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location IATA code.",
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

func dataSourceVLANsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	vlans, err := c.GetVLANs()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	vlanList := make([]map[string]interface{}, len(vlans))
	for i, vlan := range vlans {
		locations := make([]map[string]interface{}, len(vlan.ProvisionedLocations))
		for j, location := range vlan.ProvisionedLocations {
			locations[j] = map[string]interface{}{
				"provisioned": location.Provisioned,
				"name":        location.Name,
				"location_id": location.LocationID,
				"flag":        location.Flag,
				"iata_code":   location.IATACode,
			}
		}

		vlanList[i] = map[string]interface{}{
			"id":                    vlan.ID,
			"mbid":                  vlan.MBID,
			"private":               vlan.Private,
			"allow_sriov":           vlan.AllowSRIOV,
			"display_name":          vlan.DisplayName,
			"description":           vlan.Description,
			"last_updated":          vlan.LastUpdated,
			"created":               vlan.Created,
			"provisioned_locations": locations,
		}
	}

	setValue("vlans", vlanList, d, &diags)
	d.SetId("vlans")

	return diags
}
