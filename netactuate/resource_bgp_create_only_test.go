package netactuate

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestBGPGroupDeleteWarnsAndClearsState(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceBGPGroup().Schema, map[string]interface{}{
		"name":        "edge",
		"description": "edge group",
	})
	d.SetId("42")

	diags := resourceBGPGroupDelete(context.Background(), d, nil)
	assertCreateOnlyDeleteWarning(t, d, diags, "group remains")
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
