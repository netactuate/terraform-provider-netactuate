package netactuate

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// ipsecPeerUnknownValue marks a config value as unknown at plan time.
const ipsecPeerUnknownValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// ipsecPeerDiff runs the resource's Diff and returns its error.
func ipsecPeerDiff(t *testing.T, id string, priorAttrs map[string]string, rawConfig map[string]interface{}) (*terraform.InstanceDiff, error) {
	t.Helper()
	r := resourceRouterVRFIPSecPeer()
	var prior *terraform.InstanceState
	if id != "" {
		prior = &terraform.InstanceState{ID: id, Attributes: priorAttrs}
	}
	return r.Diff(context.Background(), prior, terraform.NewResourceConfigRaw(rawConfig), &ProviderClients{})
}

// ipsecPeerConfig returns a peer config with the required fields.
func ipsecPeerConfig() map[string]interface{} {
	return map[string]interface{}{
		"router_id":    106,
		"vrf_id":       1,
		"name":         "vm1",
		"remote_id":    "203.0.113.10",
		"psk_secret":   "dummy-psk",
		"overlay_ipv4": "169.254.10.1/30",
	}
}

func TestRouterVRFIPSecPeerCustomizeDiffPassiveWithPeerAddress(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = false
	raw["peer_address"] = "203.0.113.10"

	if _, err := ipsecPeerDiff(t, "", nil, raw); err != nil {
		t.Fatalf("passive peer with peer_address: unexpected error: %v", err)
	}
}

func TestRouterVRFIPSecPeerCustomizeDiffPassiveWithoutPeerAddress(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = false

	_, err := ipsecPeerDiff(t, "", nil, raw)
	if err == nil {
		t.Fatal("passive peer without peer_address: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "peer_address is required") {
		t.Fatalf("error = %q, want it to contain %q", err, "peer_address is required")
	}
}

func TestRouterVRFIPSecPeerCustomizeDiffInitiatingWithoutPeerAddress(t *testing.T) {
	raw := ipsecPeerConfig()

	_, err := ipsecPeerDiff(t, "", nil, raw)
	if err == nil {
		t.Fatal("initiating peer without peer_address: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "peer_address is required") {
		t.Fatalf("error = %q, want it to contain %q", err, "peer_address is required")
	}
}

func TestRouterVRFIPSecPeerCustomizeDiffInitiatingWithPeerAddress(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["peer_address"] = "203.0.113.10"

	if _, err := ipsecPeerDiff(t, "", nil, raw); err != nil {
		t.Fatalf("initiating peer with peer_address: unexpected error: %v", err)
	}
}

// An unknown peer_address passes the plan.
func TestRouterVRFIPSecPeerCustomizeDiffUnknownPeerAddressDeferred(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = true
	raw["peer_address"] = ipsecPeerUnknownValue

	diff, err := ipsecPeerDiff(t, "", nil, raw)
	if err != nil {
		t.Fatalf("unknown peer_address: unexpected error: %v", err)
	}
	if a := diff.Attributes["peer_address"]; a == nil || !a.NewComputed {
		t.Fatalf("peer_address diff = %#v, want NewComputed", a)
	}
}

// ipsecPeerPassiveState is the state of an existing passive peer.
func ipsecPeerPassiveState() map[string]string {
	return map[string]string{
		"id":                     "106/1/964",
		"router_id":              "106",
		"vrf_id":                 "1",
		"ipsec_peer_id":          "964",
		"name":                   "vm1",
		"remote_id":              "203.0.113.10",
		"psk_secret":             "dummy-psk",
		"do_initiate_connection": "false",
		"peer_address":           "203.0.113.10",
		"overlay_ipv4":           "169.254.10.1/30",
		"local_id":               "198.51.100.1",
	}
}

// An existing passive peer with peer_address plans no change.
func TestRouterVRFIPSecPeerCustomizeDiffExistingPassivePeerPlansEmpty(t *testing.T) {
	prior := ipsecPeerPassiveState()
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = false
	raw["peer_address"] = "203.0.113.10"

	diff, err := ipsecPeerDiff(t, "106/1/964", prior, raw)
	if err != nil {
		t.Fatalf("existing passive peer re-plan: unexpected error: %v", err)
	}
	if diff != nil {
		if a, ok := diff.Attributes["peer_address"]; ok {
			t.Fatalf("peer_address diff = %#v, want none", a)
		}
		if !diff.Empty() {
			t.Fatalf("diff = %#v, want empty", diff.Attributes)
		}
	}
}

// An unknown peer_address on an existing peer passes the plan.
func TestRouterVRFIPSecPeerCustomizeDiffUnknownPeerAddressOnUpdateDeferred(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = false
	raw["peer_address"] = ipsecPeerUnknownValue

	diff, err := ipsecPeerDiff(t, "106/1/964", ipsecPeerPassiveState(), raw)
	if err != nil {
		t.Fatalf("unknown peer_address on update: unexpected error: %v", err)
	}
	if a := diff.Attributes["peer_address"]; a == nil || !a.NewComputed {
		t.Fatalf("peer_address diff = %#v, want NewComputed", a)
	}
}

// Removing peer_address from a passive peer fails at plan.
func TestRouterVRFIPSecPeerCustomizeDiffPeerAddressRemovedOnUpdate(t *testing.T) {
	raw := ipsecPeerConfig()
	raw["do_initiate_connection"] = false

	_, err := ipsecPeerDiff(t, "106/1/964", ipsecPeerPassiveState(), raw)
	if err == nil {
		t.Fatal("peer_address removed on update: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "peer_address is required") {
		t.Fatalf("error = %q, want it to contain %q", err, "peer_address is required")
	}
}
