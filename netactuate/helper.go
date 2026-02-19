package netactuate

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func setValue(key string, value interface{}, d *schema.ResourceData, diags *diag.Diagnostics) {
	err := d.Set(key, value)
	if err != nil {
		*diags = append(*diags, diag.Diagnostic{Severity: diag.Error, Summary: err.Error()})
	}
}

func updateValue(key string, value interface{}, d *schema.ResourceData, diags *diag.Diagnostics) {
	_, exists := d.GetOk(key)
	if exists {
		setValue(key, value, d, diags)
	}
}

func getLocationID(d *schema.ResourceData, c *gona.Client) (int, *diag.Diagnostic) {
	if v, ok := d.GetOk("location_id"); ok {
		return v.(int), nil
	}

	locationName := d.Get("location").(string)
	if locationName == "" {
		return 0, &diag.Errorf("Please provide a location or location_id")[0]
	}

	locations, err := c.GetLocations()
	if err != nil {
		return 0, &diag.FromErr(err)[0]
	}

	for _, loc := range locations {
		if strings.EqualFold(loc.Name, locationName) || strings.EqualFold(loc.IATACode, locationName) {
			return loc.ID, nil
		}
	}

	return 0, &diag.Errorf("location %q not found", locationName)[0]
}

func getStorageLocationID(d *schema.ResourceData, c *gona.V3Client) (int, *diag.Diagnostic) {
	if v, ok := d.GetOk("location_id"); ok {
		return v.(int), nil
	}

	locationName := d.Get("location").(string)
	if locationName == "" {
		return 0, &diag.Errorf("Please provide a location or location_id")[0]
	}

	locations, err := c.ListStorageLocations()
	if err != nil {
		return 0, &diag.FromErr(err)[0]
	}

	for _, loc := range locations {
		if strings.EqualFold(loc.Location.Name, locationName) {
			return loc.Location.ID, nil
		}
	}

	return 0, &diag.Errorf("storage location %q not found", locationName)[0]
}
