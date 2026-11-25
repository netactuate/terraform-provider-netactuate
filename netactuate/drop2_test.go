package netactuate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func drop2Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{"vpc_ip_reservation": resourceVPCIPReservation(), "nke_cluster_api_allow": resourceNKEClusterAPIAllow(), "nke_cluster_api_access": resourceNKEClusterAPIAccess(), "vpc_attachment": resourceVPCAttachment(), "nke_cluster_private_network": resourceNKEClusterPrivateNetwork()}
}
func drop2TestClient(t *testing.T, h http.HandlerFunc) *ProviderClients {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return &ProviderClients{V3: gona.NewV3Client("q31-secret-key", s.URL)}
}
func TestDrop2Schema(t *testing.T) {
	resources := drop2Resources()
	resources["allow_list"] = dataSourceNKEClusterAPIAllowList()
	resources["attachments"] = dataSourceVPCAttachments()
	resources["reservations"] = dataSourceVPCIPReservations()
	for name, r := range resources {
		if err := schema.InternalMap(r.Schema).InternalValidate(nil); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, r := range drop2Resources() {
		if r.Timeouts == nil || r.Timeouts.Create == nil || r.Timeouts.Update == nil || r.Timeouts.Delete == nil {
			t.Errorf("%s missing timeouts", name)
		}
	}
	r := resourceVPCAttachment()
	config := map[string]interface{}{"vpc_id": 1, "type": "customer-router", "accept": []interface{}{}}
	errs := r.Validate(terraform.NewResourceConfigRaw(config))
	if len(errs) == 0 {
		t.Fatal("empty accept accepted")
	}
	for _, key := range []string{"accept", "advertise"} {
		if r.Schema[key].ForceNew {
			t.Errorf("%s requires recreation", key)
		}
	}
}
func TestDrop2Import(t *testing.T) {
	for name, r := range drop2Resources() {
		t.Run(name, func(t *testing.T) {
			valid := "1/2"
			if name == "nke_cluster_api_allow" {
				valid = "1/customer-2"
			}
			if name == "nke_cluster_api_access" || name == "nke_cluster_private_network" {
				valid = "1"
			}
			for _, id := range []string{valid, "", "0", "-1", "a", "1/", "0/2", "1/2/3", "1/a?key=x"} {
				d := schema.TestResourceDataRaw(t, r.Schema, nil)
				d.SetId(id)
				out, err := r.Importer.StateContext(context.Background(), d, nil)
				if (err == nil) != (id == valid) {
					t.Errorf("%q: %v", id, err)
				}
				if err == nil {
					parent := "cluster_id"
					if strings.HasPrefix(name, "vpc_") {
						parent = "vpc_id"
					}
					if len(out) != 1 || d.Get(parent).(int) != 1 || d.Id() != valid {
						t.Errorf("bad imported state: %v", out)
					}
				}
			}
		})
	}
}
func TestDrop2NotFound(t *testing.T) {
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"code":404,"message":"object does not exist"}`)
	})
	for name, r := range drop2Resources() {
		t.Run(name, func(t *testing.T) {
			id := "1/2"
			if name == "nke_cluster_api_access" || name == "nke_cluster_private_network" {
				id = "1"
			}
			d := schema.TestResourceDataRaw(t, r.Schema, nil)
			d.SetId(id)
			if diags := r.ReadContext(context.Background(), d, c); diags.HasError() || d.Id() != "" {
				t.Fatalf("Read: %v ID=%s", diags, d.Id())
			}
			d.SetId(id)
			if diags := r.DeleteContext(context.Background(), d, c); diags.HasError() || d.Id() != "" {
				t.Fatalf("Delete: %v ID=%s", diags, d.Id())
			}
		})
	}
}
func TestDrop2CustomerErrors(t *testing.T) {
	for _, status := range []int{400, 403, 409} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			message := fmt.Sprintf("server text %d: requested operation refused", status)
			c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": status, "message": message})
			})
			for name, r := range drop2Resources() {
				d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"cluster_id": 1, "vpc_id": 1, "description": "test", "cidr": "192.0.2.0/24", "type": "customer-router", "accept": []interface{}{"192.0.2.0/24"}})
				diags := r.CreateContext(context.Background(), d, c)
				text := fmt.Sprint(diags)
				if !diags.HasError() || !strings.Contains(text, message) || strings.Contains(text, "q31-secret-key") {
					t.Errorf("%s: %v", name, diags)
				}
			}
		})
	}
}
func TestDrop2AttachmentPrefixes(t *testing.T) {
	calls := []string{}
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch r.Method {
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/prefixes") {
				fmt.Fprint(w, `{"data":[{"id":7,"direction":"accept","network":"192.0.2.0/24"},{"id":8,"direction":"advertise","network":"10.0.0.0/24"}]}`)
			} else {
				fmt.Fprint(w, `{"data":{"id":2,"type":"customer-router","state":"ready","accept":["198.51.100.0/24"],"advertise":[]}}`)
			}
		case "POST":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(body, map[string]string{"network": "198.51.100.0/24", "direction": "accept"}) {
				t.Errorf("body %v", body)
			}
			fmt.Fprint(w, `{"data":{"prefixId":9}}`)
		case "DELETE":
			fmt.Fprint(w, `{"data":{}}`)
		default:
			t.Error("unexpected request")
		}
	})
	r := resourceVPCAttachment()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1, "type": "customer-router", "accept": []interface{}{"198.51.100.0/24"}})
	d.SetId("1/2")
	if diags := r.UpdateContext(context.Background(), d, c); diags.HasError() {
		t.Fatal(diags)
	}
	want := []string{"GET /vpcs/1/attachments/2/prefixes", "POST /vpcs/1/attachments/2/prefixes", "DELETE /vpcs/1/attachments/2/prefixes/7", "DELETE /vpcs/1/attachments/2/prefixes/8", "GET /vpcs/1/attachments/2", "GET /vpcs/1/attachments/2"}
	if !reflect.DeepEqual(calls, want) || d.Id() != "1/2" {
		t.Fatalf("calls %v ID %s", calls, d.Id())
	}
	state := d.State()
	config := map[string]interface{}{"vpc_id": 1, "type": "customer-router", "accept": []interface{}{"203.0.113.0/24"}}
	diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	if diff.RequiresNew() {
		t.Fatal("prefix change recreates attachment")
	}
}
func TestDrop2Timeouts(t *testing.T) {
	for _, kind := range []string{"access-create", "access-update", "access-delete", "attachment-create", "attachment-update", "attachment-delete", "private-create"} {
		t.Run(kind, func(t *testing.T) {
			c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "/api-access"):
					fmt.Fprint(w, `{"data":{"state":"applying","privateEndpointState":"provisioning"}}`)
				case strings.Contains(r.URL.Path, "/private-network"):
					if r.Method == "POST" {
						fmt.Fprint(w, `{"data":{"clusterId":1}}`)
					} else {
						fmt.Fprint(w, `{"data":{"state":"in_progress"}}`)
					}
				case strings.Contains(r.URL.Path, "/prefixes") && r.Method == "POST":
					fmt.Fprint(w, `{"data":{"prefixId":5}}`)
				case strings.Contains(r.URL.Path, "/prefixes"):
					fmt.Fprint(w, `{"data":[]}`)
				case r.Method == "POST":
					fmt.Fprint(w, `{"data":{"attachmentId":2}}`)
				default:
					fmt.Fprint(w, `{"data":{"id":2,"state":"building"}}`)
				}
			})
			var r *schema.Resource
			switch {
			case strings.HasPrefix(kind, "access"):
				r = resourceNKEClusterAPIAccess()
			case strings.HasPrefix(kind, "attachment"):
				r = resourceVPCAttachment()
			default:
				r = resourceNKEClusterPrivateNetwork()
			}
			tiny := 10 * time.Millisecond
			r.Timeouts = &schema.ResourceTimeout{Create: &tiny, Update: &tiny, Delete: &tiny}
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"cluster_id": 1, "vpc_id": 1, "type": "customer-router", "accept": []interface{}{"192.0.2.0/24"}})
			d.SetId("1")
			if strings.HasPrefix(kind, "attachment") {
				d.SetId("1/2")
			}
			d = r.Data(d.State())
			if d.Timeout(schema.TimeoutCreate) != tiny {
				t.Fatal("test did not load custom timeout")
			}
			start := time.Now()
			var fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics = r.CreateContext
			if strings.HasSuffix(kind, "update") {
				fn = r.UpdateContext
			}
			if strings.HasSuffix(kind, "delete") {
				fn = r.DeleteContext
			}
			diags := fn(context.Background(), d, c)
			if !diags.HasError() || !strings.Contains(fmt.Sprint(diags), "deadline exceeded") {
				t.Fatalf("wait diagnostics: %v", diags)
			}
			if time.Since(start) > time.Second {
				t.Fatal("configured timeout not used")
			}
		})
	}
}
func TestDrop2PrivateNetworkWarnings(t *testing.T) {
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fmt.Fprint(w, `{"data":{"clusterId":1}}`)
		} else {
			fmt.Fprint(w, `{"data":{"clusterId":1,"vpcId":2,"state":"roll_required","nodesRequiringRoll":[3]}}`)
		}
	})
	r := resourceNKEClusterPrivateNetwork()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"cluster_id": 1, "vpc_id": 2})
	diags := r.CreateContext(context.Background(), d, c)
	if diags.HasError() || len(diags) != 1 || d.Get("state") != "roll_required" {
		t.Fatal(diags)
	}
	diags = r.DeleteContext(context.Background(), d, c)
	if diags.HasError() || len(diags) != 1 || !strings.Contains(diags[0].Detail, "no detach") {
		t.Fatal(diags)
	}
}
func TestDrop2SchemaFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/drop2_schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]map[string]frozenNKEAttribute
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	resources := drop2Resources()
	resources["vpc"] = resourceVPC()
	resources["reservations"] = dataSourceVPCIPReservations()
	resources["allow_list"] = dataSourceNKEClusterAPIAllowList()
	resources["attachments"] = dataSourceVPCAttachments()
	for name, fields := range fixture {
		r := resources[name]
		if r == nil {
			t.Fatalf("missing %s", name)
		}
		for key, want := range fields {
			var s *schema.Schema
			fields := r.Schema
			for _, part := range strings.Split(key, ".") {
				s = fields[part]
				if s == nil {
					break
				}
				if child, ok := s.Elem.(*schema.Resource); ok {
					fields = child.Schema
				}
			}
			if s == nil {
				t.Errorf("missing %s.%s", name, key)
				continue
			}
			got := frozenNKEAttribute{Type: s.Type.String(), Required: s.Required, Optional: s.Optional, Computed: s.Computed, ForceNew: s.ForceNew, Default: s.Default}
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(want)
			if string(a) != string(b) {
				t.Errorf("%s.%s: %s want %s", name, key, a, b)
			}
		}
	}
}

func TestDrop2Lifecycle(t *testing.T) {
	for name, r := range drop2Resources() {
		t.Run(name, func(t *testing.T) {
			deleted := false
			patches := []map[string]bool{}
			c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == "DELETE" {
					deleted = true
					fmt.Fprint(w, `{"data":{}}`)
					return
				}
				if deleted {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"code":404,"message":"gone"}`)
					return
				}
				switch name {
				case "vpc_ip_reservation":
					if req.Method == "POST" {
						fmt.Fprint(w, `{"data":{"ipReservationId":2}}`)
					} else {
						fmt.Fprint(w, `{"data":{"reservations":[{"ipReservationId":2,"address":"10.0.0.5","description":"nah-q31-test","reservedOn":"today","inUse":false}]}}`)
					}
				case "nke_cluster_api_allow":
					if req.Method == "POST" {
						fmt.Fprint(w, `{"data":{"allowListEntryId":"customer-2"}}`)
					} else {
						fmt.Fprint(w, `{"data":[{"allowListEntryId":"customer-2","network":"192.0.2.0/24","description":"nah-q31-test"}]}`)
					}
				case "nke_cluster_api_access":
					if req.Method == "PATCH" {
						var p map[string]bool
						if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
							t.Error(err)
						}
						patches = append(patches, p)
					}
					pub, priv := true, false
					if len(patches) > 0 {
						pub = patches[len(patches)-1]["publicEnabled"]
						priv = patches[len(patches)-1]["privateEndpointEnabled"]
					}
					json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"publicEnabled": pub, "privateEndpointEnabled": priv, "state": "applied", "privateEndpointState": "ready", "privateEndpointAddress": "10.0.0.2", "privateDnsName": "api.internal"}})
				case "vpc_attachment":
					if req.Method == "POST" {
						fmt.Fprint(w, `{"data":{"attachmentId":2}}`)
					} else {
						fmt.Fprint(w, `{"data":{"id":2,"type":"customer-router","address":"10.0.0.2","description":"nah-q31-test","state":"ready","accept":["192.0.2.0/24"],"advertise":[]}}`)
					}
				case "nke_cluster_private_network":
					if req.Method == "POST" {
						fmt.Fprint(w, `{"data":{"clusterId":1}}`)
					} else {
						fmt.Fprint(w, `{"data":{"clusterId":1,"vpcId":1,"state":"applied","nodesRequiringRoll":[]}}`)
					}
				}
			})
			config := map[string]interface{}{"cluster_id": 1, "vpc_id": 1, "description": "nah-q31-test", "label": "nah-q31-test", "cidr": "192.0.2.0/24", "type": "customer-router", "accept": []interface{}{"192.0.2.0/24"}, "public_enabled": false, "private_endpoint_enabled": true}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			if diags := r.CreateContext(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			if d.Id() == "" {
				t.Fatal("create lost ID")
			}
			id := d.Id()
			if name == "nke_cluster_api_access" {
				if d.Get("private_dns_name") != "api.internal" {
					t.Fatal("missing private fields")
				}
				d.Set("public_enabled", true)
				d.Set("private_endpoint_enabled", false)
				if diags := r.UpdateContext(context.Background(), d, c); diags.HasError() {
					t.Fatal(diags)
				}
			}
			imported := schema.TestResourceDataRaw(t, r.Schema, nil)
			imported.SetId(id)
			if _, err := r.Importer.StateContext(context.Background(), imported, c); err != nil {
				t.Fatal(err)
			}
			if diags := r.ReadContext(context.Background(), imported, c); diags.HasError() || imported.Id() != id {
				t.Fatal(diags)
			}
			if diags := r.DeleteContext(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			if d.Id() != "" {
				t.Fatal("delete kept ID")
			}
			if name == "nke_cluster_api_access" {
				if len(patches) != 3 || !patches[2]["publicEnabled"] || patches[2]["privateEndpointEnabled"] {
					t.Fatalf("destroy did not reset API access: %v", patches)
				}
			}
		})
	}
}

func TestDrop2LegacyStateNoDiff(t *testing.T) {
	for _, name := range []string{"vpc", "reservations"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/drop2_" + name + "_v0.4.9.json")
			if err != nil {
				t.Fatal(err)
			}
			var f struct {
				ID       string
				Config   map[string]interface{}
				State    map[string]string
				Response json.RawMessage
			}
			if err = json.Unmarshal(raw, &f); err != nil {
				t.Fatal(err)
			}
			c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("method %s", req.Method)
				}
				if name == "reservations" && req.URL.Query().Has("type") {
					t.Error("untyped request gained type")
				}
				fmt.Fprintf(w, `{"data":%s}`, f.Response)
			})
			r := resourceVPC()
			if name == "reservations" {
				r = dataSourceVPCIPReservations()
			}
			state := &terraform.InstanceState{ID: f.ID, Attributes: f.State}
			d := r.Data(state)
			if diags := r.ReadContext(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			if name == "reservations" {
				for _, key := range []string{"gateways", "interfaces", "vms"} {
					if d.Get(key) != f.State[key] {
						t.Errorf("%s bytes changed: %q want %q", key, d.Get(key), f.State[key])
					}
				}
			} else {
				if d.Get("max_floating_ipv4s") != 20 || d.Get("used_floating_ipv4s") != 3 {
					t.Error("missing counts")
				}
			}
			diff, err := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(f.Config), c)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil {
				// Newly computed outputs can be unknown in the SDK's unit diff.
				// They do not propose a change to the configured object.
				for key, a := range diff.Attributes {
					if a.RequiresNew {
						t.Errorf("upgrade replaces on %s: %#v", key, a)
					}
					if !a.NewComputed && a.New != a.Old {
						t.Errorf("upgrade diff %s: %#v", key, a)
					}
				}
				if diff.RequiresNew() {
					t.Fatal("upgrade requires replacement")
				}
			}
		})
	}
}
func TestDrop2DataSources(t *testing.T) {
	for _, kind := range []string{"typed", "allow", "attachments"} {
		t.Run(kind, func(t *testing.T) {
			c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch kind {
				case "typed":
					if r.URL.Query().Get("type") != "lbVip" {
						t.Error("missing type")
					}
					fmt.Fprint(w, `{"data":{"reservations":[{"ipReservationId":2,"address":"10.0.0.5","inUse":true,"description":"vip","reservedOn":"today"}]}}`)
				case "allow":
					fmt.Fprint(w, `{"data":[{"allowListEntryId":"platform-0","network":"192.0.2.0/24","isPlatform":true,"owner":"system"}]}`)
				case "attachments":
					if strings.HasSuffix(r.URL.Path, "/attachments") {
						fmt.Fprint(w, `{"data":[{"id":2}]}`)
					} else {
						fmt.Fprint(w, `{"data":{"id":2,"type":"customer-router","state":"ready","accept":["192.0.2.0/24"],"advertise":[]}}`)
					}
				}
			})
			var r *schema.Resource
			field := "reservations"
			switch kind {
			case "typed":
				r = dataSourceVPCIPReservations()
			case "allow":
				r = dataSourceNKEClusterAPIAllowList()
				field = "entries"
			case "attachments":
				r = dataSourceVPCAttachments()
				field = "attachments"
			}
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1, "cluster_id": 1, "type": "lbVip"})
			if diags := r.ReadContext(context.Background(), d, c); diags.HasError() {
				t.Fatal(diags)
			}
			rows := d.Get(field).([]interface{})
			if len(rows) != 1 {
				t.Fatal(rows)
			}
			row := rows[0].(map[string]interface{})
			switch kind {
			case "typed":
				if row["ip_reservation_id"] != 2 || row["in_use"] != true {
					t.Fatal(row)
				}
			case "allow":
				if row["is_platform"] != true || row["owner"] != "system" {
					t.Fatal(row)
				}
			case "attachments":
				if row["attachment_id"] != 2 || len(row["accept"].(*schema.Set).List()) != 1 {
					t.Fatal(row)
				}
			}
		})
	}
}

func TestDrop2TransportErrors(t *testing.T) {
	err := fmt.Errorf("create request: %w", &url.Error{Op: "Post", URL: "https://example.invalid?api_key=q31-secret-key", Err: errors.New("connection refused")})
	diags := drop2Error(err)
	if !diags.HasError() || strings.Contains(fmt.Sprint(diags), "q31-secret-key") || strings.Contains(fmt.Sprint(diags), "https://") || !strings.Contains(fmt.Sprint(diags), "connection refused") {
		t.Fatal(diags)
	}
}
func TestDrop2AttachmentPrefixLimit(t *testing.T) {
	active := map[int]string{}
	wanted := []string{}
	for i := 0; i < 32; i++ {
		active[i+1] = fmt.Sprintf("192.0.%d.0/24", i)
		wanted = append(wanted, fmt.Sprintf("198.51.%d.0/24", i))
	}
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rows := []map[string]interface{}{}
			for id, n := range active {
				rows = append(rows, map[string]interface{}{"id": id, "network": n, "direction": "accept"})
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"data": rows})
		case "DELETE":
			parts := strings.Split(r.URL.Path, "/")
			id, _ := strconv.Atoi(parts[len(parts)-1])
			if len(active) <= 1 {
				t.Error("removed last accept prefix")
			}
			delete(active, id)
			fmt.Fprint(w, `{"data":{}}`)
		case "POST":
			if len(active) >= 32 {
				t.Error("exceeded API prefix limit")
			}
			var p map[string]string
			json.NewDecoder(r.Body).Decode(&p)
			id := 1000
			for {
				if _, exists := active[id]; !exists {
					break
				}
				id++
			}
			active[id] = p["network"]
			fmt.Fprintf(w, `{"data":{"prefixId":%d}}`, id)
		}
	})
	if err := drop2SyncPrefixes(c.V3, 1, 2, map[string][]string{"accept": wanted}); err != nil {
		t.Fatal(err)
	}
	if len(active) != 32 {
		t.Fatalf("active count %d", len(active))
	}
	got := map[string]bool{}
	for _, n := range active {
		got[n] = true
	}
	for _, n := range wanted {
		if !got[n] {
			t.Errorf("missing %s", n)
		}
	}
}

func TestDrop2ProviderRegistration(t *testing.T) {
	p := Provider()
	for name := range drop2Resources() {
		if p.ResourcesMap["netactuate_"+name] == nil {
			t.Errorf("resource %s not registered", name)
		}
	}
	for _, name := range []string{"netactuate_nke_cluster_api_allow_list", "netactuate_vpc_attachments"} {
		if p.DataSourcesMap[name] == nil {
			t.Errorf("data source %s not registered", name)
		}
	}
	if err := p.InternalValidate(); err != nil {
		t.Fatal(err)
	}
}
