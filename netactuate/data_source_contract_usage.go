package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceContractUsage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceContractUsageRead,
		Schema: map[string]*schema.Schema{
			// NOT "id". Terraform reserves the top level id and always treats it as a
			// string, and this data source sets it to a fixed "contract-usage". Declaring a
			// second id of type Int collided with that and panicked the provider on read:
			//   Error reading level set: strconv.ParseInt: parsing "contract-usage"
			// The contract's own numeric id is exposed under its own name.
			"contract_usage_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Contract usage record ID, as returned by the API.",
			},
			"contract_mbpkgid": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Package ID associated with the contract usage record.",
			},
			"parent_contract_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Parent contract ID when the contract usage record has one.",
			},
			"brand": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Brand value returned by the contract usage API.",
			},
			"mb_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Account ID for the contract usage record.",
			},
			"contract_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Contract type returned by the API.",
			},
			"is_free": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Free-contract flag as returned by the API. The API uses an integer as a boolean.",
			},
			"include_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Bandwidth inclusion flag as returned by the API. The API uses an integer as a boolean.",
			},
			"customer_po": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Customer purchase order text returned by the API.",
			},
			"customer_description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Customer description text returned by the API.",
			},
			"po_monthly_limit": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Monthly purchase order limit when one is set.",
			},
			"monthly_discount": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Monthly discount when one is set.",
			},
			"hourly_discount": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Hourly discount when one is set.",
			},
			"max_cpus": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum CPUs when a contract limit is set.",
			},
			"max_ram": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum RAM when a contract limit is set.",
			},
			"max_disk": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum disk when a contract limit is set.",
			},
			"allow_overage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Overage flag as returned by the API. The API uses an integer as a boolean.",
			},
		},
	}
}

func dataSourceContractUsageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	usage, err := c.GetContractUsage()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setIntPtrValue("contract_usage_id", usage.ID, d, &diags)
	setIntPtrValue("contract_mbpkgid", usage.ContractMBPkgID, d, &diags)
	setIntPtrValue("parent_contract_id", usage.ParentContractID, d, &diags)
	setStringPtrValue("brand", usage.Brand, d, &diags)
	setIntPtrValue("mb_id", usage.MBID, d, &diags)
	setStringPtrValue("contract_type", usage.ContractType, d, &diags)
	setIntPtrValue("is_free", usage.IsFree, d, &diags)
	setIntPtrValue("include_bandwidth", usage.IncludeBandwidth, d, &diags)
	setStringPtrValue("customer_po", usage.CustomerPO, d, &diags)
	setStringPtrValue("customer_description", usage.CustomerDescription, d, &diags)
	setIntPtrValue("po_monthly_limit", usage.POMonthlyLimit, d, &diags)
	setIntPtrValue("monthly_discount", usage.MonthlyDiscount, d, &diags)
	setIntPtrValue("hourly_discount", usage.HourlyDiscount, d, &diags)
	setIntPtrValue("max_cpus", usage.MaxCPUs, d, &diags)
	setIntPtrValue("max_ram", usage.MaxRAM, d, &diags)
	setIntPtrValue("max_disk", usage.MaxDisk, d, &diags)
	setIntPtrValue("allow_overage", usage.AllowOverage, d, &diags)

	d.SetId("contract-usage")

	return diags
}

func setIntPtrValue(key string, val *int, d *schema.ResourceData, diags *diag.Diagnostics) {
	if val == nil {
		return
	}
	setValue(key, *val, d, diags)
}

func setStringPtrValue(key string, val *string, d *schema.ResourceData, diags *diag.Diagnostics) {
	if val == nil {
		return
	}
	setValue(key, *val, d, diags)
}
