package netactuate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// fakeFirewallServer mimics the live draft/publish cycle closely enough to
// reproduce a publish defect: after Update publishes a draft, the
// published rule's priority is NOT guaranteed to match its pre-publish
// value -- a freshly-created rule's priority (0) was reassigned to 1 by the
// very next publish live, with no explicit rule_priority ever configured.
// The old code looked up the post-publish rule by priority and failed
// ("rule not found after publish"), and a subsequent refresh then dropped
// the resource from state entirely (its priority-based fallback in Read
// also missed), so the next apply would have created a live DUPLICATE rule.
//
// This mock: setID 777 has one rule (id 2084, priority 1). Update drafts it
// (draft 778, same rule mirrored at priority 1 -- the draft always mirrors
// current live state, which is why the DRAFT lookup by priority is fine),
// updates its destination port, and on publish the rule reappears on 777
// with a NEW id (2085) AND a NEW priority (2) -- simulating the exact
// renumbering the API performs.
func fakeFirewallServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	liveRule := map[string]interface{}{
		"id": 2084, "ip_version": "IPv4", "direction": "IN", "action": "ACCEPT",
		"enabled": true, "admin_comment": "allow ssh in", "rule_priority": 1,
		"match_criteria": map[string]interface{}{
			"protocol": "tcp", "source_net": []string{"0.0.0.0/0"},
			"destination_port_start": 22, "destination_port_end": 22,
		},
	}
	draftRule := map[string]interface{}{}

	mux.HandleFunc("/firewall/sets/777", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{"id": 777, "name": "example-fw", "description": "", "draft_firewall_set_id": nil}))
	})
	mux.HandleFunc("/firewall/sets/777/create-draft", func(w http.ResponseWriter, r *http.Request) {
		draftRule = map[string]interface{}{}
		for k, v := range liveRule {
			draftRule[k] = v
		}
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{"id": 778, "name": "example-fw-draft"}))
	})
	mux.HandleFunc("/firewall/sets/778/rules", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope([]map[string]interface{}{draftRule}))
	})
	mux.HandleFunc("/firewall/778/2084", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		draftRule["match_criteria"] = req["match_criteria"]
		json.NewEncoder(w).Encode(envelope(draftRule))
	})
	mux.HandleFunc("/firewall/sets/publish-draft/778", func(w http.ResponseWriter, r *http.Request) {
		liveRule["id"] = 2085
		liveRule["rule_priority"] = 2
		liveRule["match_criteria"] = draftRule["match_criteria"]
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{"id": 777}))
	})
	mux.HandleFunc("/firewall/sets/777/rules", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope([]map[string]interface{}{liveRule}))
	})
	mux.HandleFunc("/firewall/sets/777/rules/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope(liveRule))
	})
	mux.HandleFunc("/firewall/sets/777/vm/sync-all", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{}))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestResourceFirewallRuleUpdateSurvivesPriorityRenumberOnPublish is the
// regression for the confirmed live defect above: Update must succeed (and
// pick up the new id/priority) even though the published rule's priority
// changed out from under it.
func TestResourceFirewallRuleUpdateSurvivesPriorityRenumberOnPublish(t *testing.T) {
	srv := fakeFirewallServer(t)
	clients := &ProviderClients{V2: gona.NewClientCustom("test-key", srv.URL+"/")}

	priorAttrs := map[string]string{
		"firewall_set_id":        "777",
		"rule_id":                "2084",
		"ip_version":             "IPv4",
		"direction":              "IN",
		"action":                 "ACCEPT",
		"enabled":                "true",
		"admin_comment":          "allow ssh in",
		"rule_priority":          "1",
		"protocol":               "tcp",
		"destination_port_start": "22",
		"destination_port_end":   "22",
		"source_net.#":           "1",
		"source_net.0":           "0.0.0.0/0",
		"sync_after_publish":     "true",
	}
	rawConfig := map[string]interface{}{
		"firewall_set_id":        777,
		"ip_version":             "IPv4",
		"action":                 "ACCEPT",
		"enabled":                true,
		"admin_comment":          "allow ssh in",
		"protocol":               "tcp",
		"destination_port_start": 2222,
		"destination_port_end":   2222,
		"source_net":             []interface{}{"0.0.0.0/0"},
	}

	priorState := &terraform.InstanceState{ID: "777/2084", Attributes: priorAttrs}
	r := resourceFirewallRule()
	diff, err := r.Diff(context.Background(), priorState, terraform.NewResourceConfigRaw(rawConfig), clients)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	d, err := schema.InternalMap(r.Schema).Data(priorState, diff)
	if err != nil {
		t.Fatalf("Data: %v", err)
	}

	diags := resourceFirewallRuleUpdate(context.Background(), d, clients)
	if diags.HasError() {
		t.Fatalf("unexpected error updating firewall rule: %v", diags)
	}

	if got := d.Id(); got != "777/2085" {
		t.Fatalf("expected id to be refreshed to 777/2085 (new rule id after publish), got %q", got)
	}
	if got := d.Get("rule_priority").(int); got != 2 {
		t.Fatalf("expected rule_priority=2 (post-publish renumbered value), got %d", got)
	}
	if got := d.Get("destination_port_start").(int); got != 2222 {
		t.Fatalf("expected destination_port_start=2222, got %d", got)
	}
}

// TestFindRuleByRequestAmbiguous confirms the content-based correlation
// fails loudly (not silently attaching to the wrong rule) when more than
// one live rule matches the requested configuration.
func TestFindRuleByRequestAmbiguous(t *testing.T) {
	mux := http.NewServeMux()
	rule := map[string]interface{}{
		"id": 1, "ip_version": "IPv4", "direction": "IN", "action": "ACCEPT",
		"enabled": true, "admin_comment": "dup", "rule_priority": 1,
	}
	mux.HandleFunc("/firewall/sets/9/rules", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envelope([]map[string]interface{}{rule, {"id": 2, "ip_version": "IPv4", "direction": "IN", "action": "ACCEPT", "enabled": true, "admin_comment": "dup", "rule_priority": 2}}))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := gona.NewClientCustom("test-key", srv.URL+"/")

	req := &gona.CreateFirewallRuleRequest{IPVersion: "IPv4", Action: "ACCEPT", Enabled: true, AdminComment: "dup"}
	_, err := findRuleByRequest(c, 9, req)
	if err == nil || !strings.Contains(err.Error(), "cannot unambiguously correlate") {
		t.Fatalf("expected an ambiguous-match error, got: %v", err)
	}
}

func TestRuleMatchesRequestIgnoresPriorityAndID(t *testing.T) {
	rule := gona.FirewallRule{
		ID: 999, RulePriority: 42, IPVersion: "IPv4", Direction: "IN", Action: "ACCEPT",
		Enabled: true, AdminComment: "x",
		MatchCriteria: &gona.FirewallMatchCriteria{Protocol: "tcp"},
	}
	req := &gona.CreateFirewallRuleRequest{
		IPVersion: "IPv4", Direction: "IN", Action: "ACCEPT", Enabled: true, AdminComment: "x",
		MatchCriteria: &gona.FirewallMatchCriteria{Protocol: "tcp"},
	}
	if !ruleMatchesRequest(rule, req) {
		t.Fatal("expected rule to match request regardless of ID/priority")
	}
	req.MatchCriteria.Protocol = "udp"
	if ruleMatchesRequest(rule, req) {
		t.Fatal("expected rule to NOT match once match_criteria differs")
	}
}
