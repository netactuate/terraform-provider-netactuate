package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNKEAddons() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKEAddonsRead,
		Schema: map[string]*schema.Schema{
			"addons": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available NKE addons",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"addon_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Addon catalog ID",
						},
						"addon_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Addon type",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Addon version",
						},
						"channel": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Addon channel",
						},
						"display_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Addon display name",
						},
						"min_kubernetes_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Minimum Kubernetes version",
						},
						"max_kubernetes_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Maximum Kubernetes version",
						},
						"is_default": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the addon is installed by default",
						},
						"requires_vpc": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the addon requires a VPC cluster",
						},
					},
				},
			},
		},
	}
}

func dataSourceNKEAddonsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	addons, err := c.ListAddonCatalog()
	if err != nil {
		return diag.FromErr(err)
	}

	addonList := make([]map[string]interface{}, 0, len(addons))
	for _, addon := range addons {
		addonList = append(addonList, map[string]interface{}{
			"addon_id":               addon.AddonID,
			"addon_type":             addon.AddonType,
			"version":                addon.Version,
			"channel":                addon.Channel,
			"display_name":           addon.DisplayName,
			"min_kubernetes_version": addon.MinKubernetesVersion,
			"max_kubernetes_version": addon.MaxKubernetesVersion,
			"is_default":             addon.IsDefault,
			"requires_vpc":           addon.RequiresVpc,
		})
	}

	var diags diag.Diagnostics
	setValue("addons", addonList, d, &diags)
	d.SetId("nke-addons")
	return diags
}
