package netactuate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func TestNKELBServicesDataSourceRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nke/clusters/512/lb-services" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"code":200,"data":{"data":[
		  {"lbServiceId": 7, "clusterId": 512, "kind": "service", "namespace": "default", "name": "web-lb",
		   "protocol": "tcp", "ports": [{"port": 80, "nodePort": 31080}], "floatingIpId": 901,
		   "sourceRanges": null, "tlsMode": null, "hostnames": [], "vip": "198.51.100.20",
		   "state": "ready", "failureReason": null, "health": {"lastHeartbeatOn": "2026-11-10 12:00:00"},
		   "timestamps": {"createdOn": "2026-11-10 12:00:01"}, "firewallAllowList": []}
		],"meta":{"total":1}}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := schema.TestResourceDataRaw(t, dataSourceNKELBServices().Schema, map[string]interface{}{"cluster_id": 512})
	if diags := dataSourceNKELBServicesRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	svcs := d.Get("services").([]interface{})
	if len(svcs) != 1 {
		t.Fatalf("services: %v", svcs)
	}
	s := svcs[0].(map[string]interface{})
	if s["vip"] != "198.51.100.20" || s["name"] != "web-lb" || s["kind"] != "service" || s["state"] != "ready" {
		t.Fatalf("service: %v", s)
	}
	if s["ports"] != `[{"port":80,"nodePort":31080}]` || s["hostnames"] != "" {
		t.Fatalf("ports/hostnames: %q %q", s["ports"], s["hostnames"])
	}
	if d.Id() != "nke-lb-services-512" {
		t.Fatalf("id: %q", d.Id())
	}
}

func TestNKELBServicesDataSourceNullColumns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{"data":[
		  {"lbServiceId": 8, "clusterId": 512, "kind": "ingress", "namespace": "default", "name": "web",
		   "ports": null, "sourceRanges": null, "hostnames": ["a.example"], "vip": null, "state": "requested"}
		],"meta":{"total":1}}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := schema.TestResourceDataRaw(t, dataSourceNKELBServices().Schema, map[string]interface{}{"cluster_id": 512})
	if diags := dataSourceNKELBServicesRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	s := d.Get("services").([]interface{})[0].(map[string]interface{})
	if s["ports"] != "" || s["source_ranges"] != "" || s["vip"] != "" {
		t.Fatalf("null columns must read as empty strings, got ports=%q source_ranges=%q vip=%q", s["ports"], s["source_ranges"], s["vip"])
	}
	if s["hostnames"] != `["a.example"]` {
		t.Fatalf("hostnames: %q", s["hostnames"])
	}
}
