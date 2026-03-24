package netactuate

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func locationIATA(location string) string {
	parts := strings.Fields(location)
	if len(parts) == 0 {
		return ""
	}
	return strings.ToUpper(parts[0])
}

func sameLocation(a, b string) bool {
	ia, ib := locationIATA(a), locationIATA(b)
	return ia != "" && ib != "" && ia == ib
}

func setLocationPreserveFormat(apiLocation string, d *schema.ResourceData, diags *diag.Diagnostics) {
	current := d.Get("location").(string)
	if current != "" && sameLocation(current, apiLocation) {
		// same location — keep the user's format in state
		return
	}
	setValue("location", apiLocation, d, diags)
}

func suppressLocationDiff(k, old, new string, d *schema.ResourceData) bool {
	if new == "" {
		return true
	}
	return sameLocation(old, new)
}

func suppressStorageLocationDiff(k, old, new string, d *schema.ResourceData) bool {
	if new == "" {
		return true
	}
	if sameLocation(old, new) {
		return true
	}
	if d.Id() != "" {
		oldID, newID := d.GetChange("location_id")
		if oldID.(int) != 0 && oldID == newID {
			return true
		}
	}
	return false
}

func setValue(key string, value interface{}, d *schema.ResourceData, diags *diag.Diagnostics) {
	err := d.Set(key, value)
	if err != nil {
		*diags = append(*diags, diag.Diagnostic{Severity: diag.Error, Summary: err.Error()})
	}
}

func setIntPtr(key string, val *int, d *schema.ResourceData, diags *diag.Diagnostics) {
	if val != nil {
		setValue(key, *val, d, diags)
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

func getStorageLocationID(d *schema.ResourceData, c *gona.V3Client, v2 *gona.Client) (int, *diag.Diagnostic) {
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

	if v2 != nil {
		v2Locations, err := v2.GetLocations()
		if err == nil {
			for _, v2Loc := range v2Locations {
				if strings.EqualFold(v2Loc.IATACode, locationName) {
					for _, loc := range locations {
						if strings.EqualFold(loc.Location.Name, v2Loc.Name) {
							return loc.Location.ID, nil
						}
					}
				}
			}
		}
	}

	return 0, &diag.Errorf("storage location %q not found", locationName)[0]
}

// getPackageID resolves a plan name string to its integer package ID via the V2 GetPlans API.
func getPackageID(planName string, c *gona.Client) (int, *diag.Diagnostic) {
	plans, err := c.GetPlans()
	if err != nil {
		d := diag.FromErr(err)[0]
		return 0, &d
	}

	for _, plan := range plans {
		if strings.EqualFold(plan.Name, planName) {
			return plan.ID, nil
		}
	}

	notFound := diag.Errorf("plan %q not found", planName)[0]
	return 0, &notFound
}
