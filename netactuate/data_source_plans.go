package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePlans() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlansRead,
		Schema: map[string]*schema.Schema{
			"plans": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available plans",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"plan_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Plan ID",
						},
						"plan": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan name",
						},
						"ram": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan RAM value as returned by the API",
						},
						"disk": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan disk value as returned by the API",
						},
						"transfer": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan transfer value as returned by the API",
						},
						"price": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan price value as returned by the API",
						},
						"available": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "Plan availability value",
						},
					},
				},
			},
		},
	}
}

func dataSourcePlansRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	plans, err := c.GetPlans()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	plansList := make([]map[string]interface{}, len(plans))
	for i, plan := range plans {
		plansList[i] = map[string]interface{}{
			"plan_id":   plan.ID,
			"plan":      plan.Name,
			"ram":       plan.RAM,
			"disk":      plan.Disk,
			"transfer":  plan.Transfer,
			"price":     plan.Price,
			"available": plan.Available,
		}
	}

	setValue("plans", plansList, d, &diags)
	d.SetId("plans")

	return diags
}
