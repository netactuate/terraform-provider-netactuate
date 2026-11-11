package netactuate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// fakeInboundFirewallAPI answers GET and PATCH /vpcs/1004, keeping the inbound flags it is sent.
type fakeInboundFirewallAPI struct {
	mu         sync.Mutex
	ipv4, ipv6 bool
	patches    []map[string]interface{}
}

func (f *fakeInboundFirewallAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path != "/vpcs/1004" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		if r.Method == http.MethodPatch {
			b, _ := io.ReadAll(r.Body)
			body := map[string]interface{}{}
			json.Unmarshal(b, &body)
			f.patches = append(f.patches, body)
			var req gona.UpdateVPCRequest
			json.Unmarshal(b, &req)
			if req.Firewalls != nil && req.Firewalls.IPv4 != nil && req.Firewalls.IPv4.Inbound != nil {
				f.ipv4 = *req.Firewalls.IPv4.Inbound
			}
			if req.Firewalls != nil && req.Firewalls.IPv6 != nil && req.Firewalls.IPv6.Inbound != nil {
				f.ipv6 = *req.Firewalls.IPv6.Inbound
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": map[string]interface{}{"vpcId": 1004}})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": map[string]interface{}{
			"vpcId": 1004,
			"firewalls": map[string]interface{}{
				"ipv4": map[string]interface{}{"inbound": map[string]interface{}{"enabled": f.ipv4}, "outbound": map[string]interface{}{"enabled": false}},
				"ipv6": map[string]interface{}{"inbound": map[string]interface{}{"enabled": f.ipv6}, "outbound": map[string]interface{}{"enabled": false}},
			},
		}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Create and delete send only the inbound flags.
func TestVPCInboundFirewallCreateAndDelete(t *testing.T) {
	f := &fakeInboundFirewallAPI{}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := schema.TestResourceDataRaw(t, resourceVPCInboundFirewall().Schema, map[string]interface{}{"vpc_id": 1004, "ipv4": true, "ipv6": false})
	if diags := resourceVPCInboundFirewallPut(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if d.Id() != "1004" || !d.Get("ipv4").(bool) || d.Get("ipv6").(bool) {
		t.Fatalf("state after create: id=%q ipv4=%v ipv6=%v", d.Id(), d.Get("ipv4"), d.Get("ipv6"))
	}
	want := `{"firewalls":{"ipv4":{"inbound":true},"ipv6":{"inbound":false}}}`
	if got, _ := json.Marshal(f.patches[0]); string(got) != want {
		t.Fatalf("create body: %s, want %s", got, want)
	}

	if diags := resourceVPCInboundFirewallDelete(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
	want = `{"firewalls":{"ipv4":{"inbound":false},"ipv6":{"inbound":false}}}`
	if got, _ := json.Marshal(f.patches[1]); string(got) != want {
		t.Fatalf("delete body: %s, want %s", got, want)
	}
	if f.ipv4 || f.ipv6 {
		t.Fatal("delete left the inbound firewall on")
	}
}

// A firewall turned off elsewhere plans to turn it back on.
func TestVPCInboundFirewallReadDetectsDrift(t *testing.T) {
	f := &fakeInboundFirewallAPI{ipv4: false}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := schema.TestResourceDataRaw(t, resourceVPCInboundFirewall().Schema, map[string]interface{}{"vpc_id": 1004, "ipv4": true, "ipv6": false})
	d.SetId("1004")
	if diags := resourceVPCInboundFirewallRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if d.Get("ipv4").(bool) {
		t.Fatal("ipv4: want false from the API")
	}
}

// A VPC whose config leaves the inbound flags out plans no change.
func TestVPCInboundFlagsAdoptAPIValue(t *testing.T) {
	r := resourceVPC()
	for _, k := range []string{"firewall_ipv4_inbound", "firewall_ipv6_inbound"} {
		if s := r.Schema[k]; !s.Optional || !s.Computed || s.Default != nil {
			t.Fatalf("%s: want Optional and Computed with no default", k)
		}
	}
	prior := &terraform.InstanceState{ID: "1004", Attributes: map[string]string{
		"id": "1004", "label": "x", "description": "x", "location": "SJC", "firewall_ipv4_inbound": "true",
	}}
	diff, err := r.SimpleDiff(context.Background(), prior, terraform.NewResourceConfigRaw(map[string]interface{}{
		"label": "x", "description": "x", "location": "SJC",
	}), nil)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if diff != nil {
		if a, ok := diff.Attributes["firewall_ipv4_inbound"]; ok {
			t.Fatalf("firewall_ipv4_inbound diff: %+v", a)
		}
	}
}
