package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceColocationPackages() *schema.Resource {
	return nonCloudPackagesDataSource(dataSourceColocationPackagesRead, "List of colocation packages for the authenticated account.")
}

func dataSourceColocationPackage() *schema.Resource {
	return nonCloudPackageDataSource(dataSourceColocationPackageRead, "Colocation package ID.")
}

func dataSourceTransitPackages() *schema.Resource {
	return nonCloudPackagesDataSource(dataSourceTransitPackagesRead, "List of transit packages for the authenticated account.")
}

func dataSourceTransitPackage() *schema.Resource {
	return nonCloudPackageDataSource(dataSourceTransitPackageRead, "Transit package ID.")
}

func nonCloudPackagesDataSource(read schema.ReadContextFunc, description string) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			"packages": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: description,
				Elem: &schema.Resource{
					Schema: nonCloudPackageSchema(),
				},
			},
		},
	}
}

func nonCloudPackageDataSource(read schema.ReadContextFunc, idDescription string) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: idDescription,
			},
			"packages": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Package returned by the API.",
				Elem: &schema.Resource{
					Schema: nonCloudPackageSchema(),
				},
			},
		},
	}
}

func nonCloudPackageSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mbpkgid": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Package ID from the API mbpkgid field.",
		},
		"package_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Package status from the API package_status field.",
		},
		"fqdn": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Package FQDN.",
		},
		"billingcycle": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Billing cycle value.",
		},
		"nextduedate": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Next due date string exactly as returned by the API.",
		},
		"amount": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Billing amount as returned by the API. The API returns this value as a string.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Service status when the API returns one, or an empty string when it is absent.",
		},
		"details": nonCloudPackageDetailsSchema(),
	}
}

func nonCloudPackageDetailsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "Customer-service package details. Internal monitoring and out of band management fields are intentionally omitted: pingdom_url, observium_id, observium_id_2, observium_id_3, observium_id_4, observium_id_5, observium_id_6, ipmi_ip, network_info, bgp_info, dc_summary and dc_details.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"dc_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Data center name returned in package details.",
				},
				"iata_code": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "IATA code returned in package details.",
				},
				"bw_commit": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Bandwidth commit returned in package details.",
				},
				"overage_type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Overage type returned in package details.",
				},
				"overage_rate": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Overage rate returned in package details.",
				},
				"agg_bw_mbpkgid": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Aggregate bandwidth package ID returned as a string in package details.",
				},
			},
		},
	}
}

func dataSourceColocationPackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	packages, err := c.GetColocationPackages()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("packages", flattenColocationPackages(packages), d, &diags)
	d.SetId("colocation-packages")

	return diags
}

func dataSourceTransitPackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	packages, err := c.GetTransitPackages()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("packages", flattenTransitPackages(packages), d, &diags)
	d.SetId("transit-packages")

	return diags
}

func dataSourceColocationPackageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgid := d.Get("mbpkgid").(int)

	pkg, err := c.GetColocationPackage(mbpkgid)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("packages", flattenColocationPackages([]gona.ColocationPackage{pkg}), d, &diags)
	d.SetId(fmt.Sprintf("colocation-package-%d", mbpkgid))
	return diags
}

func dataSourceTransitPackageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgid := d.Get("mbpkgid").(int)

	pkg, err := c.GetTransitPackage(mbpkgid)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("packages", flattenTransitPackages([]gona.TransitPackage{pkg}), d, &diags)
	d.SetId(fmt.Sprintf("transit-package-%d", mbpkgid))
	return diags
}

func flattenColocationPackages(packages []gona.ColocationPackage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		result[i] = flattenNonCloudPackage(
			pkg.MBPkgID,
			pkg.PackageStatus,
			pkg.FQDN,
			pkg.BillingCycle,
			pkg.NextDueDate,
			pkg.Amount,
			pkg.Status,
			pkg.Details,
		)
	}
	return result
}

func flattenTransitPackages(packages []gona.TransitPackage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		result[i] = flattenNonCloudPackage(
			pkg.MBPkgID,
			pkg.PackageStatus,
			pkg.FQDN,
			pkg.BillingCycle,
			pkg.NextDueDate,
			pkg.Amount,
			pkg.Status,
			pkg.Details,
		)
	}
	return result
}

func flattenNonCloudPackage(mbpkgid int, packageStatus, fqdn, billingCycle, nextDueDate, amount, status string, details gona.NonCloudPackageDetails) map[string]interface{} {
	return map[string]interface{}{
		"mbpkgid":        mbpkgid,
		"package_status": packageStatus,
		"fqdn":           fqdn,
		"billingcycle":   billingCycle,
		"nextduedate":    nextDueDate,
		"amount":         amount,
		"status":         status,
		"details": []map[string]interface{}{
			{
				"dc_name":        details.DCName,
				"iata_code":      details.IATACode,
				"bw_commit":      details.BWCommit,
				"overage_type":   details.OverageType,
				"overage_rate":   details.OverageRate,
				"agg_bw_mbpkgid": details.AggBWMbPkgID,
			},
		},
	}
}
