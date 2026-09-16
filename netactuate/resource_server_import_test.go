package netactuate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// See findings/server_tags_import_gap.md: a bare `terraform import` used to
// leave tags unset regardless of what the server actually carries live,
// because setServerTagsState (Read) only fetches tags when state already
// has a tracked value -- never true immediately after
// schema.ImportStatePassthroughContext's bare-ID import. resourceServerImport
// now does its own unconditional fetch instead of relying on that gate.
//
// tagsByResourceName maps the exact "tags/resource/<name>/id/<id>" segment
// GetResourceTags requests to the tags that endpoint should return -- a
// request for any other resource name gets an empty list, so a test using
// the wrong resource_name fails loudly instead of passing by accident.
func serverImportTestServer(t *testing.T, vpcID *int, tagsByResourceName map[string][]gona.Tag) *ProviderClients {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/cloud/server/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": gona.Server{VpcID: vpcID},
		})
	})
	mux.HandleFunc("/tags/resource/", func(w http.ResponseWriter, r *http.Request) {
		// Real endpoint: GET /tags/resource/{resourceName}/id/{resourceID}.
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tags/resource/"), "/id/")
		if len(parts) != 2 {
			t.Errorf("unexpected tags path: %s", r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": tagsByResourceName[parts[0]]})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &ProviderClients{V2: gona.NewClientCustom("test-key", server.URL+"/")}
}

func TestResourceServerImportRecoversLiveTags(t *testing.T) {
	const id = 582335
	clients := serverImportTestServer(t, nil, map[string][]gona.Tag{
		"virtual-server": {{ID: 512, Name: "prod"}},
	})
	d := resourceServer().Data(&terraform.InstanceState{ID: strconv.Itoa(id)})

	results, err := resourceServerImport(context.Background(), d, clients)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly one result, got %d", len(results))
	}
	if got := results[0].Get("tags").(string); got != "prod" {
		t.Fatalf("expected tags to be recovered from the live API, got %q", got)
	}
}

func TestResourceServerImportUsesVPCResourceNameForVPCMembers(t *testing.T) {
	const id = 582336
	vpcID := 55
	clients := serverImportTestServer(t, &vpcID, map[string][]gona.Tag{
		// Deliberately assigned under the VPC-member resource name only, so
		// a lookup using the wrong resource_name ("virtual-server") gets an
		// empty list from the mock instead -- proving vpc_id really is
		// hydrated before the tag fetch happens, not just coincidentally
		// correct.
		"virtual-server-vpc": {{ID: 513, Name: "vpc-tagged"}},
	})
	d := resourceServer().Data(&terraform.InstanceState{ID: strconv.Itoa(id)})

	results, err := resourceServerImport(context.Background(), d, clients)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := results[0].Get("vpc_id").(int); got != vpcID {
		t.Fatalf("expected vpc_id to be hydrated to %d, got %d", vpcID, got)
	}
	if got := results[0].Get("tags").(string); got != "vpc-tagged" {
		t.Fatalf("expected the VPC-member tag to be recovered, got %q", got)
	}
}

func TestResourceServerImportSetsEmptyStringWhenNoTags(t *testing.T) {
	const id = 582337
	clients := serverImportTestServer(t, nil, nil)
	d := resourceServer().Data(&terraform.InstanceState{ID: strconv.Itoa(id)})

	results, err := resourceServerImport(context.Background(), d, clients)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := results[0].Get("tags").(string); got != "" {
		t.Fatalf("expected empty tags for an untagged server, got %q", got)
	}
}

func TestResourceServerImportRecoversMultipleTagsSorted(t *testing.T) {
	const id = 582338
	clients := serverImportTestServer(t, nil, map[string][]gona.Tag{
		"virtual-server": {{ID: 1, Name: "zeta"}, {ID: 2, Name: "alpha"}},
	})
	d := resourceServer().Data(&terraform.InstanceState{ID: strconv.Itoa(id)})

	results, err := resourceServerImport(context.Background(), d, clients)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := results[0].Get("tags").(string); got != "alpha, zeta" {
		t.Fatalf("expected sorted, comma-joined tag names, got %q", got)
	}
}
