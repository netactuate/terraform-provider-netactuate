package netactuate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// Both create-time defaults go in one defaults object; nothing is sent when neither is set.
func TestVPCCreateDefaults(t *testing.T) {
	r := resourceVPC()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"label": "x", "description": "x", "location": "SJC"})
	if got := vpcCreateDefaults(d); got != nil {
		t.Fatalf("no defaults set: got %+v", got)
	}
	d = schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"label": "x", "description": "x", "location": "SJC", "enable_default_snat": true, "allow_api_access": true,
	})
	got := vpcCreateDefaults(d)
	if got == nil || got.EnableDefaultSnatRule == nil || !*got.EnableDefaultSnatRule || got.AddEgressIPToAPIACL == nil || !*got.AddEgressIPToAPIACL {
		t.Fatalf("both defaults: got %+v", got)
	}
}

// Read takes allow_api_access from defaults.
func TestVPCReadAllowAPIAccess(t *testing.T) {
	body := `{"code":200,"data":{"vpcId":901,"metadata":{"label":"x","description":"x"},"defaults":{"addEgressIpToApiAcl":true}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := schema.TestResourceDataRaw(t, resourceVPC().Schema, map[string]interface{}{"label": "x", "description": "x", "location": "SJC"})
	d.SetId("901")
	if diags := resourceVPCRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if !d.Get("allow_api_access").(bool) {
		t.Fatal("want true from defaults.addEgressIpToApiAcl")
	}

	body = `{"code":200,"data":{"vpcId":901,"metadata":{"label":"x","description":"x"}}}`
	if diags := resourceVPCRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read without defaults: %v", diags)
	}
	if !d.Get("allow_api_access").(bool) {
		t.Fatal("an API that does not return defaults must leave the value as it was")
	}
}

// State written before the attribute existed plans no change.
func TestVPCAllowAPIAccessSchemaAndUpgrade(t *testing.T) {
	r := resourceVPC()
	s := r.Schema["allow_api_access"]
	if !s.Optional || !s.Computed || !s.ForceNew {
		t.Fatalf("want Optional, Computed and ForceNew: %+v", s)
	}
	prior := &terraform.InstanceState{ID: "901", Attributes: map[string]string{
		"id": "901", "label": "x", "description": "x", "location": "SJC",
	}}
	diff, err := r.SimpleDiff(context.Background(), prior, terraform.NewResourceConfigRaw(map[string]interface{}{
		"label": "x", "description": "x", "location": "SJC",
	}), nil)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if diff != nil {
		if a, ok := diff.Attributes["allow_api_access"]; ok {
			t.Fatalf("allow_api_access diff on old state: %+v", a)
		}
	}
}
