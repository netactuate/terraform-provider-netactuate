package netactuate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// These are the live catalog's display names and API codes. They deliberately
// differ: v0.3.0 accepted TOR/AMS, while v0.4.0 dropped those display codes.
func TestServerLocationDisplayCodeCompatibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/cloud/locations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": []gona.Location{
				{ID: 72, Name: "AMS - Amsterdam, NL", IATACode: "ams2"},
				{ID: 127, Name: "TOR - Toronto, CA", IATACode: "yyz"},
				{ID: 236, Name: "RDU - Raleigh, NC", IATACode: "rdu"},
				{ID: 101, Name: "DFW - Dallas, TX", IATACode: "dfw"},
				{ID: 281, Name: "DFW2 - Dallas, TX", IATACode: "dfw2"},
			},
		})
	}))
	defer server.Close()
	client := gona.NewClientCustom("test-key", server.URL+"/")

	for _, tc := range []struct {
		location string
		wantID   int
	}{
		{"TOR", 127}, {"tor", 127}, {"Tor", 127},
		{"AMS", 72}, {"ams", 72}, {"aMs", 72},
		{"YYZ", 127}, {"yyz", 127}, {"AMS2", 72}, {"ams2", 72},
		{"TOR - Toronto, CA", 127}, {"tor - toronto, ca", 127},
		{"AMS - Amsterdam, NL", 72}, {"ams - amsterdam, nl", 72},
		{"RDU", 236}, {"rdu", 236},
		{"DFW", 101}, {"dfw", 101}, {"DFW - Dallas, TX", 101},
		{"DFW2", 281}, {"dfw2", 281}, {"DFW2 - Dallas, TX", 281},
		{"DF", 0}, {"DFW3", 0},
		{"TOR-not-a-location", 0}, {"TOR wrong name", 0},
		{"AM", 0}, {"AMS3", 0}, {"", 0},
	} {
		t.Run(tc.location, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceServer().Schema, map[string]interface{}{
				"location": tc.location, "image_id": 5794,
			})
			id, imageID, diags := getParams(d, client)
			if tc.wantID == 0 {
				if !diags.HasError() || id != 0 {
					t.Fatalf("invalid location resolved: id=%d, diagnostics=%v", id, diags)
				}
				return
			}
			if diags.HasError() || id != tc.wantID || imageID != 5794 {
				t.Fatalf("getParams(%q) = (%d, %d, %v), want location %d and image 5794", tc.location, id, imageID, diags, tc.wantID)
			}
		})
	}
}

func TestServerDallasLocationsRemainDistinct(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{
		{ID: 101, Name: "DFW - Dallas, TX", IATACode: "dfw"},
		{ID: 281, Name: "DFW2 - Dallas, TX", IATACode: "dfw2"},
	})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })
	for _, tc := range []struct {
		oldName string
		oldID   int
		newName string
	}{
		{"DFW", 101, "DFW2"}, {"dfw", 101, "dfw2"},
		{"DFW2", 281, "DFW"}, {"dfw2", 281, "dfw"},
	} {
		t.Run(tc.oldName+"_to_"+tc.newName, func(t *testing.T) {
			d := rebuildDiffWithRealRawConfig(t, map[string]string{
				"hostname": "dallas.example.test", "plan": "VR1x1x25",
				"location": tc.oldName, "location_id": fmt.Sprint(tc.oldID),
				"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
			}, map[string]interface{}{
				"hostname": "dallas.example.test", "plan": "VR1x1x25",
				"location": tc.newName, "image_id": 5794, "ssh_key_id": "2001",
			})
			if !d.HasChange("location") || !rebuildRequired(d) {
				t.Fatal("different Dallas sites must retain their location change")
			}
		})
	}
}

func TestSharedLocationAliasSchemaDiffs(t *testing.T) {
	versions := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/nke/versions") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{"1.34.0"}})
	}))
	defer versions.Close()
	clients := &ProviderClients{V3: gona.NewV3Client("test-key", versions.URL)}
	setCachedLocationCatalog([]gona.Location{
		{ID: 127, Name: "TOR - Toronto, CA", IATACode: "yyz"},
		{ID: 72, Name: "AMS - Amsterdam, NL", IATACode: "ams2"},
	})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })
	for _, resource := range []struct {
		name       string
		factory    func() *schema.Resource
		config     map[string]interface{}
		locationID int
	}{
		{"vpc", resourceVPC, map[string]interface{}{"label": "alias-test", "description": "alias-test"}, 127},
		{"nke", resourceNKECluster, map[string]interface{}{
			"name": "alias-test", "version": "1.34.0", "minimum_nodes": 1,
			"maximum_nodes": 1, "plan": "VR1x1x25", "contract_id": 440,
		}, 127},
		// Storage has its own catalog: prove alias equivalence without assuming
		// that its numeric IDs equal the cloud catalog's IDs.
		{"bucket", resourceStorageBucket, map[string]interface{}{"label": "alias-test"}, 9127},
		{"object_store", resourceStorageObjectStore, map[string]interface{}{"label": "alias-test"}, 9127},
		{"block_namespace", resourceStorageBlockNamespace, map[string]interface{}{"label": "alias-test"}, 9127},
		{"block_volume", resourceStorageBlockVolume, map[string]interface{}{"label": "alias-test"}, 9127},
	} {
		t.Run(resource.name, func(t *testing.T) {
			for _, oldName := range []string{"TOR - Toronto, CA", "TOR", "YYZ"} {
				for _, input := range []interface{}{resource.locationID, "TOR", "tor", "YYZ", "yyz", "TOR - Toronto, CA", "AMS", "ams2"} {
					t.Run(fmt.Sprintf("%s_to_%v", oldName, input), func(t *testing.T) {
						priorConfig := map[string]interface{}{"location": oldName, "location_id": resource.locationID}
						config := map[string]interface{}{}
						for key, value := range resource.config {
							priorConfig[key] = value
							config[key] = value
						}
						if id, ok := input.(int); ok {
							config["location_id"] = id
						} else {
							config["location"] = input
						}
						r := resource.factory()
						priorData := schema.TestResourceDataRaw(t, r.Schema, priorConfig)
						// A completed Read records empty computed collections too.
						// Leaving them absent would test missing state hydration.
						for key, field := range r.Schema {
							if field.Computed && (field.Type == schema.TypeList || field.Type == schema.TypeSet) {
								if err := priorData.Set(key, []interface{}{}); err != nil {
									t.Fatal(err)
								}
							}
						}
						priorData.SetId("1001")
						prior := priorData.State()
						prior.RawConfig = ctyObjectFromRawConfig(t, config)
						diff, err := r.Diff(context.Background(), prior, terraform.NewResourceConfigRaw(config), clients)
						if err != nil {
							t.Fatal(err)
						}
						d, err := schema.InternalMap(r.Schema).Data(prior, diff)
						if err != nil {
							t.Fatal(err)
						}
						wantChange := input == "AMS" || input == "ams2"
						if wantChange {
							if !d.HasChange("location") || !diff.RequiresNew() {
								t.Fatal("different location must retain a replacement diff")
							}
						} else if !diff.Empty() {
							t.Fatalf("equivalent location declaration produced a diff: %#v", diff.Attributes)
						}
					})
				}
			}
		})
	}
}

func TestServerEquivalentLocationDeclarations(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{
		{ID: 127, Name: "TOR - Toronto, CA", IATACode: "yyz"},
		{ID: 72, Name: "AMS - Amsterdam, NL", IATACode: "ams2"},
	})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })

	for _, oldName := range []string{"TOR - Toronto, CA", "TOR", "YYZ", "yyz"} {
		for _, input := range []interface{}{127, "TOR", "tor", "YYZ", "yyz", "TOR - Toronto, CA"} {
			t.Run(fmt.Sprintf("%s_to_%v", oldName, input), func(t *testing.T) {
				config := map[string]interface{}{
					"hostname": "server.example.test", "plan": "VR1x1x25",
					"image_id": 5794, "ssh_key_id": "2001",
				}
				if id, ok := input.(int); ok {
					config["location_id"] = id
				} else {
					config["location"] = input
				}
				d := rebuildDiffWithRealRawConfig(t, map[string]string{
					"hostname": "server.example.test", "plan": "VR1x1x25",
					"location": oldName, "location_id": "127",
					"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
				}, config)
				if rebuildRequired(d) {
					t.Fatalf("equivalent declaration %q -> %v would rebuild the server", oldName, input)
				}
			})
		}
	}

	for _, name := range []string{"TOR", "YYZ", "AMS", "AMS2"} {
		t.Run("hydrate_"+name, func(t *testing.T) {
			id, full := 127, "TOR - Toronto, CA"
			if name == "AMS" || name == "AMS2" {
				id, full = 72, "AMS - Amsterdam, NL"
			}
			d := schema.TestResourceDataRaw(t, resourceServer().Schema, map[string]interface{}{"location": name})
			var diags diag.Diagnostics
			serverLocationPair.Hydrate(d, id, full, &diags)
			if diags.HasError() || d.Get("location") != name || d.Get("location_id") != id {
				t.Fatalf("Read changed equivalent spelling %q: location=%v location_id=%v diagnostics=%v", name, d.Get("location"), d.Get("location_id"), diags)
			}
		})
	}

	for _, input := range []string{"AMS", "ams2"} {
		t.Run("different_location_"+input, func(t *testing.T) {
			d := rebuildDiffWithRealRawConfig(t, map[string]string{
				"hostname": "server.example.test", "plan": "VR1x1x25",
				"location": "TOR", "location_id": "127",
				"image": "Debian 12 x64 (20241016)", "image_id": "5794", "ssh_key_id": "2001",
			}, map[string]interface{}{
				"hostname": "server.example.test", "plan": "VR1x1x25",
				"location": input, "image_id": 5794, "ssh_key_id": "2001",
			})
			if !d.HasChange("location") {
				t.Fatal("genuinely different location must retain its diff")
			}
		})
	}
}
