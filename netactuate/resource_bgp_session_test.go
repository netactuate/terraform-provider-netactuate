package netactuate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// fakeBGPServer mimics the three chained live endpoints GetBGPSessions
// actually calls (confirmed by reading gona/bgp.go): bgp/bgpsessions (list
// all), cloud/networkips/{mbpkgid} (this package's owned IPs, used to
// filter the list down), and bgp/bgpsession/{id} (per-session detail) for
// each match. sessions is the fixture list to serve.
// envelope wraps a payload in the real API's response shape
// ({"result":"success","message":"","code":200,"data":...}), which
// gona.Client.do() always expects regardless of endpoint.
func envelope(data interface{}) map[string]interface{} {
	return map[string]interface{}{"result": "success", "message": "", "code": 200, "data": data}
}

func fakeBGPServer(t *testing.T, sessions []map[string]interface{}) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/bgp/bgpsessions", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope(sessions))
	})
	mux.HandleFunc("/cloud/networkips/1001", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{
			"IPv4": []map[string]interface{}{{"ip": "192.0.2.10"}},
			"IPv6": []map[string]interface{}{{"ip": "2001:db8::10"}},
		}))
	})
	mux.HandleFunc("/bgp/bgpsession/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/bgp/bgpsession/"):]
		for _, s := range sessions {
			intID, _ := s["id"].(int)
			if strconv.Itoa(intID) == id {
				json.NewEncoder(w).Encode(envelope(s))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func bgpSessionFixture(id int, customerIP, providerIPType string) map[string]interface{} {
	return map[string]interface{}{
		"id": id, "customer_peer_ip": customerIP, "group_id": 3001,
		"provider_ip_type": providerIPType,
	}
}

// TestResourceBGPSessionReadRecoversFieldsAfterImport is the regression for
// a confirmed import gap: resourceBGPSessionRead used to only ever call
// d.Set("mbpkgid", ...) -- group_id/ipv6/redundant were never set at all,
// so a real import left them null, and since all three are ForceNew, a
// subsequent plan with the CORRECT config values still forced a
// destroy+recreate (null -> any real value is a "change" on a ForceNew
// field). Fixtures use synthetic resource identifiers and documentation IPs.
//
// All three are recoverable from the session list itself: group_id is
// carried directly on every session; ipv6 is true iff any session is on
// the IPv6 address family; redundant is inferred from session count --
// this platform creates two sessions per enabled address family when
// redundant is requested (confirmed directly by the user, who has
// first-hand knowledge of this platform behavior), so a per-family count
// > 1 means redundant was in effect.
func TestResourceBGPSessionReadRecoversFieldsAfterImport(t *testing.T) {
	cases := []struct {
		name          string
		sessions      []map[string]interface{}
		wantIPv6      bool
		wantRedundant bool
	}{
		{
			name:          "single IPv4 session, not redundant",
			sessions:      []map[string]interface{}{bgpSessionFixture(4003, "192.0.2.10", "ipv4")},
			wantIPv6:      false,
			wantRedundant: false,
		},
		{
			name: "IPv4+IPv6 pair, not redundant",
			sessions: []map[string]interface{}{
				bgpSessionFixture(4001, "192.0.2.10", "ipv4"),
				bgpSessionFixture(4002, "2001:db8::10", "ipv6"),
			},
			wantIPv6:      true,
			wantRedundant: false,
		},
		{
			name: "redundant IPv4 only: two IPv4 sessions",
			sessions: []map[string]interface{}{
				bgpSessionFixture(4004, "192.0.2.10", "ipv4"),
				bgpSessionFixture(4005, "192.0.2.10", "ipv4"),
			},
			wantIPv6:      false,
			wantRedundant: true,
		},
		{
			name: "redundant IPv4+IPv6: four sessions total",
			sessions: []map[string]interface{}{
				bgpSessionFixture(4006, "192.0.2.10", "ipv4"),
				bgpSessionFixture(4007, "192.0.2.10", "ipv4"),
				bgpSessionFixture(4008, "2001:db8::10", "ipv6"),
				bgpSessionFixture(4009, "2001:db8::10", "ipv6"),
			},
			wantIPv6:      true,
			wantRedundant: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := fakeBGPServer(t, tc.sessions)
			clients := &ProviderClients{V2: gona.NewClientCustom("test-key", srv.URL+"/")}
			d := resourceBGPSessions().Data(&terraform.InstanceState{ID: "1001"})

			diags := resourceBGPSessionRead(nil, d, clients)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got := d.Get("mbpkgid").(int); got != 1001 {
				t.Fatalf("expected mbpkgid 1001, got %d", got)
			}
			if got := d.Get("group_id").(int); got != 3001 {
				t.Fatalf("expected group_id 3001, got %d", got)
			}
			if got := d.Get("ipv6").(bool); got != tc.wantIPv6 {
				t.Fatalf("expected ipv6=%v, got %v", tc.wantIPv6, got)
			}
			if got := d.Get("redundant").(bool); got != tc.wantRedundant {
				t.Fatalf("expected redundant=%v, got %v", tc.wantRedundant, got)
			}
		})
	}
}
