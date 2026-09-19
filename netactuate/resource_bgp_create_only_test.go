package netactuate

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestBGPGroupDeleteErrors(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBGPGroup().Schema, map[string]interface{}{
		"name":        "edge",
		"description": "edge group",
	})
	d.SetId("42")

	diags := resourceBGPGroupDelete(context.Background(), d, nil)
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic, got %#v", diags)
	}
	if !strings.Contains(strings.ToLower(diags[0].Summary), "not supported") {
		t.Fatalf("expected the error to explain delete is not supported, got %q", diags[0].Summary)
	}
	// The API has no delete endpoint for a BGP group, so a failed delete must
	// leave the id in state: the group still exists and is released through support.
	if d.Id() == "" {
		t.Fatalf("expected the state id to be preserved on a failed delete")
	}
}

func TestBGPPrefixPurchaseDeleteWarnsAndClearsState(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBGPPrefixPurchase().Schema, map[string]interface{}{
		"name":         "edge-prefix",
		"group_id":     42,
		"agreement_id": 7,
	})
	d.SetId("99")

	diags := resourceBGPPrefixPurchaseDelete(context.Background(), d, nil)
	assertCreateOnlyDeleteWarning(t, d, diags, "prefix remains")
}

func assertCreateOnlyDeleteWarning(t *testing.T, d *schema.ResourceData, diags diag.Diagnostics, want string) {
	t.Helper()
	if d.Id() != "" {
		t.Fatalf("expected Terraform state ID to be cleared, got %q", d.Id())
	}
	if len(diags) != 1 {
		t.Fatalf("expected one warning diagnostic, got %#v", diags)
	}
	if diags[0].Severity != diag.Warning {
		t.Fatalf("expected warning diagnostic, got %#v", diags[0])
	}
	if !strings.Contains(strings.ToLower(diags[0].Summary), want) {
		t.Fatalf("expected warning summary to contain %q, got %q", want, diags[0].Summary)
	}
}
