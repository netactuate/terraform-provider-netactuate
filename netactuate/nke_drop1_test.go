package netactuate

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-cty/cty"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestNKEDrop1CIDRs(t *testing.T) {
	for _, tc := range []struct {
		cidr         string
		pod, service bool
	}{
		{"10.20.0.0/16", true, true}, {"172.16.0.0/20", true, true}, {"192.168.0.0/22", true, true}, {"10.20.0.0/24", false, true}, {"10.20.0.0/25", false, false}, {"10.20.0.0/15", false, false}, {"100.64.0.0/16", true, true}, {"8.8.0.0/16", false, false}, {"10.20.1.0/20", false, false}, {"172.15.0.0/16", false, false}, {"2001:db8::/16", false, false}, {"bad", false, false},
	} {
		t.Run(tc.cidr, func(t *testing.T) {
			if got := !validatePodCIDR(tc.cidr, nil).HasError(); got != tc.pod {
				t.Errorf("pod accepted=%v", got)
			}
			if got := !validateServiceCIDRIsSlash16(tc.cidr, nil).HasError(); got != tc.service {
				t.Errorf("service accepted=%v", got)
			}
		})
	}
}
func TestNKEDrop1Schema(t *testing.T) {
	for name, r := range map[string]*schema.Resource{"pool": resourceNKENodePool(), "autoscaler": resourceNKEAutoscalerAddon(), "pools": dataSourceNKENodePools(), "storage": dataSourceNKEStorageAvailability()} {
		if err := schema.InternalMap(r.Schema).InternalValidate(nil); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	s := resourceNKENodePool().Schema
	if !s["name"].ValidateDiagFunc("default", nil).HasError() {
		t.Fatal("default must be refused")
	}
	for _, name := range []string{"valid-pool", "nah-123", "a.b_c"} {
		if s["name"].ValidateDiagFunc(name, nil).HasError() {
			t.Fatalf("valid name %s rejected", name)
		}
	}
	for _, name := range []string{"", "UPPER", "-bad", "bad/thing"} {
		if !s["name"].ValidateDiagFunc(name, nil).HasError() {
			t.Fatalf("invalid name %s accepted", name)
		}
	}
}
func TestNKEDrop1Import(t *testing.T) {
	for _, id := range []string{"1/2", "", "1", "1/2/3", "a/2", "0/2", "-1/2"} {
		d := schema.TestResourceDataRaw(t, resourceNKENodePool().Schema, nil)
		d.SetId(id)
		_, err := resourceNKENodePoolImport(context.Background(), d, nil)
		if (err == nil) != (id == "1/2") {
			t.Errorf("pool import %q: %v", id, err)
		}
	}
	for _, id := range []string{"1", "", "a", "1/2", "0", "-1"} {
		d := schema.TestResourceDataRaw(t, resourceNKEAutoscalerAddon().Schema, nil)
		d.SetId(id)
		_, err := resourceNKEAutoscalerAddonImport(context.Background(), d, nil)
		if (err == nil) != (id == "1") {
			t.Errorf("addon import %q: %v", id, err)
		}
	}
}
func TestNKEDrop1FlattenExpand(t *testing.T) {
	var p gona.NKENodePool
	if err := json.Unmarshal([]byte(`{"nodePoolId":12,"name":"nah-test","mode":"managed","packageId":4,"autoscalingEnabled":1,"minimumNodes":0,"maximumNodes":3,"desiredNodes":0,"labels":{"team":"platform"},"taints":[{"key":"dedicated","value":"batch","effect":"NoSchedule"}],"nodes":{"ready":0,"total":0,"building":0,"deleting":0,"outdated":0}}`), &p); err != nil {
		t.Fatal(err)
	}
	flat := flattenNKENodePool(&p)
	flat["cluster_id"] = 1
	labels := map[string]interface{}{}
	for k, v := range p.Labels {
		labels[k] = v
	}
	flat["labels"] = labels
	d := schema.TestResourceDataRaw(t, resourceNKENodePool().Schema, flat)
	req := expandNKENodePool(d)
	if req.MinimumNodes == nil || *req.MinimumNodes != 0 || req.Mode != "managed" || req.Labels["team"] != "platform" || len(req.Taints) != 1 {
		t.Fatalf("request: %#v", req)
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	json.Unmarshal(body, &decoded)
	if decoded["minimumNodes"] != float64(0) {
		t.Fatalf("zero missing: %s", body)
	}
	var c gona.NKECluster
	if err = json.Unmarshal([]byte(`{"networkMode":"edge-with-vpc","networks":{"pod":"10.20.0.0/20","service":"10.21.0.0/24","podEgressMode":"snat"},"nodes":{"limit":64}}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Nodes.Limit != 64 || c.Networks.PodEgressMode != "snat" {
		t.Fatalf("cluster: %#v", c)
	}
	var a gona.NKEAddon
	if err = json.Unmarshal([]byte(`{"config":{"scaleDownEnabled":false,"maxNodesTotal":10,"expander":"priority"}}`), &a); err != nil {
		t.Fatal(err)
	}
	v := flattenNKEAutoscalerConfig(a.Config.NKEAutoscalerConfig)
	if v[0].(map[string]interface{})["scale_down_enabled"] != false {
		t.Fatal(v)
	}
	var n gona.NKEWorkerNode
	json.Unmarshal([]byte(`{"publicIpAddress":"203.0.113.1","privateIpAddress":"10.0.0.2","privateNicState":"applied"}`), &n)
	d = schema.TestResourceDataRaw(t, resourceNKEWorkerNode().Schema, nil)
	if diags := setNKEWorkerNodeState(d, &n); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Get("private_nic_state") != "applied" {
		t.Fatal("private NIC state missing")
	}
}

type frozenNKEAttribute struct {
	Type                                   string
	Optional, Required, Computed, ForceNew bool
	Default                                interface{}
}

func TestNKEDrop1SchemaCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/nke_schema_origin_master.json")
	if err != nil {
		t.Fatal(err)
	}
	var baseline map[string]map[string]frozenNKEAttribute
	if err = json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	resources := map[string]*schema.Resource{"cluster": resourceNKECluster(), "worker": resourceNKEWorkerNode(), "workers": dataSourceNKEWorkerNodes()}
	for name, r := range resources {
		fields := map[string]*schema.Schema{}
		var collect func(map[string]*schema.Schema)
		collect = func(m map[string]*schema.Schema) {
			for k, s := range m {
				fields[k] = s
				if child, ok := s.Elem.(*schema.Resource); ok {
					collect(child.Schema)
				}
			}
		}
		collect(r.Schema)
		for key, want := range baseline[name] {
			s, ok := fields[key]
			if !ok {
				t.Errorf("%s.%s removed", name, key)
				continue
			}
			got := frozenNKEAttribute{Type: s.Type.String(), Optional: s.Optional, Required: s.Required, Computed: s.Computed, ForceNew: s.ForceNew, Default: s.Default}
			gotDefault, _ := json.Marshal(got.Default)
			wantDefault, _ := json.Marshal(want.Default)
			got.Default = string(gotDefault)
			want.Default = string(wantDefault)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s.%s changed: got %#v want %#v", name, key, got, want)
			}
		}
		allowed := map[string]bool{"pod_egress_mode": true, "network_mode": true, "pod_network_prefix": true, "service_network_prefix": true, "node_limit": true, "public_ip_address": true, "private_ip_address": true, "private_nic_state": true}
		for key := range fields {
			if _, ok := baseline[name][key]; !ok && !allowed[key] {
				t.Errorf("unexpected addition %s.%s", name, key)
			}
		}
	}
}
func TestNKEDrop1LegacyStateNoDiff(t *testing.T) {
	r := resourceNKECluster()
	data, err := os.ReadFile("testdata/nke_cluster_v0.4.10.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		ID     string
		Config map[string]interface{}
		State  map[string]interface{}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	config := fixture.Config
	d := schema.TestResourceDataRaw(t, r.Schema, config)
	d.SetId(fixture.ID)
	for key, value := range fixture.State {
		if err = d.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	state := d.State()
	for _, key := range []string{"pod_egress_mode", "network_mode", "pod_network_prefix", "service_network_prefix", "node_limit"} {
		delete(state.Attributes, key)
	}
	diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil {
		for k, a := range diff.Attributes {
			if !a.NewComputed && a.New != a.Old {
				t.Errorf("unexpected diff %s: %#v", k, a)
			}
		}
	}
}

func TestNKEDrop1NetworkModeReplacement(t *testing.T) {
	for _, tc := range []struct {
		old, configured string
		replace         bool
	}{
		{"", "edge-with-vpc", false}, {"edge", "edge", false}, {"edge", "edge-with-vpc", true}, {"edge", "", false},
	} {
		t.Run(tc.old+"_to_"+tc.configured, func(t *testing.T) {
			r := resourceNKECluster()
			config := map[string]interface{}{"name": "nah-test", "version": "1.35.1", "minimum_nodes": 1, "maximum_nodes": 2, "plan": "test", "contract_id": 1, "location_id": 1}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			d.SetId("1")
			if tc.old != "" {
				d.Set("network_mode", tc.old)
			}
			if tc.configured != "" {
				config["network_mode"] = tc.configured
			}
			state := d.State()
			configured := cty.NullVal(cty.String)
			if tc.configured != "" {
				configured = cty.StringVal(tc.configured)
			}
			state.RawConfig = cty.ObjectVal(map[string]cty.Value{"network_mode": configured})
			diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff.RequiresNew() != tc.replace {
				t.Errorf("replacement=%v want %v", diff.RequiresNew(), tc.replace)
			}
		})
	}
}
func TestNKEDrop1AutoscalerValidators(t *testing.T) {
	fields := resourceNKEAutoscalerAddon().Schema["config"].Elem.(*schema.Resource).Schema
	for _, tc := range []struct {
		key   string
		value interface{}
		valid bool
	}{
		{"scale_down_unneeded_time", "1m", true}, {"scale_down_unneeded_time", "24h", true}, {"scale_down_unneeded_time", "59s", false}, {"scale_down_unneeded_time", "25h", false}, {"scale_down_unneeded_time", "1m1s", false},
		{"max_node_provision_time", "5m", true}, {"max_node_provision_time", "60m", true}, {"max_node_provision_time", "4m", false}, {"max_node_provision_time", "61m", false},
		{"scale_down_utilization_threshold", 0.1, true}, {"scale_down_utilization_threshold", 0.9, true}, {"scale_down_utilization_threshold", 0.0, false}, {"scale_down_utilization_threshold", 1.0, false},
		{"expander", "priority", true}, {"expander", "random", false}, {"max_nodes_total", 1, true}, {"max_nodes_total", 1000, true}, {"max_nodes_total", 0, false}, {"max_nodes_total", 1001, false},
	} {
		if valid := !fields[tc.key].ValidateDiagFunc(tc.value, nil).HasError(); valid != tc.valid {
			t.Errorf("%s=%v accepted=%v", tc.key, tc.value, valid)
		}
	}
}

func TestNKEDrop1ConfiguredZeroAndFalse(t *testing.T) {
	pool := resourceNKENodePool()
	for _, configured := range []bool{false, true} {
		d := schema.TestResourceDataRaw(t, pool.Schema, map[string]interface{}{"cluster_id": 1, "name": "nah-test", "type": "managed", "package_id": 2, "desired_nodes": 0, "autoscaling": []interface{}{map[string]interface{}{"enabled": false, "minimum_nodes": 0}}})
		d.SetId("1/2")
		state := d.State()
		enabled, minimum, desired := cty.NullVal(cty.Bool), cty.NullVal(cty.Number), cty.NullVal(cty.Number)
		if configured {
			enabled = cty.False
			minimum = cty.NumberIntVal(0)
			desired = cty.NumberIntVal(0)
		}
		state.RawConfig = cty.ObjectVal(map[string]cty.Value{"desired_nodes": desired, "autoscaling": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"enabled": enabled, "minimum_nodes": minimum, "maximum_nodes": cty.NullVal(cty.Number)})})})
		d = pool.Data(state)
		req := expandNKENodePool(d)
		if (req.MinimumNodes != nil) != configured {
			t.Errorf("minimum configured=%v request=%#v", configured, req)
		}
		if nkePoolDesiredConfigured(d) != configured || nkePoolBlockFieldConfigured(d, "enabled") != configured {
			t.Errorf("zero or false presence lost: configured=%v", configured)
		}
	}
	addon := resourceNKEAutoscalerAddon()
	d := schema.TestResourceDataRaw(t, addon.Schema, map[string]interface{}{"cluster_id": 1, "config": []interface{}{map[string]interface{}{"scale_down_enabled": false, "expander": "priority"}}})
	d.SetId("1")
	state := d.State()
	attrs := map[string]cty.Value{}
	configType := addon.CoreConfigSchema().ImpliedType().AttributeType("config").ElementType()
	for key, typ := range configType.AttributeTypes() {
		attrs[key] = cty.NullVal(typ)
	}
	attrs["scale_down_enabled"] = cty.False
	attrs["expander"] = cty.StringVal("priority")
	state.RawConfig = cty.ObjectVal(map[string]cty.Value{"config": cty.ListVal([]cty.Value{cty.ObjectVal(attrs)})})
	req := expandNKEAutoscalerConfig(addon.Data(state))
	if req.ScaleDownEnabled == nil || *req.ScaleDownEnabled || req.Expander == nil || *req.Expander != "priority" || req.MaxNodesTotal != nil {
		t.Fatalf("addon request: %#v", req)
	}
}
