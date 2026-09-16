package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBillingPackages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBillingPackagesRead,
		Schema: map[string]*schema.Schema{
			"billing_packages": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of billing packages for the authenticated account.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"billing_package_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Billing package ID from the API id field.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Billing package name.",
						},
						"domu_label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DomU label, or an empty string when the API returns null.",
						},
						"packageid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "WHMCS package ID returned by the API packageid field.",
						},
						"domain": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Billing package domain value.",
						},
						"amount": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Billing amount as returned by the API. The API returns this value as a string.",
						},
						"billingcycle": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Billing cycle value.",
						},
						"domainstatus": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Billing package domain status.",
						},
						"nextduedate": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Next due date string exactly as returned by the API. Unset values may be the literal string 0000-00-00.",
						},
						"dedicatedip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Dedicated IP value.",
						},
					},
				},
			},
		},
	}
}

func dataSourceBillingPackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	packages, err := c.GetBillingPackages()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	packageList := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		domuLabel := ""
		if pkg.DomuLabel != nil {
			domuLabel = *pkg.DomuLabel
		}
		packageList[i] = map[string]interface{}{
			"billing_package_id": pkg.ID,
			"name":               pkg.Name,
			"domu_label":         domuLabel,
			"packageid":          pkg.PackageID,
			"domain":             pkg.Domain,
			"amount":             pkg.Amount,
			"billingcycle":       pkg.BillingCycle,
			"domainstatus":       pkg.DomainStatus,
			"nextduedate":        pkg.NextDueDate,
			"dedicatedip":        pkg.DedicatedIP,
		}
	}

	setValue("billing_packages", packageList, d, &diags)
	d.SetId("billing-packages")

	return diags
}
