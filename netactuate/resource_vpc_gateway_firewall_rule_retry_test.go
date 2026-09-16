package netactuate

import (
	"errors"
	"testing"

	"github.com/netactuate/gona/gona"
)

// The retry path for the intermittent HTTP 500 on gateway firewall rule creation cannot be
// exercised on demand: the fault appears about one run in five and five consecutive live runs
// after the fix did not produce one. These tests cover the two pieces that carry the logic, so
// the behaviour is proven even though the fault cannot be summoned.

func TestIsVPCFirewallRetryable(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not retryable", nil, false},
		{"500 is retryable", errors.New("API error on POST /vpcs/1/gateway/rules/firewall: HTTP 500"), true},
		{"502 is retryable", errors.New("HTTP 502 bad gateway"), true},
		{"503 is retryable", errors.New("HTTP 503"), true},
		{"504 is retryable", errors.New("HTTP 504"), true},
		// A 4xx means the request is wrong and will stay wrong. Retrying it would turn one
		// clear error into three and delay it.
		{"400 is NOT retryable", errors.New("HTTP 400 bad request"), false},
		{"404 is NOT retryable", errors.New("HTTP 404"), false},
		{"422 is NOT retryable", errors.New("HTTP 422"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVPCFirewallRetryable(tc.err); got != tc.want {
				t.Fatalf("isVPCFirewallRetryable(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestVPCFirewallRuleMatchesRequest(t *testing.T) {
	req := func(start, end int) *gona.CreateVPCFirewallRuleRequest {
		r := &gona.CreateVPCFirewallRuleRequest{
			IPVersion: 4, Direction: "inbound", Protocol: "TCP", Network: "0.0.0.0/0",
		}
		if start != 0 || end != 0 {
			r.Port = &gona.VPCPortRange{Start: start, End: end}
		}
		return r
	}
	rule := func(start, end int) *gona.VPCFirewallRule {
		r := &gona.VPCFirewallRule{
			IPVersion: 4, Direction: "inbound", Protocol: "TCP", Network: "0.0.0.0/0",
		}
		if start != 0 || end != 0 {
			r.Port = &gona.VPCPortRange{Start: start, End: end}
		}
		return r
	}

	t.Run("a single port rule read back with end null matches", func(t *testing.T) {
		// This is the case that matters. The API stores start 22 end 22 and returns end null,
		// which unmarshals to 0. Without this the retry would never adopt and would duplicate.
		if !vpcFirewallRuleMatchesRequest(rule(22, 0), req(22, 22)) {
			t.Fatal("rule with end 0 did not match a request for 22 to 22")
		}
	})
	t.Run("a genuine range matches", func(t *testing.T) {
		if !vpcFirewallRuleMatchesRequest(rule(80, 90), req(80, 90)) {
			t.Fatal("range 80 to 90 did not match itself")
		}
	})
	t.Run("a different port does not match", func(t *testing.T) {
		if vpcFirewallRuleMatchesRequest(rule(443, 0), req(22, 22)) {
			t.Fatal("port 443 matched a request for 22, which would adopt the wrong rule")
		}
	})
	t.Run("a different end does not match", func(t *testing.T) {
		if vpcFirewallRuleMatchesRequest(rule(80, 90), req(80, 81)) {
			t.Fatal("range 80 to 90 matched a request for 80 to 81")
		}
	})
	t.Run("direction protocol and network all discriminate", func(t *testing.T) {
		r := rule(22, 0)
		r.Direction = "outbound"
		if vpcFirewallRuleMatchesRequest(r, req(22, 22)) {
			t.Fatal("an outbound rule matched an inbound request")
		}
		r = rule(22, 0)
		r.Protocol = "UDP"
		if vpcFirewallRuleMatchesRequest(r, req(22, 22)) {
			t.Fatal("a UDP rule matched a TCP request")
		}
		r = rule(22, 0)
		r.Network = "10.0.0.0/8"
		if vpcFirewallRuleMatchesRequest(r, req(22, 22)) {
			t.Fatal("a rule with a different network matched")
		}
	})
	t.Run("description discriminates only when the request sets one", func(t *testing.T) {
		r := rule(22, 0)
		r.Description = "something else"
		if !vpcFirewallRuleMatchesRequest(r, req(22, 22)) {
			t.Fatal("a request with no description should not be filtered by the rule's")
		}
		want := req(22, 22)
		want.Description = "allow ssh"
		if vpcFirewallRuleMatchesRequest(r, want) {
			t.Fatal("descriptions differ and it matched anyway")
		}
	})
	t.Run("a portless request matches only a portless rule", func(t *testing.T) {
		if !vpcFirewallRuleMatchesRequest(rule(0, 0), req(0, 0)) {
			t.Fatal("portless rule did not match portless request")
		}
		if vpcFirewallRuleMatchesRequest(rule(22, 0), req(0, 0)) {
			t.Fatal("a rule with a port matched a portless request")
		}
	})
}
