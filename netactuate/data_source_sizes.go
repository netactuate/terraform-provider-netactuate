package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSizes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSizesRead,
		Schema: map[string]*schema.Schema{
			"sizes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available cloud sizes. This is a superset of netactuate_plans with cpu and port values.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"plan_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Plan ID.",
						},
						"plan": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan name.",
						},
						"ram": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan RAM value as returned by the API.",
						},
						"disk": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan disk value as returned by the API.",
						},
						"transfer": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan transfer value as returned by the API.",
						},
						"price": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan price value as returned by the API.",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Plan CPU count.",
						},
						"port": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Plan port value as returned by the API.",
						},
						"available": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "Plan availability value.",
						},
					},
				},
			},
		},
	}
}

func dataSourceSizesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	sizes, err := c.GetSizes()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	sizesList := make([]map[string]interface{}, len(sizes))
	for i, size := range sizes {
		sizesList[i] = map[string]interface{}{
			"plan_id":   size.PlanID,
			"plan":      size.Plan,
			"ram":       size.RAM,
			"disk":      size.Disk,
			"transfer":  size.Transfer,
			"price":     size.Price,
			"cpu":       size.CPU,
			"port":      size.Port,
			"available": size.Available,
		}
	}

	setValue("sizes", sizesList, d, &diags)
	d.SetId("sizes")

	return diags
}
