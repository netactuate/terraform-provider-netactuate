package netactuate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// fakeVPCServer serves a single GetVPC response in the V3 API's real
// envelope shape ({code, data}), with
// nameservers CIDR-suffixed ("192.0.2.53/32") exactly as the real API
// returns them even though a bare address is what's configured.
func fakeVPCServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/vpcs/1004", func(w http.ResponseWriter, r *http.Request) {
		vpc := map[string]interface{}{
			"vpcId": 1004,
			"metadata": map[string]interface{}{
				"label":       "example-vpc",
				"description": "Example VPC",
				"status":      "Running",
			},
			"location": map[string]interface{}{"id": 236, "name": "RDU - Raleigh, NC"},
			"bastion":  map[string]interface{}{"enabled": false, "addresses": map[string]interface{}{"ipv4": "192.0.2.12", "ipv6": "2001:db8::12"}},
			"internalNetwork": map[string]interface{}{
				"ipv4": "10.50.0.0/24",
			},
			"dhcp": map[string]interface{}{
				"nameservers": map[string]interface{}{
					"ipv4": []string{"192.0.2.53/32", "192.0.2.54/32"},
					"ipv6": []string{"2001:db8::53/128"},
				},
			},
			"floatingIps": map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": vpc})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestResourceVPCReadHydratesNetworkAndNameservers is the regression for a
// import defect:
// resourceVPCRead never called d.Set for network_ipv4/network_ipv6 (both
// ForceNew) or nameservers_ipv4/nameservers_ipv6 at all. Importing a VPC
// that genuinely had these configured, with a config that matched the live
// values exactly, still showed "1 to add, 1 to destroy" on the very next
// plan -- a forced destroy+recreate of the whole VPC, since the ForceNew
// fields were stuck at their Go zero values instead of the real ones.
func TestResourceVPCReadHydratesNetworkAndNameservers(t *testing.T) {
	srv := fakeVPCServer(t)
	clients := &ProviderClients{V3: gona.NewV3Client("test-key", srv.URL)}

	d := resourceVPC().Data(&terraform.InstanceState{ID: "1004"})
	diags := resourceVPCRead(context.Background(), d, clients)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := d.Get("network_ipv4").(string); got != "10.50.0.0/24" {
		t.Fatalf("expected network_ipv4=10.50.0.0/24, got %q", got)
	}

	wantV4 := []string{"192.0.2.53", "192.0.2.54"}
	gotV4Raw := d.Get("nameservers_ipv4").([]interface{})
	if len(gotV4Raw) != len(wantV4) {
		t.Fatalf("expected %d ipv4 nameservers, got %d: %v", len(wantV4), len(gotV4Raw), gotV4Raw)
	}
	for i, want := range wantV4 {
		if got := gotV4Raw[i].(string); got != want {
			t.Fatalf("nameservers_ipv4[%d]: expected %q (CIDR suffix stripped), got %q", i, want, got)
		}
	}

	gotV6Raw := d.Get("nameservers_ipv6").([]interface{})
	if len(gotV6Raw) != 1 || gotV6Raw[0].(string) != "2001:db8::53" {
		t.Fatalf("expected nameservers_ipv6=[2001:db8::53] (128 suffix stripped), got %v", gotV6Raw)
	}
}

func TestStripHostCIDRSuffixes(t *testing.T) {
	in := []string{"192.0.2.53/32", "192.0.2.54/32", "2001:db8::53/128", "10.0.0.5"}
	want := []string{"192.0.2.53", "192.0.2.54", "2001:db8::53", "10.0.0.5"}
	got := stripHostCIDRSuffixes(in)
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestEnableDefaultSnatNotForceNew confirms the schema fix: this field is
// no longer ForceNew, since the API has no persistent representation of it
// to ever hydrate correctly (UpdateVPCRequest carries no such field) -- the
// old ForceNew forced a full VPC destroy+recreate on the very first apply
// after any import, even with a config that matched the live intent.
func TestEnableDefaultSnatNotForceNew(t *testing.T) {
	sch := resourceVPC().Schema["enable_default_snat"]
	if sch.ForceNew {
		t.Fatal("enable_default_snat must not be ForceNew -- it has no live API counterpart to ever verify, so ForceNew only forces spurious destroy/recreate on import")
	}
	if !sch.Computed {
		t.Fatal("enable_default_snat must be Computed so an unconfigured/unhydrated value doesn't force a diff")
	}
}
