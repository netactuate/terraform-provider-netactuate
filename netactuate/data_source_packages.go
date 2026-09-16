package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePackages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePackagesRead,
		Schema: map[string]*schema.Schema{
			"packages": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of account packages",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mbpkgid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package ID",
						},
						"package_status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Package status",
						},
						"locked": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package locked value as returned by the API",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Package plan name",
						},
						"installed": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package installed flag",
						},
					},
				},
			},
		},
	}
}

func dataSourcePackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	packages, err := c.GetPackages()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	packagesList := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		// ID, Locked and Installed are json.Number, because the API sends them unquoted here
		// and this SDK cannot assume that holds everywhere. A value that will not parse is
		// reported rather than silently becoming zero, since a package id of 0 addresses
		// nothing and would be worse than an error.
		id, err := pkg.ID.Int64()
		if err != nil {
			return diag.Errorf("package %q has an id that is not a number: %v", pkg.PlanName, err)
		}
		locked, err := pkg.Locked.Int64()
		if err != nil {
			return diag.Errorf("package %s has a locked flag that is not a number: %v", pkg.ID, err)
		}
		installed, err := pkg.Installed.Int64()
		if err != nil {
			return diag.Errorf("package %s has an installed flag that is not a number: %v", pkg.ID, err)
		}
		packagesList[i] = map[string]interface{}{
			"mbpkgid":        int(id),
			"package_status": pkg.Status,
			"locked":         int(locked),
			"name":           pkg.PlanName,
			"installed":      int(installed),
		}
	}

	setValue("packages", packagesList, d, &diags)
	d.SetId("packages")

	return diags
}
