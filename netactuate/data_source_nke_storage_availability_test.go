package netactuate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// The fixture is the vapi3 GET /nke/storage-availability response shape
// (vapi3#244, src/api/endpoint/nke/storage-availability/get.js).
func TestDataSourceNKEStorageAvailabilityRead(t *testing.T) {
	var gotPath, gotLocation string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotLocation = r.URL.Query().Get("locationId")
		w.Write([]byte(`{"code":200,"data":{"meta":{"total":2,"offset":0,"limit":100},"data":[
			{"locationId":1,"storageAvailable":true},
			{"locationId":7,"storageAvailable":false}
		]}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL+"/")}

	d := schema.TestResourceDataRaw(t, dataSourceNKEStorageAvailability().Schema, map[string]interface{}{})
	if diags := dataSourceNKEStorageAvailabilityRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if gotPath != "/nke/storage-availability" || gotLocation != "" {
		t.Fatalf("path %q location %q", gotPath, gotLocation)
	}
	if n := d.Get("locations.#").(int); n != 2 {
		t.Fatalf("want 2 locations, got %d", n)
	}
	if d.Get("locations.0.location_id") != 1 || d.Get("locations.0.storage_available") != true ||
		d.Get("locations.1.location_id") != 7 || d.Get("locations.1.storage_available") != false {
		t.Fatalf("locations: %v", d.Get("locations"))
	}

	d2 := schema.TestResourceDataRaw(t, dataSourceNKEStorageAvailability().Schema, map[string]interface{}{"location_id": 7})
	if diags := dataSourceNKEStorageAvailabilityRead(context.Background(), d2, clients); diags.HasError() {
		t.Fatalf("read filtered: %v", diags)
	}
	if gotLocation != "7" {
		t.Fatalf("filter not sent, got %q", gotLocation)
	}
}
