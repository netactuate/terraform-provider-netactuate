package netactuate

import (
	"github.com/netactuate/gona/gona"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testRef() CatalogRef {
	return CatalogRef{NameField: "image", IDField: "image_id"}
}

func testRefData(t *testing.T, id, name string, idVal int) *schema.ResourceData {
	t.Helper()
	return resourceServer().Data(&terraform.InstanceState{
		ID: id,
		Attributes: map[string]string{
			"image":    name,
			"image_id": strconv.Itoa(idVal),
		},
	})
}

// A config that pins the id only leaves the name unset (new == ""), which
// must never show a diff regardless of whatever name got written into state
// by a prior Read.
func TestCatalogRefSuppressNameDiffAllowsIDOnlyConfig(t *testing.T) {
	r := testRef()
	if !r.suppressNameDiff("image", "Ubuntu 22.04 (20220420)", "", nil) {
		t.Fatal("expected an empty configured name (id-only config) to suppress the diff")
	}
}

func TestCatalogRefSuppressNameDiffDefaultsToEqualFold(t *testing.T) {
	r := testRef()
	if !r.suppressNameDiff("image", "Ubuntu 22.04 (20220420)", "ubuntu 22.04 (20220420)", nil) {
		t.Fatal("expected a case-only difference to suppress the diff with the default (EqualFold) SameName")
	}
}

// A custom SameName (e.g. an IATA-aware matcher) must be used instead of the
// EqualFold default when provided.
func TestCatalogRefSuppressNameDiffUsesCustomSameName(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{{ID: 1, Name: "RDU - Raleigh, NC", IATACode: "rdu"}})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })
	r := CatalogRef{NameField: "location", IDField: "location_id", SameName: sameLocation}
	if !r.suppressNameDiff("location", "RDU - Raleigh, NC", "RDU", nil) {
		t.Fatal("expected a code that resolves to the same location to suppress the diff")
	}
	if r.suppressNameDiff("location", "RDU - Raleigh, NC", "LGA", nil) {
		t.Fatal("expected the custom SameName to reject a genuinely different location")
	}
}

// An id EXPLICITLY reconfigured to the same value it already had proves the
// underlying entry did not change even if its display name was re-spelled
// (e.g. by the API, between reads). Uses rebuildDiffWithRealRawConfig, not
// the old .Data(state)-only construction: unchangedID now requires the id
// field to actually appear in GetRawConfig() before trusting old==new (see
// TestSuppressNameDiffDoesNotSuppressGenuineNameOnlyChange for why), and
// .Data(state) alone never populates RawConfig at all.
func TestCatalogRefSuppressNameDiffTrustsUnchangedID(t *testing.T) {
	d := rebuildDiffWithRealRawConfig(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Ubuntu 22.04 (20220420)", "image_id": "391",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "ssh_key_id": 2001,
			"location": "RDU",
			"image":    "Ubuntu 22.04 LTS", // re-spelled display name
			"image_id": 391,                // explicitly reconfigured to the SAME id
		},
	)

	if !serverImagePair.suppressNameDiff("image", "Ubuntu 22.04 (20220420)", "Ubuntu 22.04 LTS", d) {
		t.Fatal("expected an id explicitly reconfigured to the same value to suppress a display-name mismatch")
	}
}

// The bug this whole family of tests exists to pin down: unchangedID used
// to trust oldID == newID even when the id field was simply never
// configured at all (the ordinary way a user edits a name-only config) --
// which is ALWAYS true for an untouched field, so it silently suppressed
// any name-only change, including changing the Debian image to a
// genuinely different, unrelated Ubuntu image, with image_id left
// unconfigured, planned as "No changes." -- the config change would never
// have taken effect. Fixed by requiring the id field to actually appear in
// GetRawConfig() before unchangedID trusts it.
func TestSuppressNameDiffDoesNotSuppressGenuineNameOnlyChange(t *testing.T) {
	d := rebuildDiffWithRealRawConfig(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "ssh_key_id": 2001,
			"location": "RDU",
			"image":    "Ubuntu 24.04 LTS (20240423)", // genuinely different; image_id NOT configured
		},
	)

	if d.HasChange("image") == false {
		t.Fatal("expected a genuine name-only image change (image_id left unconfigured) to produce a real diff, not be silently suppressed")
	}
}

// Same bug class, location side: sameLocation's IATA-prefix matching won't
// catch a genuinely different location either, so this exercises the same
// unchangedID fix from the other field CatalogRef guards.
func TestSuppressNameDiffDoesNotSuppressGenuineLocationOnlyChange(t *testing.T) {
	d := rebuildDiffWithRealRawConfig(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"image_id": 5794, "ssh_key_id": 2001,
			"location": "LGA", // genuinely different; location_id NOT configured
		},
	)

	if !d.HasChange("location") {
		t.Fatal("expected a genuine name-only location change (location_id left unconfigured) to produce a real diff, not be silently suppressed")
	}
}

// Without an id there is no prior id to compare, so a mismatch is a real
// change.
func TestCatalogRefSuppressNameDiffReportsMismatchOnCreate(t *testing.T) {
	r := testRef()
	d := testRefData(t, "", "", 0)

	if r.suppressNameDiff("image", "Ubuntu 22.04 (20220420)", "CentOS 7", d) {
		t.Fatal("expected a name mismatch with no prior id to produce a diff")
	}
}

// A zero id carries no information, so it must not be read as proof the
// entry is unchanged.
func TestCatalogRefSuppressNameDiffIgnoresZeroID(t *testing.T) {
	r := testRef()
	d := testRefData(t, "1003", "Ubuntu 22.04 (20220420)", 0)

	if r.suppressNameDiff("image", "Ubuntu 22.04 (20220420)", "CentOS 7", d) {
		t.Fatal("expected a zero id to produce a diff")
	}
}

func TestResolveCatalogEntryMatchesNameOrAltName(t *testing.T) {
	entries := []CatalogEntry{
		{ID: 236, Name: "RDU - Raleigh, NC", AltNames: []string{"rdu"}},
		{ID: 500, Name: "LGA - New York, NY", AltNames: []string{"lga"}},
	}

	if e, ok := resolveCatalogEntry("RDU", entries); !ok || e.ID != 236 {
		t.Fatalf("expected AltName match on RDU -> id 236, got ok=%v id=%d", ok, e.ID)
	}
	if e, ok := resolveCatalogEntry("lga - new york, ny", entries); !ok || e.ID != 500 {
		t.Fatalf("expected case-insensitive Name match -> id 500, got ok=%v id=%d", ok, e.ID)
	}
}

func TestResolveCatalogEntryReturnsNotFoundForUnmatchedName(t *testing.T) {
	entries := []CatalogEntry{{ID: 236, Name: "RDU - Raleigh, NC", AltNames: []string{"rdu"}}}
	if _, ok := resolveCatalogEntry("nonexistent", entries); ok {
		t.Fatal("expected no match for a name absent from the catalog")
	}
}

func TestCatalogRefResolvePrefersConfiguredID(t *testing.T) {
	r := testRef()
	d := testRefData(t, "1", "", 391) // image_id configured, image absent

	fetchCalled := false
	id, dg := r.Resolve(d, func() ([]CatalogEntry, error) {
		fetchCalled = true
		return nil, nil
	})
	if dg != nil {
		t.Fatalf("unexpected diagnostic: %v", *dg)
	}
	if id != 391 {
		t.Fatalf("expected the configured id (391) to be returned directly, got %d", id)
	}
	if fetchCalled {
		t.Fatal("expected fetch not to be called when the id field is already configured")
	}
}

func TestCatalogRefResolveResolvesNameAgainstFetch(t *testing.T) {
	r := testRef()
	d := resourceServer().Data(&terraform.InstanceState{
		ID:         "1",
		Attributes: map[string]string{"image": "Ubuntu 22.04 (20220420)"},
	})

	id, dg := r.Resolve(d, func() ([]CatalogEntry, error) {
		return []CatalogEntry{{ID: 391, Name: "Ubuntu 22.04 (20220420)"}}, nil
	})
	if dg != nil {
		t.Fatalf("unexpected diagnostic: %v", *dg)
	}
	if id != 391 {
		t.Fatalf("expected the resolved id 391, got %d", id)
	}
}

func TestCatalogRefResolveErrorsWhenNeitherFieldConfigured(t *testing.T) {
	r := testRef()
	d := testRefData(t, "", "", 0)

	_, dg := r.Resolve(d, func() ([]CatalogEntry, error) { return nil, nil })
	if dg == nil {
		t.Fatal("expected a diagnostic when neither the name nor the id field is configured")
	}
}

func TestCatalogRefResolvePropagatesFetchError(t *testing.T) {
	r := testRef()
	d := resourceServer().Data(&terraform.InstanceState{
		ID:         "1",
		Attributes: map[string]string{"image": "Ubuntu 22.04 (20220420)"},
	})

	_, dg := r.Resolve(d, func() ([]CatalogEntry, error) { return nil, errTest })
	if dg == nil {
		t.Fatal("expected a diagnostic when fetch fails")
	}
}

// TestCatalogRefHydrateWritesIDUnconditionally is the direct, network-free
// proof the hydration gap is closed: seed a stale/zero id in prior state
// (exactly what a name-only-created server looks like today before any fix),
// call Hydrate, and confirm the id field is now the real value -- Hydrate
// must not gate on whether the field was already set (that gate, in the old
// updateValue helper, was the entire bug).
func TestCatalogRefHydrateWritesIDUnconditionally(t *testing.T) {
	r := testRef()
	d := testRefData(t, "1001", "Debian 12 x64 (20241016)", 0) // id never hydrated, exactly the live bug's shape

	var diags diag.Diagnostics
	r.Hydrate(d, 5794, "Debian 12 x64 (20241016)", &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := d.Get("image_id").(int); got != 5794 {
		t.Fatalf("expected image_id to be hydrated to 5794, got %d", got)
	}
}

// TestCatalogRefHydratePreservesEquivalentUserSpelling generalizes
// setLocationPreserveFormat's existing contract: if state's current name
// already denotes the same entry as the API's value, keep the user's
// spelling rather than overwriting it every read.
func TestCatalogRefHydratePreservesEquivalentUserSpelling(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{{ID: 236, Name: "RDU - Raleigh, NC", IATACode: "rdu"}})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })
	r := CatalogRef{NameField: "location", IDField: "location_id", SameName: sameLocation}
	d := resourceVPC().Data(&terraform.InstanceState{
		ID: "1",
		Attributes: map[string]string{
			"location":    "RDU",
			"location_id": "236",
		},
	})

	var diags diag.Diagnostics
	r.Hydrate(d, 236, "RDU - Raleigh, NC", &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := d.Get("location").(string); got != "RDU" {
		t.Fatalf("expected the user's spelling (RDU) to be preserved, got %q", got)
	}
}

var errTest = &testError{"fetch failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// --- ID/name normalization matrix: mock coverage for cases the live
// catalog has no real fixture for. The live location/OS catalogs have zero
// disabled entries and zero duplicate names/IATA codes, so negative and
// edge cases are exercised in mocks.

// A negative id configured directly (not resolved from a name) is passed
// straight through by Resolve with no validation at all -- it is sent to
// the API as-is; Resolve's job is only to decide "use the configured id, or
// resolve a name," not to range-check it, so any rejection is the API's to
// make.
func TestCatalogRefResolvePassesThroughNegativeIDUnvalidated(t *testing.T) {
	r := testRef()
	for _, id := range []int{-1, -999} {
		d := testRefData(t, "1", "", id)
		got, dg := r.Resolve(d, func() ([]CatalogEntry, error) {
			t.Fatal("fetch should not be called when the id field is configured, even to a negative value")
			return nil, nil
		})
		if dg != nil {
			t.Fatalf("id=%d: unexpected diagnostic: %v", id, *dg)
		}
		if got != id {
			t.Fatalf("expected Resolve to pass id=%d through unvalidated, got %d", id, got)
		}
	}
}

// A configured id of exactly 0 is NOT passed through: Resolve's
// d.GetOk(r.IDField) check can't distinguish "explicitly set to 0" from
// "never configured" (the same zero-value ambiguity behind the tags
// GetOkExists bug and the server package_billing_contract_id null-vs-zero
// handling), so image_id = 0 / location_id = 0 falls through to the
// name-based path and errors if no name is configured either. Documenting
// this rather than asserting it's wrong: 0 is never a legitimate id in
// this API, so treating an explicit 0 as "not configured" is a defensible,
// if easy-to-overlook, consequence of GetOk's design -- not something to
// silently work around without understanding why it happens.
func TestCatalogRefResolveTreatsZeroIDAsUnconfigured(t *testing.T) {
	r := testRef()
	d := testRefData(t, "1", "", 0)
	_, dg := r.Resolve(d, func() ([]CatalogEntry, error) {
		t.Fatal("fetch should not be reached via the id path for id=0 -- GetOk treats it as unconfigured, so Resolve falls to the name check instead")
		return nil, nil
	})
	if dg == nil {
		t.Fatal("expected an error (falls through to the name-required check) for image_id=0 with no name configured")
	}
}

// resolveCatalogEntry has no ambiguity detection: if two entries share a
// name (or share an AltName), it silently returns whichever appears first
// in the slice, with no error and no signal that the match was ambiguous.
// The live location/OS catalogs have zero such collisions, so this has not
// been a live problem. It is a real gap if a future catalog ever does have
// a collision (e.g. two OS images sharing a display name after a rebuild).
// Documenting current behavior, not asserting it's correct.
func TestResolveCatalogEntryPicksFirstMatchOnAmbiguousName(t *testing.T) {
	entries := []CatalogEntry{
		{ID: 100, Name: "Ubuntu 22.04 LTS"},
		{ID: 200, Name: "Ubuntu 22.04 LTS"}, // ambiguous duplicate, real ID differs
	}
	got, ok := resolveCatalogEntry("Ubuntu 22.04 LTS", entries)
	if !ok {
		t.Fatal("expected a match despite the ambiguity")
	}
	if got.ID != 100 {
		t.Fatalf("expected the first entry (id 100) to win silently on an ambiguous name match, got id %d -- if this changed, resolveCatalogEntry's ambiguity handling changed, which may be an improvement (e.g. now erroring) but changes this documented behavior", got.ID)
	}
}

// locationCatalog (and, by the same shape, imageCatalog) never carries the
// API's Disabled flag into CatalogEntry at all -- CatalogEntry has no field
// for it. So a disabled catalog entry, if the live account ever has one, is
// just as resolvable as an enabled one: nothing downstream (Resolve,
// suppressNameDiff, Hydrate) can tell the difference. This is a real,
// confirmed gap (not a hypothetical) -- documented here since the live
// catalog has zero disabled entries to test this against for real. Simulates what
// locationCatalog's translation of a disabled gona.Location produces today:
// a CatalogEntry indistinguishable from an enabled one.
func TestResolveCatalogEntryDoesNotDistinguishDisabledEntries(t *testing.T) {
	// This is exactly the CatalogEntry shape locationCatalog() would produce
	// for a gona.Location{ID: 99, Name: "Stale Datacenter", IATACode: "stl",
	// Disabled: 1} -- Disabled is dropped entirely in the translation.
	entries := []CatalogEntry{
		{ID: 99, Name: "Stale Datacenter", AltNames: []string{"stl"}},
	}
	got, ok := resolveCatalogEntry("stl", entries)
	if !ok || got.ID != 99 {
		t.Fatalf("expected a disabled-shaped entry to still resolve normally (documenting the current gap), got ok=%v id=%d", ok, got.ID)
	}
}
