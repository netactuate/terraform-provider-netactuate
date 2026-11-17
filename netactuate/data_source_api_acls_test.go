package netactuate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// The data source lists each entry with the resources holding it.
func TestDataSourceAPIACLsRead(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/api-acls" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"result":"success","code":200,"data":[
			{"id":11,"type":4,"start_ip":"192.0.2.10","end_ip":"192.0.2.10","bits":32,"sub_id":0,
			 "created_at":"2026-11-17 10:00:00","description":"VPC 901 egress","editable":true,
			 "resources":[{"resource_type":"vpc","resource_id":901,"resource_label":"web"}]},
			{"id":12,"type":4,"start_ip":"203.0.113.0","end_ip":"203.0.113.255","bits":24,"sub_id":0,
			 "created_at":"2026-11-01 09:00:00","description":null,"editable":true,"resources":[]}
		]}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V2: gona.NewClientCustom("k", srv.URL+"/")}

	d := schema.TestResourceDataRaw(t, dataSourceAPIACLs().Schema, map[string]interface{}{})
	if diags := dataSourceAPIACLsRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if gotQuery != "key=k" {
		t.Fatalf("query: %q", gotQuery)
	}
	if d.Id() != "api-acls" {
		t.Fatalf("id: %q", d.Id())
	}
	if n := d.Get("acls.#").(int); n != 2 {
		t.Fatalf("want 2 entries, got %d", n)
	}
	if d.Get("acls.0.start_ip") != "192.0.2.10" || d.Get("acls.0.bits") != 32 ||
		d.Get("acls.0.resources.0.resource_type") != "vpc" || d.Get("acls.0.resources.0.resource_id") != 901 ||
		d.Get("acls.0.resources.0.resource_label") != "web" {
		t.Fatalf("first entry: %v", d.Get("acls.0"))
	}
	if d.Get("acls.1.resources.#") != 0 || d.Get("acls.1.description") != "" {
		t.Fatalf("second entry: %v", d.Get("acls.1"))
	}
}
