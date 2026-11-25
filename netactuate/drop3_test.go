package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func drop3Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"security_group": resourceSecurityGroup(), "security_group_rule": resourceSecurityGroupRule(), "security_group_member": resourceSecurityGroupMember(), "vpc_security_group_enforcement": resourceVPCSecurityGroupEnforcement(), "vpc_subnet": resourceVPCSubnet(), "vpc_server_attachment": resourceVPCServerAttachment()}
}
func TestDrop3Schema(t *testing.T) {
	resources := drop3Resources()
	resources["security_groups"] = dataSourceSecurityGroups()
	resources["vpc_security_groups"] = dataSourceVPCSecurityGroups()
	resources["vpc_subnets"] = dataSourceVPCSubnets()
	resources["vpc_servers"] = dataSourceVPCServers()
	resources["vpc_attachments"] = dataSourceVPCAttachments()
	for name, r := range resources {
		if e := schema.InternalMap(r.Schema).InternalValidate(nil); e != nil {
			t.Errorf("%s: %v", name, e)
		}
	}
	for name, r := range drop3Resources() {
		if r.Importer == nil || r.Timeouts == nil || r.Timeouts.Create == nil || r.Timeouts.Delete == nil {
			t.Errorf("%s missing importer or timeouts", name)
		}
	}
	for _, r := range []*schema.Resource{resourceSecurityGroupRule(), resourceSecurityGroupMember()} {
		for k, s := range r.Schema {
			if !s.Computed && !s.ForceNew {
				t.Errorf("%s is mutable", k)
			}
		}
	}
	b := resourceVPCAttachment().Schema["bgp"]
	if !b.Optional || b.ForceNew || b.MaxItems != 1 {
		t.Fatal("BGP compatibility")
	}
	fields := b.Elem.(*schema.Resource).Schema
	if !fields["password"].Sensitive {
		t.Fatal("password is not sensitive")
	}
	for k, s := range fields {
		if s.ForceNew {
			t.Errorf("BGP %s recreates", k)
		}
	}
}
func TestDrop3Import(t *testing.T) {
	for name, r := range drop3Resources() {
		t.Run(name, func(t *testing.T) {
			valid := "1/2"
			if name == "security_group" || name == "vpc_security_group_enforcement" {
				valid = "1"
			}
			for _, id := range []string{valid, "", "0", "-1", "a", "1/", "1/0", "1/a", "0/2", "1/2/3"} {
				d := schema.TestResourceDataRaw(t, r.Schema, nil)
				d.SetId(id)
				out, e := r.Importer.StateContext(context.Background(), d, nil)
				if (e == nil) != (id == valid) {
					t.Errorf("%q: %v", id, e)
				}
				if e == nil && (len(out) != 1 || d.Id() != valid) {
					t.Fatal("import state")
				}
				if e == nil {
					parent := "vpc_id"
					if strings.HasPrefix(name, "security_group") {
						parent = "security_group_id"
					}
					if d.Get(parent).(int) != 1 {
						t.Fatal("parent not parsed")
					}
				}

			}
		})
	}
}
func TestDrop3NotFound(t *testing.T) {
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"code":404,"message":"object does not exist"}`)
	})
	for name, r := range drop3Resources() {
		t.Run(name, func(t *testing.T) {
			id := "1/2"
			if name == "security_group" || name == "vpc_security_group_enforcement" {
				id = "1"
			}
			d := schema.TestResourceDataRaw(t, r.Schema, nil)
			d.SetId(id)
			if ds := r.ReadContext(context.Background(), d, c); ds.HasError() || d.Id() != "" {
				t.Fatalf("read: %v %s", ds, d.Id())
			}
			d.SetId(id)
			if ds := r.DeleteContext(context.Background(), d, c); ds.HasError() || d.Id() != "" {
				t.Fatalf("delete: %v %s", ds, d.Id())
			}
		})
	}
}
func TestDrop3CustomerErrors(t *testing.T) {
	for _, status := range []int{400, 403, 409} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			text := fmt.Sprintf("server text %d", status)
			c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"code":%d,"message":%q}`, status, text)
			})
			for name, r := range drop3Resources() {
				d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1, "security_group_id": 1, "mbpkgid": 2, "name": "test", "network": "10.1.0.0/24", "label": "test", "ip_version": 4})
				d.SetId("1/2")
				if name == "security_group" || name == "vpc_security_group_enforcement" {
					d.SetId("1")
				}
				for _, fn := range []func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics{r.ReadContext, r.DeleteContext, r.CreateContext} {
					ds := fn(context.Background(), d, c)
					got := fmt.Sprint(ds)
					if !ds.HasError() || !strings.Contains(got, text) || strings.Contains(got, "q31-secret-key") || strings.Contains(got, "?key=") {
						t.Errorf("%s: %v", name, ds)
					}
				}
			}
		})
	}
	// A 404 is a diagnostic on create, and absence on read or delete.
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"code":404,"message":"server missing text"}`)
	})
	for name, r := range drop3Resources() {
		d := schema.TestResourceDataRaw(t, r.Schema, nil)
		if ds := r.CreateContext(context.Background(), d, c); !ds.HasError() || !strings.Contains(fmt.Sprint(ds), "server missing text") {
			t.Errorf("%s: %v", name, ds)
		}
	}
}
func TestDrop3SubnetTimeouts(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprint(deleted), func(t *testing.T) {
			c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"data":{"subnetId":2,"state":"pending"}}`)
			})
			r := resourceVPCSubnet()
			tiny := 10 * time.Millisecond
			r.Timeouts = &schema.ResourceTimeout{Create: &tiny, Delete: &tiny}
			d := schema.TestResourceDataRaw(t, r.Schema, nil)
			d = r.Data(d.State())
			start := time.Now()
			err := drop3WaitSubnet(context.Background(), d, c.V3, 1, 2, deleted)
			if err == nil || !strings.Contains(err.Error(), "deadline exceeded") || time.Since(start) > time.Second {
				t.Fatalf("timeout: %v", err)
			}
		})
	}
}
func TestDrop3Q31AttachmentNoDiff(t *testing.T) {
	c := drop2TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"id":2,"type":"customer-router","address":"10.0.0.2","accept":["192.0.2.0/24"],"advertise":[],"state":"ready","bgp":null}}`)
	})
	r := resourceVPCAttachment()
	config := map[string]interface{}{"vpc_id": 1, "type": "customer-router", "address": "10.0.0.2", "accept": []interface{}{"192.0.2.0/24"}}
	d := schema.TestResourceDataRaw(t, r.Schema, config)
	d.SetId("1/2")
	state := d.State()
	delete(state.Attributes, "bgp.#")
	d = r.Data(state)
	if ds := r.ReadContext(context.Background(), d, c); ds.HasError() {
		t.Fatal(ds)
	}
	diff, e := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), c)
	if e != nil {
		t.Fatal(e)
	}
	if diff != nil && !diff.Empty() {
		t.Fatalf("Q31 diff: %#v", diff)
	}
}
func TestDrop3BGPInPlace(t *testing.T) {
	r := resourceVPCAttachment()
	base := map[string]interface{}{"vpc_id": 1, "type": "customer-router", "address": "10.0.0.2", "accept": []interface{}{"192.0.2.0/24"}}
	block := func(max int) []interface{} {
		return []interface{}{map[string]interface{}{"peer_asn": 65001, "max_prefix": max, "password": "configured-password"}}
	}
	state := schema.TestResourceDataRaw(t, r.Schema, base)
	state.SetId("1/2")
	for step := 0; step < 3; step++ {
		config := map[string]interface{}{}
		for k, v := range base {
			config[k] = v
		}
		if step < 2 {
			config["bgp"] = block(100 + step)
		}
		calls := []string{}
		c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, req.Method+" "+req.URL.Path)
			if strings.HasSuffix(req.URL.Path, "/bgp") {
				fmt.Fprint(w, `{"data":{"attachmentId":2}}`)
				return
			}
			if step == 2 {
				fmt.Fprint(w, `{"data":{"id":2,"type":"customer-router","address":"10.0.0.2","accept":["192.0.2.0/24"],"state":"ready","bgp":null}}`)
			} else {
				fmt.Fprintf(w, `{"data":{"id":2,"type":"customer-router","address":"10.0.0.2","accept":["192.0.2.0/24"],"state":"ready","bgp":{"peerAsn":65001,"maxPrefix":%d,"hasPassword":true,"password":"response-must-be-ignored"}}}`, 100+step)
			}
		})
		diff, e := r.Diff(context.Background(), state.State(), terraform.NewResourceConfigRaw(config), c)
		if e != nil {
			t.Fatal(e)
		}
		if diff == nil || diff.RequiresNew() {
			t.Fatal("BGP must update")
		}
		next, ds := r.Apply(context.Background(), state.State(), diff, c)
		if ds.HasError() {
			t.Fatal(ds)
		}
		// Apply returns state, so reconstruct the data for the next plan.
		state = r.Data(next)
		want := "PUT /vpcs/1/attachments/2/bgp"
		if step == 2 {
			want = "DELETE /vpcs/1/attachments/2/bgp"
		}
		if len(calls) == 0 || calls[0] != want {
			t.Fatalf("calls %v", calls)
		}
		for _, call := range calls {
			if strings.HasPrefix(call, "POST ") || call == "DELETE /vpcs/1/attachments/2" {
				t.Fatal(calls)
			}
		}
		if state.Id() != "1/2" {
			t.Fatal(state.Id())
		}
		if step < 2 && state.Get("bgp.0.password") != "configured-password" {
			t.Fatal("password overwritten")
		}
	}
}
func TestDrop3DataSources(t *testing.T) {
	for name, r := range map[string]*schema.Resource{"security_groups": dataSourceSecurityGroups(), "vpc_security_groups": dataSourceVPCSecurityGroups(), "vpc_subnets": dataSourceVPCSubnets(), "vpc_servers": dataSourceVPCServers()} {
		t.Run(name, func(t *testing.T) {
			calls := []string{}
			c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, req.URL.Path)
				switch name {
				case "security_groups", "vpc_security_groups":
					fmt.Fprint(w, `{"data":{"data":[{"securityGroupId":2,"name":"default","isDefault":true}]}}`)
				case "vpc_subnets":
					fmt.Fprint(w, `{"data":{"data":[{"subnetId":0,"network":"10.0.0.0/24","isDefault":true}]}}`)
				case "vpc_servers":
					fmt.Fprint(w, `{"data":{"data":[{"mbpkgid":2,"membership":"native"},{"mbpkgid":3,"membership":"attached"}]}}`)
				}
			})
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1})
			if ds := r.ReadContext(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			key := strings.TrimPrefix(name, "vpc_")
			if name == "vpc_subnets" {
				key = "subnets"
			}
			if name == "vpc_servers" {
				key = "servers"
			}
			rows := d.Get(key).([]interface{})
			if len(rows) == 0 {
				t.Fatal("empty result")
			}
			if name == "vpc_servers" && !reflect.DeepEqual([]interface{}{rows[0].(map[string]interface{})["membership"], rows[1].(map[string]interface{})["membership"]}, []interface{}{"native", "attached"}) {
				t.Fatal(rows)
			}
		})
	}
}

func TestDrop3Lifecycle(t *testing.T) {
	for name, r := range drop3Resources() {
		t.Run(name, func(t *testing.T) {
			deleted := false
			renamed := false
			calls := []string{}
			c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, req.Method+" "+req.URL.Path)
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
				if req.Method == "PATCH" {
					renamed = true
					fmt.Fprint(w, `{"data":{}}`)
					return
				}
				if req.Method == "POST" {
					switch name {
					case "security_group":
						fmt.Fprint(w, `{"data":{"securityGroupId":2}}`)
					case "security_group_rule":
						fmt.Fprint(w, `{"data":{"ruleId":2}}`)
					case "security_group_member":
						fmt.Fprint(w, `{"data":{"memberId":2}}`)
					case "vpc_subnet":
						fmt.Fprint(w, `{"data":{"subnetId":2}}`)
					case "vpc_server_attachment":
						fmt.Fprint(w, `{"data":{"mbpkgid":2,"nicId":3,"state":"attached"}}`)
					}
					return
				}
				switch name {
				case "security_group":
					label := "web"
					if renamed {
						label = "renamed"
					}
					fmt.Fprintf(w, `{"data":{"securityGroupId":2,"name":%q,"description":"tier","memberCount":1}}`, label)
				case "security_group_rule":
					fmt.Fprint(w, `{"data":{"data":[{"ruleId":2,"ipVersion":4,"direction":"inbound","protocol":"tcp","portStart":443,"portEnd":443,"network":"192.0.2.0/24"}]}}`)
				case "security_group_member":
					fmt.Fprint(w, `{"data":{"data":[{"memberId":2,"vpcId":1,"mbpkgid":2,"enforcementState":"enforced"}]}}`)
				case "vpc_security_group_enforcement":
					fmt.Fprint(w, `{"data":{"vpcId":1,"enabled":true,"enabledOn":"today"}}`)
				case "vpc_subnet":
					fmt.Fprint(w, `{"data":{"subnetId":2,"network":"10.1.0.0/24","label":"app","state":"ready"}}`)
				case "vpc_server_attachment":
					fmt.Fprint(w, `{"data":{"data":[{"mbpkgid":2,"membership":"attached","subnetId":0,"nicId":3,"state":"attached","address":"10.0.0.4"}]}}`)
				}
			})
			config := map[string]interface{}{"vpc_id": 1, "security_group_id": 1, "mbpkgid": 2, "name": "web", "description": "tier", "ip_version": 4, "protocol": "tcp", "port_start": 443, "port_end": 443, "network": "10.1.0.0/24", "label": "app"}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			if ds := r.CreateContext(context.Background(), d, c); ds.HasError() || d.Id() == "" {
				t.Fatalf("create: %v", ds)
			}
			id := d.Id()
			if ds := r.ReadContext(context.Background(), d, c); ds.HasError() || d.Id() != id {
				t.Fatalf("read: %v", ds)
			}
			if name == "security_group" {
				if ds := r.UpdateContext(context.Background(), d, c); ds.HasError() || d.Get("name") != "renamed" || d.Id() != id {
					t.Fatalf("update: %v", ds)
				}
			}
			if ds := r.DeleteContext(context.Background(), d, c); ds.HasError() || d.Id() != "" {
				t.Fatalf("delete: %v", ds)
			}
			if len(calls) < 3 {
				t.Fatal(calls)
			}
		})
	}
}
func TestDrop3ImmutablePlans(t *testing.T) {
	for name, r := range map[string]*schema.Resource{"rule": resourceSecurityGroupRule(), "member": resourceSecurityGroupMember()} {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{"security_group_id": 1, "vpc_id": 1, "mbpkgid": 2, "ip_version": 4, "network": "192.0.2.0/24"}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			d.SetId("1/2")
			if name == "rule" {
				config["network"] = "198.51.100.0/24"
			} else {
				config["mbpkgid"] = 3
			}
			delete(config, "vpc_id")
			if name == "member" {
				config["vpc_id"] = 1
				delete(config, "ip_version")
				delete(config, "network")
			} else {
				delete(config, "mbpkgid")
			}
			diff, e := r.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
			if e != nil {
				t.Fatal(e)
			}
			if diff == nil || !diff.RequiresNew() {
				t.Fatal("immutable change must recreate")
			}
		})
	}
}

func TestDrop3Validation(t *testing.T) {
	tests := []struct {
		name   string
		r      *schema.Resource
		config map[string]interface{}
	}{
		{"rule missing source", resourceSecurityGroupRule(), map[string]interface{}{"security_group_id": 1, "ip_version": 4}},
		{"rule two sources", resourceSecurityGroupRule(), map[string]interface{}{"security_group_id": 1, "ip_version": 4, "network": "192.0.2.0/24", "source_security_group_id": 2}},
		{"rule version", resourceSecurityGroupRule(), map[string]interface{}{"security_group_id": 1, "ip_version": 5, "network": "192.0.2.0/24"}},
		{"rule direction", resourceSecurityGroupRule(), map[string]interface{}{"security_group_id": 1, "ip_version": 4, "network": "192.0.2.0/24", "direction": "outbound"}},
		{"rule half port", resourceSecurityGroupRule(), map[string]interface{}{"security_group_id": 1, "ip_version": 4, "network": "192.0.2.0/24", "port_start": 443}},
		{"member missing target", resourceSecurityGroupMember(), map[string]interface{}{"security_group_id": 1, "vpc_id": 1}},
		{"member two targets", resourceSecurityGroupMember(), map[string]interface{}{"security_group_id": 1, "vpc_id": 1, "mbpkgid": 2, "node_pool_id": 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ds := tt.r.Validate(terraform.NewResourceConfigRaw(tt.config)); !ds.HasError() {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestSecurityGroupRuleProtocolCaseInsensitive(t *testing.T) {
	f := resourceSecurityGroupRule().Schema["protocol"].DiffSuppressFunc
	if f == nil || !f("protocol", "TCP", "tcp", nil) || f("protocol", "UDP", "tcp", nil) {
		t.Fatal("protocol must compare case-insensitively and still see a real change (LD-241)")
	}
}

// vapi3 refuses an attachment until vpc.readyOn is set, which can trail the VPC reading Running.
// Create retries that refusal and then reads a list row whose subnetId is a string (LD-242).
func TestDrop3ServerAttachWaitsForVPCReady(t *testing.T) {
	posts := 0
	c := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			posts++
			if posts < 3 {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"code":400,"message":"The VPC is not available for server attachments."}`)
				return
			}
			fmt.Fprint(w, `{"data":{"mbpkgid":2,"nicId":3,"state":"attached","subnetId":null}}`)
			return
		}
		fmt.Fprint(w, `{"data":{"data":[{"mbpkgid":2,"membership":"attached","subnetId":"9","nicId":3,"state":"attached","address":"10.0.0.4"}]}}`)
	})
	r := resourceVPCServerAttachment()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1, "mbpkgid": 2})
	if ds := r.CreateContext(context.Background(), d, c); ds.HasError() {
		t.Fatalf("create: %v", ds)
	}
	if posts != 3 || d.Id() != "1/2" || d.Get("subnet_id").(int) != 9 || d.Get("address").(string) != "10.0.0.4" {
		t.Fatalf("posts %d id %q subnet %v address %v", posts, d.Id(), d.Get("subnet_id"), d.Get("address"))
	}
	// Any other refusal is not retried.
	posts = 0
	other := drop2TestClient(t, func(w http.ResponseWriter, req *http.Request) {
		posts++
		w.WriteHeader(400)
		fmt.Fprint(w, `{"code":400,"message":"This server cannot be attached to a VPC."}`)
	})
	d = schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": 1, "mbpkgid": 2})
	if ds := r.CreateContext(context.Background(), d, other); !ds.HasError() || posts != 1 || !strings.Contains(fmt.Sprint(ds), "cannot be attached") {
		t.Fatalf("posts %d: %v", posts, ds)
	}
}
