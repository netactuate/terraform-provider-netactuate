package netactuate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// fakeLBAddonAPI is a stateful fake of the NKE add-on endpoints for one cluster (512). GET
// answers from states, one entry per call, repeating the last; health conditions carry a
// boolean status, as the nke-addons operator reports them.
type fakeLBAddonAPI struct {
	mu                   sync.Mutex
	states               []string // GET answers in order; "gone" = 404
	failureReason        string
	createRefusals       int // "still being removed" 400s before a create succeeds
	posts, deletes, gets int
	lastBody             map[string]interface{}
}

func (f *fakeLBAddonAPI) addonBody(state string) map[string]interface{} {
	return map[string]interface{}{
		"id": 81, "addonId": 12, "clusterId": 512, "addonType": "netactuate-cloud-lb",
		"version": "v0.3.0", "channel": "stable", "displayName": "Cloud Load Balancer",
		"state": state, "updateAvailable": false,
		"health": map[string]interface{}{"conditions": []map[string]interface{}{
			{"type": "OperatorReady", "status": state == "ready"},
		}},
		"config":     nil,
		"timestamps": map[string]interface{}{"installedOn": "2026-11-10 12:00:00"},
	}
}

func (f *fakeLBAddonAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, code int, data interface{}) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": code, "data": data})
	}
	mux.HandleFunc("/nke/clusters/512/addons", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method != http.MethodPost {
			t.Errorf("unexpected %s on the addons collection", r.Method)
			return
		}
		f.posts++
		b, _ := io.ReadAll(r.Body)
		f.lastBody = map[string]interface{}{}
		json.Unmarshal(b, &f.lastBody)
		if f.createRefusals > 0 {
			f.createRefusals--
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code":400,"data":{"message":"The previous instance of this add-on is still being removed. Retry shortly."}}`))
			return
		}
		write(w, 200, f.addonBody("requested"))
	})
	mux.HandleFunc("/nke/clusters/512/addons/netactuate-cloud-lb", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			state := "gone"
			if len(f.states) > 0 {
				state = f.states[0]
				if len(f.states) > 1 {
					f.states = f.states[1:]
				}
			}
			f.gets++
			if state == "gone" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"code":404,"data":{"message":"Not found"}}`))
				return
			}
			write(w, 200, f.addonBody(state))
		case http.MethodDelete:
			f.deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s on the addon", r.Method)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func fastLBPoll(t *testing.T) {
	t.Helper()
	old := nkeVPCLBAddonRetryInterval
	nkeVPCLBAddonRetryInterval = time.Millisecond
	t.Cleanup(func() { nkeVPCLBAddonRetryInterval = old })
}

func lbAddonData(t *testing.T, raw map[string]interface{}) *schema.ResourceData {
	t.Helper()
	return schema.TestResourceDataRaw(t, resourceNKEVPCLBAddon().Schema, raw)
}

func TestVPCLBAddonCreate(t *testing.T) {
	fastLBPoll(t)
	f := &fakeLBAddonAPI{states: []string{"requested", "ready"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	if diags := resourceNKEVPCLBAddonCreate(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if d.Id() != "512/netactuate-cloud-lb" {
		t.Fatalf("id: %q", d.Id())
	}
	if d.Get("state") != "requested" || d.Get("version") != "v0.3.0" || d.Get("channel") != "stable" {
		t.Fatalf("state not read back: state=%v version=%v", d.Get("state"), d.Get("version"))
	}
	if f.posts != 1 || f.gets != 1 {
		t.Fatalf("posts=%d gets=%d; want 1 post and 1 read, no polling", f.posts, f.gets)
	}
	if f.lastBody["addonType"] != "netactuate-cloud-lb" {
		t.Fatalf("create body: %v", f.lastBody)
	}
	if _, ok := f.lastBody["config"]; ok {
		t.Fatalf("create must not send config: %v", f.lastBody)
	}
	if _, ok := f.lastBody["version"]; ok {
		t.Fatalf("create must not send a version: %v", f.lastBody)
	}
}

func TestVPCLBAddonCreateRetriesWhilePreviousInstanceIsRemoved(t *testing.T) {
	fastLBPoll(t)
	f := &fakeLBAddonAPI{createRefusals: 2, states: []string{"ready"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	if diags := resourceNKEVPCLBAddonCreate(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if f.posts != 3 {
		t.Fatalf("posts=%d, want 3 (two refusals, then success)", f.posts)
	}
}

func TestVPCLBAddonCreateFailsFastOnOtherErrors(t *testing.T) {
	fastLBPoll(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":400,"data":{"message":"The netactuate-cloud-lb add-on is only available on an in-VPC cluster."}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	diags := resourceNKEVPCLBAddonCreate(context.Background(), d, clients)
	if !diags.HasError() || !strings.Contains(diags[0].Summary, "in-VPC cluster") {
		t.Fatalf("want the API's refusal, got %v", diags)
	}
	if d.Id() != "" {
		t.Fatalf("id set on a refused create: %q", d.Id())
	}
}

func TestVPCLBAddonReadRemovesMissingAddon(t *testing.T) {
	f := &fakeLBAddonAPI{states: []string{"gone"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := resourceNKEVPCLBAddon().Data(&terraform.InstanceState{ID: "512/netactuate-cloud-lb"})
	if diags := resourceNKEVPCLBAddonRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if d.Id() != "" {
		t.Fatalf("a missing add-on must be removed from state, id=%q", d.Id())
	}
}

func TestVPCLBAddonReadKeepsStateOnOtherErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"data":{"message":"boom"}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := resourceNKEVPCLBAddon().Data(&terraform.InstanceState{ID: "512/netactuate-cloud-lb"})
	if diags := resourceNKEVPCLBAddonRead(context.Background(), d, clients); !diags.HasError() {
		t.Fatal("a 500 must be an error, not a silent removal")
	}
	if d.Id() == "" {
		t.Fatal("a 500 must not remove the add-on from state")
	}
}

func TestVPCLBAddonDelete(t *testing.T) {
	f := &fakeLBAddonAPI{states: []string{"ready"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	d.SetId("512/netactuate-cloud-lb")
	if diags := resourceNKEVPCLBAddonDelete(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
	if f.deletes != 1 || f.gets != 0 || d.Id() != "" {
		t.Fatalf("deletes=%d gets=%d id=%q; want 1 delete and no polling", f.deletes, f.gets, d.Id())
	}
}

func TestVPCLBAddonDeleteOfMissingAddonSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":404,"data":{"message":"Not found"}}`))
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	d.SetId("512/netactuate-cloud-lb")
	if diags := resourceNKEVPCLBAddonDelete(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
	if d.Id() != "" {
		t.Fatal("id not cleared")
	}
}

func TestVPCLBAddonImportID(t *testing.T) {
	d := resourceNKEVPCLBAddon().Data(&terraform.InstanceState{ID: "512/netactuate-cloud-lb"})
	out, err := resourceNKEVPCLBAddonImport(context.Background(), d, nil)
	if err != nil || len(out) != 1 || out[0].Get("cluster_id") != 512 {
		t.Fatalf("import: %v %v", out, err)
	}
	for _, bad := range []string{"512", "512/storage", "x/netactuate-cloud-lb", "512/netactuate-cloud-lb/1"} {
		d := resourceNKEVPCLBAddon().Data(&terraform.InstanceState{ID: bad})
		if _, err := resourceNKEVPCLBAddonImport(context.Background(), d, nil); err == nil {
			t.Fatalf("import %q should fail", bad)
		}
	}
}

// The retry stops at the create timeout.
func TestVPCLBAddonCreateStopsAtTheDeadline(t *testing.T) {
	old := nkeVPCLBAddonRetryInterval
	nkeVPCLBAddonRetryInterval = 20 * time.Millisecond
	t.Cleanup(func() { nkeVPCLBAddonRetryInterval = old })
	f := &fakeLBAddonAPI{createRefusals: 1000, states: []string{"ready"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()
	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	diags := resourceNKEVPCLBAddonCreate(ctx, d, clients)
	if !diags.HasError() || !strings.Contains(diags[0].Summary, "create timeout") || d.Id() != "" {
		t.Fatalf("want a create-timeout error and no state, got id=%q %v", d.Id(), diags)
	}
	f.mu.Lock()
	posts := f.posts
	f.mu.Unlock()
	time.Sleep(3 * nkeVPCLBAddonRetryInterval)
	f.mu.Lock()
	defer f.mu.Unlock()
	if posts < 1 || f.posts != posts {
		t.Fatalf("posts=%d at return, %d later: want retries that stop at the deadline", posts, f.posts)
	}
}

// A cancelled apply sends no create at all.
func TestVPCLBAddonCreateCancelledSendsNothing(t *testing.T) {
	f := &fakeLBAddonAPI{states: []string{"ready"}}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := lbAddonData(t, map[string]interface{}{"cluster_id": 512})
	if diags := resourceNKEVPCLBAddonCreate(ctx, d, clients); !diags.HasError() || d.Id() != "" {
		t.Fatalf("want an error and no state, got id=%q %v", d.Id(), diags)
	}
	if f.posts != 0 {
		t.Fatalf("posts=%d, want 0", f.posts)
	}
}

// The create returns the platform's refusal.
func TestVPCLBAddonPlanDoesNotConsultCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("plan made a request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)
	clients := &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}

	r := resourceNKEVPCLBAddon()
	if _, err := r.SimpleDiff(context.Background(), &terraform.InstanceState{},
		terraform.NewResourceConfigRaw(map[string]interface{}{"cluster_id": 512}), clients); err != nil {
		t.Fatalf("plan: %v", err)
	}
}
