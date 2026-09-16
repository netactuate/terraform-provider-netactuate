package netactuate

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAccountLimits() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAccountLimitsRead,
		Schema: map[string]*schema.Schema{
			"limits": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of account limits keyed by resource type in the API response, sorted by resource for stable state.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resource type key from the API response.",
						},
						"used": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Current resource usage.",
						},
						"max": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Maximum allowed resource usage.",
						},
						"allowed_plans": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Allowed plan names returned by the API. The API may return null for this field, which is represented as an empty list.",
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "Allowed plan name.",
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceAccountLimitsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	limits, err := c.GetAccountLimits()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	resources := make([]string, 0, len(limits))
	for resource := range limits {
		resources = append(resources, resource)
	}
	sort.Strings(resources)

	limitsList := make([]map[string]interface{}, len(resources))
	for i, resource := range resources {
		limit := limits[resource]
		allowedPlans := limit.AllowedPlans
		if allowedPlans == nil {
			allowedPlans = []string{}
		}
		limitsList[i] = map[string]interface{}{
			"resource":      resource,
			"used":          limit.Used,
			"max":           limit.Max,
			"allowed_plans": allowedPlans,
		}
	}

	setValue("limits", limitsList, d, &diags)
	d.SetId("account-limits")

	return diags
}
