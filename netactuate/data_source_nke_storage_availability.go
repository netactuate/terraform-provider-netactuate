package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNKEStorageAvailability() *schema.Resource {
	return &schema.Resource{ReadContext: dataSourceNKEStorageAvailabilityRead, Schema: map[string]*schema.Schema{
		"location_id": {Type: schema.TypeInt, Optional: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)), Description: "Optional location ID filter."},
		"locations": {Type: schema.TypeList, Computed: true, Description: "Storage availability at eligible NKE locations.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"location_id":       {Type: schema.TypeInt, Computed: true, Description: "Location ID."},
			"storage_available": {Type: schema.TypeBool, Computed: true, Description: "Whether persistent storage is available."},
		}}},
	}}
}
func dataSourceNKEStorageAvailabilityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("location_id").(int)
	rows, err := m.(*ProviderClients).V3.ListNKEStorageAvailability(id)
	if err != nil {
		return diag.FromErr(err)
	}
	result := make([]interface{}, 0, len(rows))
	for _, r := range rows {
		result = append(result, map[string]interface{}{"location_id": r.LocationID, "storage_available": r.StorageAvailable})
	}
	if err = d.Set("locations", result); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("nke-storage-availability-%d", id))
	return nil
}
