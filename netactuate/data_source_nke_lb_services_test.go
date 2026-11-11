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
		   "protocol": "tcp", "ports": [{"port": 80, "nodePort": 31080}, {"port": 8080, "nodePort": 31081}],
		   "floatingIpId": 901, "sourceRanges": ["203.0.113.0/24"], "tlsMode": null, "hostnames": [],
		   "vip": "198.51.100.20", "state": "ready", "failureReason": null,
		   "health": {"lastHeartbeatOn": "2026-11-10 12:00:00"},
		   "timestamps": {"requestedOn": "2026-11-10 11:59:58", "installedOn": null, "readyOn": "2026-11-10 12:00:02", "createdOn": "2026-11-10 12:00:01"},
		   "firewallAllowList": []}
		],"meta":{"total":1}}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := schema.TestResourceDataRaw(t, dataSourceNKELBServices().Schema, map[string]interface{}{"cluster_id": 512})
	if diags := dataSourceNKELBServicesRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if n := d.Get("services.#").(int); n != 1 {
		t.Fatalf("services: %d", n)
	}
	if d.Get("services.0.vip") != "198.51.100.20" || d.Get("services.0.name") != "web-lb" ||
		d.Get("services.0.kind") != "service" || d.Get("services.0.state") != "ready" {
		t.Fatalf("service: %v", d.Get("services.0"))
	}
	if d.Get("services.0.ports.#") != 2 || d.Get("services.0.ports.0.port") != 80 || d.Get("services.0.ports.0.node_port") != 31080 ||
		d.Get("services.0.ports.1.port") != 8080 || d.Get("services.0.ports.1.node_port") != 31081 {
		t.Fatalf("ports: %v", d.Get("services.0.ports"))
	}
	if d.Get("services.0.source_ranges.#") != 1 || d.Get("services.0.source_ranges.0") != "203.0.113.0/24" || d.Get("services.0.hostnames.#") != 0 {
		t.Fatalf("source_ranges/hostnames: %v %v", d.Get("services.0.source_ranges"), d.Get("services.0.hostnames"))
	}
	if d.Get("services.0.last_heartbeat_on") != "2026-11-10 12:00:00" || d.Get("services.0.requested_on") != "2026-11-10 11:59:58" ||
		d.Get("services.0.installed_on") != "" || d.Get("services.0.ready_on") != "2026-11-10 12:00:02" || d.Get("services.0.created_on") != "2026-11-10 12:00:01" {
		t.Fatalf("health/timestamps: %v", d.Get("services.0"))
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
	if d.Get("services.0.ports.#") != 0 || d.Get("services.0.source_ranges.#") != 0 || d.Get("services.0.vip") != "" ||
		d.Get("services.0.last_heartbeat_on") != "" || d.Get("services.0.created_on") != "" {
		t.Fatalf("null columns: %v", d.Get("services.0"))
	}
	if d.Get("services.0.hostnames.#") != 1 || d.Get("services.0.hostnames.0") != "a.example" {
		t.Fatalf("hostnames: %v", d.Get("services.0.hostnames"))
	}
}
