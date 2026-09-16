package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudPools() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudPoolsRead,
		Schema: map[string]*schema.Schema{
			"cloud_pools": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of cloud pools available to the authenticated account.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cloud_pool_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Cloud pool ID from the API id field.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloud pool name.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloud pool description.",
						},
						"required_vcpu": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "CPU model the pool requires, for example EPYC-Milan. Empty when the pool does not constrain it. Despite the name this is not a vcpu count.",
						},
						"hard_capabilities": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Hard capabilities exactly as returned by the API. The API may return a single empty string to mean none.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"soft_capabilities": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Soft capabilities exactly as returned by the API. The API may return a single empty string to mean none.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"private": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Private flag as returned by the API. The API uses an integer as a boolean.",
						},
						"backup_cloud_pool_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Backup cloud pool ID, or unset when the API returns null.",
						},
						"default_ram_price": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Default RAM price as returned by the API. The API returns this value as a string.",
						},
						"default_cpu_price": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Default CPU price as returned by the API. The API returns this value as a string.",
						},
						"default_disk_price": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Default disk price as returned by the API. The API returns this value as a string.",
						},
						"last_updated": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloud pool last update timestamp.",
						},
						"created": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cloud pool creation timestamp.",
						},
						"contract_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Contract ID, or unset when the API returns null.",
						},
					},
				},
			},
		},
	}
}

func dataSourceCloudPoolsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	pools, err := c.GetCloudPools()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	poolList := make([]map[string]interface{}, len(pools))
	for i, pool := range pools {
		poolList[i] = map[string]interface{}{
			"cloud_pool_id":        pool.ID,
			"name":                 pool.Name,
			"description":          pool.Description,
			"required_vcpu":        nullableString(pool.RequiredVCPU),
			"hard_capabilities":    pool.HardCapabilities,
			"soft_capabilities":    pool.SoftCapabilities,
			"private":              pool.Private,
			"backup_cloud_pool_id": nullableInt(pool.BackupCloudPoolID),
			"default_ram_price":    pool.DefaultRAMPrice,
			"default_cpu_price":    pool.DefaultCPUPrice,
			"default_disk_price":   pool.DefaultDiskPrice,
			"last_updated":         pool.LastUpdated,
			"created":              pool.Created,
			"contract_id":          nullableInt(pool.ContractID),
		}
	}

	setValue("cloud_pools", poolList, d, &diags)
	d.SetId("cloud-pools")

	return diags
}
