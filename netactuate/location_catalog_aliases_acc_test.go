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
// every entry. This test creates, updates, and deletes nothing.
//
// A spelling that identifies exactly one location (its full name always, plus
// any IATA or display code it does not share) must resolve to that location and
// must not read as a change. A code shared by more than one location must be
// refused rather than resolved to a silently chosen site: the live catalog
// shares IATA "rdu" between "RDU - Raleigh, NC" and "VR Corp - RDU, NC", and
// display code "VR" among several VR Corp entries.
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

	// The single source of truth for which spellings are unambiguous.
	index := locationResolutionIndex(locations)
	uniqueToThis := func(spelling string, id int) bool {
		got, ok := index[strings.ToLower(strings.TrimSpace(spelling))]
		return ok && got == id
	}

	resolved, refused, transitions, hydrationChecks, differentLocations := 0, 0, 0, 0, 0

	for idx, location := range locations {
		t.Run(strconv.Itoa(location.ID), func(t *testing.T) {
			// Case variants of every candidate spelling for this location.
			spellings := []string{}
			seen := map[string]bool{}
			for _, base := range []string{location.Name, locationIATA(location.Name), location.IATACode} {
				for _, variant := range []string{base, strings.ToLower(base), strings.ToUpper(base)} {
					if variant != "" && !seen[variant] {
						seen[variant] = true
						spellings = append(spellings, variant)
					}
				}
			}

			// location_id always resolves to itself.
			idData := schema.TestResourceDataRaw(t, resourceServer().Schema, map[string]interface{}{
				"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
				"image_id": 5794, "ssh_key_id": "2001", "location_id": location.ID,
			})
			if id, d := serverLocationPair.Resolve(idData, func() ([]CatalogEntry, error) { return entries, nil }); d != nil || id != location.ID {
				t.Errorf("resolve location_id %d = %d, %v; want %d", location.ID, id, d, location.ID)
			}

			for _, spelling := range spellings {
				config := map[string]interface{}{
					"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
					"image_id": 5794, "ssh_key_id": "2001", "location": spelling,
				}
				d := schema.TestResourceDataRaw(t, resourceServer().Schema, config)
				id, diagnostic := serverLocationPair.Resolve(d, func() ([]CatalogEntry, error) { return entries, nil })

				if uniqueToThis(spelling, location.ID) {
					resolved++
					if diagnostic != nil || id != location.ID {
						t.Errorf("unique spelling %q resolved to %d, %v; want %d", spelling, id, diagnostic, location.ID)
					}
					// An equivalent, unique spelling must not read as a change.
					for _, oldName := range spellings {
						if !uniqueToThis(oldName, location.ID) {
							continue
						}
						diff := rebuildDiffWithRealRawConfig(t, map[string]string{
							"hostname": "catalog-only.example.invalid", "plan": "VR1x1x25",
							"location": oldName, "location_id": strconv.Itoa(location.ID),
							"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
						}, config)
						transitions++
						if rebuildRequired(diff) {
							t.Errorf("equivalent %q -> %q at location %d would rebuild", oldName, spelling, location.ID)
						}
					}
				} else {
					// A shared code must be refused, never resolved to a guess.
					refused++
					if diagnostic == nil && id == location.ID {
						t.Errorf("ambiguous spelling %q silently resolved to %d; expected refusal", spelling, location.ID)
					}
				}
			}

			// Read must preserve any unique spelling the user wrote.
			for _, spelling := range spellings {
				if !uniqueToThis(spelling, location.ID) {
					continue
				}
				d := schema.TestResourceDataRaw(t, resourceServer().Schema, map[string]interface{}{"location": spelling})
				var diagnostics diag.Diagnostics
				serverLocationPair.Hydrate(d, location.ID, location.Name, &diagnostics)
				hydrationChecks++
				if diagnostics.HasError() || d.Get("location") != spelling || d.Get("location_id") != location.ID {
					t.Errorf("Read changed equivalent spelling %q at location %d", spelling, location.ID)
				}
			}

			// A genuinely different location must still read as a change.
			if len(locations) > 1 {
				other := locations[(idx+1)%len(locations)]
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
	t.Logf("live catalog: %d locations, %d unique resolves, %d ambiguous refusals, %d equivalence diffs, %d hydration checks, %d different-location controls",
		len(locations), resolved, refused, transitions, hydrationChecks, differentLocations)
}
