package netactuate

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestHostnameRegex(t *testing.T) {
	tests := []struct {
		hostname string
		valid    bool
		desc     string
	}{
		// Valid single-label hostnames
		{"a", true, "single character hostname"},
		{"z", true, "single character hostname"},
		{"0", true, "single digit hostname"},
		{"9", true, "single digit hostname"},
		{"example", true, "simple hostname"},
		{"server1", true, "hostname with number"},
		{"web-server", true, "hostname with hyphen"},
		{"my-server-123", true, "hostname with multiple hyphens and numbers"},
		{"a1b2c3", true, "hostname with mixed alphanumeric"},

		// Valid multi-label hostnames (FQDNs)
		{"example.com", true, "simple FQDN"},
		{"www.example.com", true, "subdomain FQDN"},
		{"api.v1.example.com", true, "multiple subdomain levels"},
		{"my-server.example.com", true, "subdomain with hyphen"},
		{"server1.dc2.example.com", true, "multiple labels with numbers"},
		{"a.b.c.d.e.f.g", true, "many label levels"},
		{"192.0.2.1", true, "numeric labels (though unusual for hostnames)"},
		{"web-01.prod-us-east-1.example.com", true, "complex production hostname"},

		// Invalid: starts with hyphen
		{"-server", false, "starts with hyphen"},
		{"-example.com", false, "label starts with hyphen"},
		{"server.-example.com", false, "second label starts with hyphen"},

		// Invalid: ends with hyphen
		{"server-", false, "ends with hyphen"},
		{"example-.com", false, "label ends with hyphen"},
		{"server.example-.com", false, "second label ends with hyphen"},

		// Invalid: starts or ends with dot
		{".example", false, "starts with dot"},
		{"example.", false, "ends with dot"},
		{".example.com", false, "starts with dot"},
		{"example.com.", false, "ends with dot"},

		// Invalid: double dots
		{"example..com", false, "double dot"},
		{"server..example.com", false, "double dot in middle"},

		// Invalid: special characters
		{"example_com", false, "underscore not allowed"},
		{"example@com", false, "@ symbol not allowed"},
		{"example#com", false, "# symbol not allowed"},
		{"example com", false, "space not allowed"},
		{"example/com", false, "slash not allowed"},

		// Invalid: empty string
		{"", false, "empty string"},

		// Edge cases: single character labels with dots
		{"a.b", true, "single char labels with dot"},
		{"a.b.c", true, "multiple single char labels"},

		// Edge cases: hyphen placement
		{"a-b", true, "hyphen between chars"},
		{"a-b-c", true, "multiple hyphens"},
		{"1-2-3", true, "hyphens between numbers"},
		{"a--b", true, "consecutive hyphens in middle (valid)"},

		// Realistic hostnames
		{"terraform.example.com", true, "realistic terraform hostname"},
		{"prod-web-01.us-east-1.example.com", true, "realistic production hostname"},
		{"db-master-001.internal.example.com", true, "realistic database hostname"},
		{"k8s-worker-node-42.cluster.local", true, "realistic kubernetes hostname"},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			match := hostnameRegex.MatchString(tt.hostname)
			if match != tt.valid {
				t.Errorf("hostname %q: expected valid=%v, got valid=%v (%s)",
					tt.hostname, tt.valid, match, tt.desc)
			}
		})
	}
}

func TestHostnameRegexValue(t *testing.T) {
	// Verify the regex pattern is what we expect
	expectedInner := "([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])"
	expectedOuter := fmt.Sprintf("^(%s\\.)*%s$", expectedInner, expectedInner)

	actualOuter := hostnameRegex.String()

	if actualOuter != expectedOuter {
		t.Errorf("hostnameRegex pattern mismatch:\nexpected: %s\ngot:      %s",
			expectedOuter, actualOuter)
	}
}

// fieldsToRebuild (resource_server.go's resourceServerUpdate) checks raw
// d.HasChange("location_id")/d.HasChange("image_id") with no DiffSuppressFunc
// and no unchangedLocationID/unchangedImageID guard on those two fields --
// that guard only protects the "location"/"image" *name* fields' own diffs
// (see suppressLocationDiff/suppressImageDiff in helper.go).
//
// resourceServerRead only ever calls updateValue("location_id", ...) /
// updateValue("image_id", ...), and updateValue is a no-op unless the field
// already has a non-zero value in state (see helper.go's updateValue: it
// gates on d.GetOk(key) before calling d.Set). So a server created with only
// `location`/`image` (name form) keeps location_id/image_id at their zero
// value forever: location_id remains null after creation and refresh.
//
// The consequence: the moment a config adds `location_id`/`image_id` set to
// the server's OWN real, unchanged location/image (e.g. migrating from name-
// based to ID-based config, exactly the v0.2.5->v0.3.0 customer scenario),
// Terraform sees a genuine null -> real-id diff on that field with nothing to
// suppress it, `d.HasChange("location_id")` is true, `rebuildRequired`
// becomes true, and resourceServerUpdate deletes and rebuilds the server --
// while `terraform plan` shows only "update in-place" (location_id is not
// ForceNew). These tests reproduce that transition without live resources.
// rebuildDiff drives the resource's real SDK diff engine (the same one
// Terraform's plan step uses, including CustomizeDiff) from a prior state and
// a raw config map, then returns a ResourceData backed by that diff so
// HasChange reflects the SDK's actual decision -- not a hand-simulated one.
// (A ResourceData built via resourceServer().Data(state) plus d.Set() looks
// like it should work but doesn't: Set() only affects Get(), not
// GetChange()/HasChange(), because there is no diff attached. Confirmed by
// direct experiment while writing this test.)
func rebuildDiff(t *testing.T, priorAttrs map[string]string, rawConfig map[string]interface{}) *schema.ResourceData {
	t.Helper()
	return rebuildDiffWithResource(t, resourceServer(), priorAttrs, rawConfig)
}

// rebuildDiffWithResource is rebuildDiff but takes an explicit *schema.Resource
// so a test can mutate a fresh copy's schema (e.g. flip a field's Computed
// flag) before driving the diff, without touching resource_server.go itself.
// resourceServer() returns a brand-new map/struct literal on every call, so
// mutating the *schema.Schema entries on one call's result never leaks into
// another call's -- each caller gets an independent copy.
func rebuildDiffWithResource(t *testing.T, r *schema.Resource, priorAttrs map[string]string, rawConfig map[string]interface{}) *schema.ResourceData {
	t.Helper()
	_, d := rawDiffAndData(t, r, priorAttrs, rawConfig)
	return d
}

// rawDiffAndData is rebuildDiffWithResource but also returns the raw
// *terraform.InstanceDiff, for tests that need to inspect a specific
// attribute's NewComputed flag directly (schema.ResourceData has no getter
// for "is this attribute Computed in the plan").
func rawDiffAndData(t *testing.T, r *schema.Resource, priorAttrs map[string]string, rawConfig map[string]interface{}) (*terraform.InstanceDiff, *schema.ResourceData) {
	t.Helper()
	priorState := &terraform.InstanceState{ID: "1001", Attributes: priorAttrs}
	diff, err := r.Diff(context.Background(), priorState, terraform.NewResourceConfigRaw(rawConfig), &ProviderClients{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	d, err := schema.InternalMap(r.Schema).Data(priorState, diff)
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	return diff, d
}

// rebuildDiffWithRawConfigTags is rebuildDiff but additionally sets the
// diff's RawConfig (a cty object exposing just "tags"), mirroring exactly
// what the real provider server does (helper/schema/grpc_provider.go sets
// `diff.RawConfig = configVal` after calling Resource.Diff -- Diff() itself
// never populates RawConfig). d.GetRawConfig() falls back to a NullVal when
// no RawConfig is set at all, which is why tagsConfiguredInRequest's "already
// configured, value unchanged" case (no diff entry for tags, since old==new)
// can't be exercised through plain rebuildDiff: needed for a true
// regression test of the GetRawConfig-based fix, not just the two cases
// that happen to produce a real diff entry.
func rebuildDiffWithRawConfigTags(t *testing.T, priorAttrs map[string]string, rawConfig map[string]interface{}, tagsConfigured bool, tagsVal string) *schema.ResourceData {
	t.Helper()
	r := resourceServer()
	priorState := &terraform.InstanceState{ID: "1001", Attributes: priorAttrs}
	diff, err := r.Diff(context.Background(), priorState, terraform.NewResourceConfigRaw(rawConfig), &ProviderClients{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	tagsCty := cty.NullVal(cty.String)
	if tagsConfigured {
		tagsCty = cty.StringVal(tagsVal)
	}
	diff.RawConfig = cty.ObjectVal(map[string]cty.Value{"tags": tagsCty})
	d, err := schema.InternalMap(r.Schema).Data(priorState, diff)
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	return d
}

// ctyObjectFromRawConfig converts a plain Go map (the same shape used
// throughout this file's rawConfig fixtures) into a cty object, so tests
// can build a realistic RawConfig without hand-writing cty.ObjectVal
// literals for every field.
func ctyObjectFromRawConfig(t *testing.T, rawConfig map[string]interface{}) cty.Value {
	t.Helper()
	vals := make(map[string]cty.Value, len(rawConfig))
	for k, v := range rawConfig {
		switch tv := v.(type) {
		case string:
			vals[k] = cty.StringVal(tv)
		case int:
			vals[k] = cty.NumberIntVal(int64(tv))
		case bool:
			vals[k] = cty.BoolVal(tv)
		default:
			t.Fatalf("ctyObjectFromRawConfig: unsupported type %T for key %q", v, k)
		}
	}
	return cty.ObjectVal(vals)
}

// rebuildDiffWithRealRawConfig is rebuildDiff but additionally sets
// priorState.RawConfig to the SAME rawConfig map handed to Resource.Diff --
// this is the mechanism the real provider server actually uses
// (helper/schema/grpc_provider.go's PlanResourceChange sets
// `priorState.RawConfig`, not `diff.RawConfig`; confirmed via direct
// experiment that GetRawConfig()'s state-level fallback picks it up
// correctly). Needed for any test exercising CatalogRef.unchangedID, which
// requires a real, plan-time-accurate RawConfig to distinguish "id field
// explicitly configured, unchanged" from "id field simply never mentioned
// at all" -- the exact ambiguity that caused a real, confirmed bug (see
// TestSuppressNameDiffDoesNotSuppressGenuineNameOnlyChange).
func rebuildDiffWithRealRawConfig(t *testing.T, priorAttrs map[string]string, rawConfig map[string]interface{}) *schema.ResourceData {
	t.Helper()
	priorState := &terraform.InstanceState{ID: "1001", Attributes: priorAttrs}
	priorState.RawConfig = ctyObjectFromRawConfig(t, rawConfig)
	diff, err := resourceServer().Diff(context.Background(), priorState, terraform.NewResourceConfigRaw(rawConfig), &ProviderClients{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	d, err := schema.InternalMap(resourceServer().Schema).Data(priorState, diff)
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	return d
}

func rebuildRequired(d *schema.ResourceData) bool {
	for _, f := range []string{"location", "location_id", "image", "image_id", "hostname", "params"} {
		if d.HasChange(f) {
			return true
		}
	}
	return false
}

// The hydration-gap bug (formerly reproduced by TestLocationIDHydrationFalsely-
// TriggersRebuild / TestImageIDHydrationFalselyTriggersRebuild) is fixed by
// CatalogRef (catalogref.go): resourceServerRead now calls
// serverLocationPair.Hydrate/serverImagePair.Hydrate unconditionally instead
// of the old updateValue (which only refreshed an already-non-zero field).
// Each test below is two parts: (A) a direct, network-free proof that Hydrate
// actually writes the id field from a stale/zero starting point -- the exact
// inversion of the original bug report -- and (B) an end-to-end proof, via
// the SDK's real diff engine, that a server whose Read has *already* hydrated
// the id field (i.e. what a fixed Read actually leaves in state, not the old
// buggy "0") shows no diff and no rebuild when a config migrates from
// name-only to the matching id-only form. (B) intentionally does NOT reuse
// the old bug's "0" fixture: feeding a real diff engine a genuinely stale "0"
// next to a configured "236" is *correctly* a diff -- the fix's guarantee is
// that stale state can no longer arise, not that this diff stops being a
// diff, so (B) must start from the corrected prior state to mean anything.
func TestLocationIDHydrationNoLongerTriggersRebuild(t *testing.T) {
	// (A) direct: Hydrate must overwrite a stale/zero location_id.
	hd := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "0",
			"image": "Debian 12 x64 (20241016)", "image_id": "0",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{"hostname": "server.example.test", "plan": "VR1x1x25", "location": "RDU", "image": "Debian 12 x64 (20241016)", "ssh_key_id": 2001},
	)
	var diags diag.Diagnostics
	serverLocationPair.Hydrate(hd, 236, "RDU - Raleigh, NC", &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := hd.Get("location_id").(int); got != 236 {
		t.Fatalf("expected Hydrate to write location_id=236 from a stale 0, got %d", got)
	}

	// (B) end-to-end: prior state corrected to what a fixed Read actually
	// leaves (location_id already 236, not stale "0"), config migrates from
	// location-only to location_id-only for the SAME real value -> no diff,
	// no rebuild for an unchanged configuration.
	d2 := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236", // correctly hydrated, not "0"
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, // config migrated to id-only, same real value
			"image":       "Debian 12 x64 (20241016)",
			"ssh_key_id":  2001,
		},
	)
	if d2.HasChange("location_id") {
		t.Fatal("expected NO change on location_id: migrating to the server's own real id must be a no-op now that Hydrate keeps it fresh")
	}
	if rebuildRequired(d2) {
		t.Fatal("expected rebuildRequired to be false: the fix should mean this migration never reaches resourceServerUpdate's rebuild path")
	}
}

// Same fix, image_id side.
func TestImageIDHydrationNoLongerTriggersRebuild(t *testing.T) {
	hd := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "0",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{"hostname": "server.example.test", "plan": "VR1x1x25", "location_id": 236, "image": "Debian 12 x64 (20241016)", "ssh_key_id": 2001},
	)
	var diags diag.Diagnostics
	serverImagePair.Hydrate(hd, 5794, "Debian 12 x64 (20241016)", &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := hd.Get("image_id").(int); got != 5794 {
		t.Fatalf("expected Hydrate to write image_id=5794 from a stale 0, got %d", got)
	}

	d2 := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794", // correctly hydrated, not "0"
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236,
			"image_id":    5794, // config migrated to id-only, same real value
			"ssh_key_id":  2001,
		},
	)
	if d2.HasChange("image_id") {
		t.Fatal("expected NO change on image_id: migrating to the server's own real id must be a no-op now that Hydrate keeps it fresh")
	}
	if rebuildRequired(d2) {
		t.Fatal("expected rebuildRequired to be false for the image_id migration case")
	}
}

// TestComputedAloneDoesNotFixHydrationGap's original hypothesis ("does adding
// Computed alone fix it") is no longer counterfactual -- image_id really does
// ship with Computed: true now, via CatalogRef.Schemas(). Repointed at what
// Computed actually does: an id field that Hydrate has already correctly
// populated, left unconfigured by the user, must stay quiet (no diff) across
// repeated plans -- the genuine, narrow job Computed does, distinct from (and
// insufficient without) the Hydrate fix the tests above cover.
func TestComputedAloneDoesNotFixHydrationGap(t *testing.T) {
	if !resourceServer().Schema["image_id"].Computed {
		t.Fatal("expected image_id to be Computed: true (via CatalogRef.Schemas()) -- if false, CatalogRef's schema wiring regressed")
	}

	d := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794", // already correctly hydrated
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236,
			"image":       "Debian 12 x64 (20241016)", // image_id left unconfigured
			"ssh_key_id":  2001,
		},
	)
	if d.HasChange("image_id") {
		t.Fatal("expected no diff on an unconfigured, already-hydrated image_id -- this is what Computed: true is actually for")
	}
	if d.HasChange("image") {
		t.Fatal("expected no diff on image (unchanged, same spelling)")
	}
}

// ComputedIf's primary_ipv4/primary_ipv6 rules used a raw d.HasChange("image"),
// which runs during CustomizeDiff -- BEFORE suppressNameDiff (image's
// DiffSuppressFunc) is applied -- so a purely cosmetic image spelling
// difference (case, IATA vs full name) showed primary_ipv4/6 as "known
// after apply" in a real plan, even though the
// real apply-time d.HasChange("image") and rebuildRequired were both false
// (apply would have been a true no-op). Fixed via CatalogRef.NameReallyChanged,
// which mirrors suppressNameDiff's own logic for CustomizeDiff's benefit.
// This test drives the actual wired-up ComputedIf rule (via a full
// resourceServer().Diff() call, not NameReallyChanged in isolation --
// schema.ResourceDiff has no public constructor outside the SDK's own
// CustomizeDiff invocation), inspecting the resulting InstanceDiff's
// NewComputed flag directly (schema.ResourceData has no getter for it).
func TestPrimaryIPNotMarkedComputedForCosmeticImageChange(t *testing.T) {
	priorAttrs := map[string]string{
		"hostname": "server.example.test", "plan": "VR1x1x25",
		"location": "RDU", "location_id": "236",
		"image": "Debian 12 x64 (20241016)", "image_id": "5794",
		"ssh_key_id":   "2001",
		"primary_ipv4": "192.0.2.10", "primary_ipv6": "2001:db8::10",
	}
	diff, _ := rawDiffAndData(t, resourceServer(), priorAttrs,
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
			"location": "RDU",
			"image":    "debian 12 X64 (20241016)", // case variant only
		},
	)
	if a, ok := diff.Attributes["primary_ipv4"]; ok && a.NewComputed {
		t.Fatalf("expected primary_ipv4 NOT to be marked Computed for a cosmetic-only image change, got %+v", a)
	}
	if a, ok := diff.Attributes["primary_ipv6"]; ok && a.NewComputed {
		t.Fatalf("expected primary_ipv6 NOT to be marked Computed for a cosmetic-only image change, got %+v", a)
	}
}

// A genuinely different image must still mark primary_ipv4/6 Computed --
// proving the fix didn't over-suppress the signal for a real change.
func TestPrimaryIPStillMarkedComputedForGenuineImageChange(t *testing.T) {
	priorAttrs := map[string]string{
		"hostname": "server.example.test", "plan": "VR1x1x25",
		"location": "RDU", "location_id": "236",
		"image": "Debian 12 x64 (20241016)", "image_id": "5794",
		"ssh_key_id":   "2001",
		"primary_ipv4": "192.0.2.10", "primary_ipv6": "2001:db8::10",
	}
	diff, _ := rawDiffAndData(t, resourceServer(), priorAttrs,
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "ssh_key_id": 2001,
			"location": "RDU",
			// genuinely different image: name AND id both change together
			// (id 5787, Ubuntu 24.04 LTS), unlike the cosmetic-change test
			// above, image_id itself changes too, so unchangedID correctly
			// does NOT treat this as "id proves nothing moved".
			"image":    "Ubuntu 24.04 LTS (20240423)",
			"image_id": 5787,
		},
	)
	a, ok := diff.Attributes["primary_ipv4"]
	if !ok || !a.NewComputed {
		t.Fatalf("expected primary_ipv4 to be marked Computed for a genuinely different image, got %+v (ok=%v)", a, ok)
	}
}

// TestHydrateServerBillingFieldsRecoversImportedValues is the regression for
// the server-import gap: resourceServerRead previously never read
// package_billing_contract_id back from the API at all -- and gona.Server
// couldn't have supplied a correct value anyway, since its struct tag didn't
// match the live API's actual response field name (fixed separately in
// gona/servers.go; see that commit). A server imported via
// schema.ImportStatePassthroughContext (Read-only) left this field null
// forever, and a subsequent apply would silently accept whatever config
// said into state without ever calling any API for it (it is not in
// resourceServerUpdate's fieldsToRebuild). This asserts the fixed behavior:
// a fresh ResourceData (nothing set, exactly import's starting point) ends
// up with the field correctly populated from a gona.Server value.
func TestHydrateServerBillingFieldsRecoversImportedValues(t *testing.T) {
	d := resourceServer().Data(&terraform.InstanceState{ID: "1002", Attributes: map[string]string{}})

	var diags diag.Diagnostics
	hydrateServerBillingFields(d, gona.Server{
		PackageBillingContractId: 440, // API's "contract_id" is a JSON number
	}, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := d.Get("package_billing_contract_id").(string); got != "440" {
		t.Fatalf("expected package_billing_contract_id = %q, got %q", "440", got)
	}
}

// A server billed via package_billing_opt_in (no contract) reports
// contract_id: 0 -- that must become "" in state, not the literal string
// "0", which would look like a real (if odd) contract id.
func TestHydrateServerBillingFieldsTreatsZeroContractIDAsEmpty(t *testing.T) {
	d := resourceServer().Data(&terraform.InstanceState{ID: "1002", Attributes: map[string]string{}})

	var diags diag.Diagnostics
	hydrateServerBillingFields(d, gona.Server{PackageBillingContractId: 0}, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := d.Get("package_billing_contract_id").(string); got != "" {
		t.Fatalf("expected package_billing_contract_id = \"\" for a zero contract id, got %q", got)
	}
}

func BenchmarkHostnameRegex(b *testing.B) {
	pattern := hostnameRegex.String()
	regex := regexp.MustCompile(pattern)
	hostname := "my-server-01.prod-us-east-1.example.com"

	b.Run("PreviousBehavior", func(b *testing.B) {

		b.ResetTimer()
		for b.Loop() {
			regexp.MatchString(pattern, hostname)
		}
	})

	b.Run("CompiledRegex", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			regex.MatchString(hostname)
		}
	})
}
