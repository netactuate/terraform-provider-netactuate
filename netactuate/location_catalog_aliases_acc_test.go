//go:build acctest

package netactuate

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Read the complete live catalog, then exercise resolution and SDK diffs for
// every entry. This test does not create, update, or delete any API resource.
func TestAccNetactuateServerLocationAliases_catalogWide(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("TF_ACC=1 must be set for live catalog tests")
	}
	testAccPreCheck(t)
	client := testAccClients().V2
	locations, err := client.GetLocations()
	if err != nil {
		t.Fatal(err)
	}
	if len(locations) == 0 {
		t.Fatal("live location catalog is empty")
	}
	entries, err := locationCatalog(client)
	if err != nil {
		t.Fatal(err)
	}
	setCachedLocationCatalog(locations)
	t.Cleanup(func() { setCachedLocationCatalog(nil) })
	resolutions, transitions, hydrationChecks, differentLocations := 0, 0, 0, 0

	for index, location := range locations {
		t.Run(strconv.Itoa(location.ID), func(t *testing.T) {
			names := []string{}
			seen := map[string]bool{}
			for _, name := range []string{location.Name, locationIATA(location.Name), location.IATACode} {
				for _, spelling := range []string{name, strings.ToLower(name), strings.ToUpper(name)} {
					if spelling != "" && !seen[spelling] {
						seen[spelling] = true
						names = append(names, spelling)
					}
				}
			}
			inputs := []interface{}{location.ID}
			for _, name := range names {
				inputs = append(inputs, name)
			}
			for _, input := range inputs {
				config := map[string]interface{}{
					"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
					"image_id": 5794, "ssh_key_id": "2001",
				}
				if id, ok := input.(int); ok {
					config["location_id"] = id
				} else {
					config["location"] = input
				}
				d := schema.TestResourceDataRaw(t, resourceServer().Schema, config)
				id, diagnostic := serverLocationPair.Resolve(d, func() ([]CatalogEntry, error) { return entries, nil })
				resolutions++
				if diagnostic != nil || id != location.ID {
					t.Errorf("resolve %v = %d, %v; want %d", input, id, diagnostic, location.ID)
				}
				for _, oldName := range names {
					diff := rebuildDiffWithRealRawConfig(t, map[string]string{
						"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
						"location": oldName, "location_id": strconv.Itoa(location.ID),
						"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
					}, config)
					transitions++
					if rebuildRequired(diff) {
						t.Errorf("equivalent %q -> %v at location %d would rebuild", oldName, input, location.ID)
					}
				}
			}
			for _, name := range names {
				d := schema.TestResourceDataRaw(t, resourceServer().Schema, map[string]interface{}{"location": name})
				var diagnostics diag.Diagnostics
				serverLocationPair.Hydrate(d, location.ID, location.Name, &diagnostics)
				hydrationChecks++
				if diagnostics.HasError() || d.Get("location") != name || d.Get("location_id") != location.ID {
					t.Errorf("Read changed equivalent spelling %q at location %d", name, location.ID)
				}
			}
			if len(locations) > 1 {
				other := locations[(index+1)%len(locations)]
				if location.ID == other.ID {
					t.Fatal("catalog contains duplicate location IDs")
				}
				diff := rebuildDiffWithRealRawConfig(t, map[string]string{
					"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
					"location": location.Name, "location_id": strconv.Itoa(location.ID),
					"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
				}, map[string]interface{}{
					"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
					"location": other.Name, "image_id": 5794, "ssh_key_id": "2001",
				})
				differentLocations++
				if !diff.HasChange("location") {
					t.Errorf("different location %d -> %d was suppressed", location.ID, other.ID)
				}
			}
		})
	}
	t.Logf("live catalog: %d locations, %d resolution checks, %d equivalent declaration diffs, %d hydration checks, %d different-location controls", len(locations), resolutions, transitions, hydrationChecks, differentLocations)
}
