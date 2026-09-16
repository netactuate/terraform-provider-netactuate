package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudCapacity() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudCapacityRead,
		Schema: map[string]*schema.Schema{
			"cloud_pool_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Cloud pool ID to query capacity for.",
			},
			"location_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Location ID to query capacity for.",
			},
			"capacity": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of cloud package capacity rows for the requested cloud pool and location.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"package_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package ID from the API pkg_id field.",
						},
						"package_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Package name from the API pkg_name field.",
						},
						"package_cpu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package CPU count from the API pkg_cpu field.",
						},
						"package_ram": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package RAM in MB from the API pkg_ram field.",
						},
						"package_disk": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package disk size in GB from the API pkg_disk field.",
						},
						"package_net": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package network value from the API pkg_net field.",
						},
						"package_port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package port value from the API pkg_port field.",
						},
						"monthly_price": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "Monthly price as returned by the API. This endpoint returns the value as a number.",
						},
						"available": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Available capacity count.",
						},
					},
				},
			},
		},
	}
}

func dataSourceCloudCapacityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	cloudPoolID := d.Get("cloud_pool_id").(int)
	locationID := d.Get("location_id").(int)

	capacity, err := c.GetCloudCapacity(cloudPoolID, locationID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	capacityList := make([]map[string]interface{}, len(capacity))
	for i, row := range capacity {
		capacityList[i] = map[string]interface{}{
			"package_id":    row.PackageID,
			"package_name":  row.PackageName,
			"package_cpu":   row.PackageCPU,
			"package_ram":   row.PackageRAM,
			"package_disk":  row.PackageDisk,
			"package_net":   row.PackageNet,
			"package_port":  row.PackagePort,
			"monthly_price": row.MonthlyPrice,
			"available":     row.Available,
		}
	}

	setValue("capacity", capacityList, d, &diags)
	d.SetId(fmt.Sprintf("cloud-capacity-%d-%d", cloudPoolID, locationID))

	return diags
}
