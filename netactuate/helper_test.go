package netactuate

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func vpcLocationData(t *testing.T, id, location string, locationID int) *schema.ResourceData {
	t.Helper()
	return resourceVPC().Data(&terraform.InstanceState{
		ID: id,
		Attributes: map[string]string{
			"location":    location,
			"location_id": strconv.Itoa(locationID),
		},
	})
}

// A location whose display name leads with its IATA code round-trips through
// sameLocation, so this path must keep working without consulting location_id.
func TestSuppressLocationDiffMatchesLeadingIATACode(t *testing.T) {
	if !suppressLocationDiff("location", "YYZ - Toronto, ON", "YYZ", nil) {
		t.Fatal("expected a display name leading with the configured IATA code to suppress the diff")
	}
}

// getLocationID resolves a configured location against either the catalog's
// name or its IATA code, so a config may legitimately pin "YYZ" while the API
// reports a display name that does not lead with it. location is ForceNew, so
// failing to suppress that diff destroys and recreates the resource -- for a
// VPC, rotating the very egress addresses it exists to hold stable. An
// unchanged location_id proves the location itself did not move.
func TestSuppressLocationDiffTrustsUnchangedLocationID(t *testing.T) {
	d := vpcLocationData(t, "331", "YYZ2 - Toronto, ON", 42)

	if !suppressLocationDiff("location", "YYZ2 - Toronto, ON", "YYZ", d) {
		t.Fatal("expected an unchanged location_id to suppress a display-name/IATA-code mismatch")
	}
}

// Without an id there is no prior location_id to compare, so a mismatch is a
// real change and must still force replacement.
func TestSuppressLocationDiffReportsMismatchOnCreate(t *testing.T) {
	d := vpcLocationData(t, "", "", 0)

	if suppressLocationDiff("location", "YYZ2 - Toronto, ON", "LGA", d) {
		t.Fatal("expected a location mismatch with no prior id to produce a diff")
	}
}

// A zero location_id carries no information, so it must not be read as proof
// the location is unchanged.
func TestSuppressLocationDiffIgnoresZeroLocationID(t *testing.T) {
	d := vpcLocationData(t, "331", "YYZ2 - Toronto, ON", 0)

	if suppressLocationDiff("location", "YYZ2 - Toronto, ON", "LGA", d) {
		t.Fatal("expected a zero location_id to produce a diff")
	}
}

func TestSuppressLocationDiffReportsDifferentResolvedLocation(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{
		{ID: 42, Name: "YYZ2 - Toronto, ON", IATACode: "YYZ"},
		{ID: 21, Name: "LAX - Los Angeles, CA", IATACode: "LAX"},
	})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })

	d := vpcLocationData(t, "331", "YYZ2 - Toronto, ON", 42)

	if suppressLocationDiff("location", "YYZ2 - Toronto, ON", "LAX - Los Angeles, CA", d) {
		t.Fatal("expected a catalog-resolved location mismatch to produce a diff")
	}
}

func TestSuppressLocationDiffSuppressesResolvedIATACode(t *testing.T) {
	setCachedLocationCatalog([]gona.Location{
		{ID: 42, Name: "YYZ2 - Toronto, ON", IATACode: "YYZ"},
	})
	t.Cleanup(func() { setCachedLocationCatalog(nil) })

	d := rebuildDiffWithResource(t, resourceVPC(),
		map[string]string{
			"label":       "example",
			"description": "example",
			"location":    "YYZ2 - Toronto, ON",
			"location_id": "42",
		},
		map[string]interface{}{
			"label":       "example",
			"description": "example",
			"location":    "YYZ",
		},
	)

	if !suppressLocationDiff("location", "YYZ2 - Toronto, ON", "YYZ", d) {
		t.Fatal("expected a configured IATA code resolving to the current location_id to suppress the diff")
	}
}

func TestSuppressLocationDiffSuppressesWhenCatalogEmpty(t *testing.T) {
	setCachedLocationCatalog(nil)

	d := vpcLocationData(t, "331", "YYZ2 - Toronto, ON", 42)

	if !suppressLocationDiff("location", "YYZ2 - Toronto, ON", "LAX - Los Angeles, CA", d) {
		t.Fatal("expected an empty catalog to use the deliberate suppression fallback")
	}
}

// suppressImageDiff and its coverage (formerly here) were superseded by
// CatalogRef.suppressNameDiff (catalogref.go) -- see catalogref_test.go for
// the generalized equivalent of TestSuppressImageDiffAllowsImageIDOnlyConfig/
// MatchesCaseInsensitively/TrustsUnchangedImageID/ReportsMismatchOnCreate/
// IgnoresZeroImageID.

func TestSplitTags(t *testing.T) {
	got := splitTags(" kube, sjc, , cluster, kube ,SJC ")
	want := []string{"cluster", "kube", "sjc"}

	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tag %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestTagsStateFunc(t *testing.T) {
	got := tagsStateFunc(" kube, sjc, , cluster ")
	want := "cluster, kube, sjc"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSuppressTagsDiff(t *testing.T) {
	if !suppressTagsDiff("tags", "kube, sjc, cluster", "cluster, kube, sjc", nil) {
		t.Fatal("expected equivalent comma-separated tag sets to suppress diff")
	}
	if suppressTagsDiff("tags", "kube, sjc", "kube, ams", nil) {
		t.Fatal("expected different comma-separated tag sets to produce a diff")
	}
}

// serverTagsConfigured must not use the deprecated
// d.GetOkExists("tags"), which reads through the merged state/config/diff/set
// stack and reports exists=true whenever ANY diff exists for the attribute --
// including the diff produced by removing a previously-configured tags line
// (old: real value, new: the schema's zero value ""), because it can't tell
// that apart from an explicit empty string. Confirmed live: doing exactly
// that deleted every tag from a real server instead of leaving it
// "unmanaged" as documented. Fixed by reading d.GetRawConfig() instead, which
// reflects the actual submitted HCL (a true cty null for an omitted
// attribute, not a schema zero-value in disguise). These tests drive the
// real SDK diff engine (rebuildDiff, resource_server_test.go) rather than
// hand-building a fixture, so they exercise the exact mechanism that broke.
func TestServerTagsConfiguredFalseWhenOmittedAfterPreviouslySet(t *testing.T) {
	d := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001",
			"tags":       "example-tag", // previously configured
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
			// tags omitted entirely -- the exact transition that broke
		},
	)

	if tagsConfiguredInRequest(d) {
		t.Fatal("expected tagsConfiguredInRequest to be false when tags is omitted from config, even though a prior value existed in state -- this is the bug under test (GetOkExists reported true here)")
	}
	desired, managed := serverDesiredTagNames(d)
	if managed {
		t.Fatalf("expected tags to be unmanaged (omitted), got managed=true desired=%v", desired)
	}
}

// Uses rebuildDiffWithRawConfigTags, not plain rebuildDiff: tags is
// unchanged (old==new), so Resource.Diff() produces no diff entry for it at
// all, and GetRawConfig() would fall back to a NullVal without a manually
// attached RawConfig -- exactly why the real fix needs verifying against a
// RawConfig that actually reflects "tags present in HCL," not inferred from
// there being a diff.
func TestServerTagsConfiguredTrueWhenExplicitlySet(t *testing.T) {
	d := rebuildDiffWithRawConfigTags(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001", "tags": "example-tag",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
			"tags": "example-tag", // still explicitly configured, unchanged
		},
		true, "example-tag",
	)

	if !tagsConfiguredInRequest(d) {
		t.Fatal("expected tagsConfiguredInRequest to be true when tags is explicitly configured")
	}
	desired, managed := serverDesiredTagNames(d)
	if !managed || len(desired) != 1 || desired[0] != "example-tag" {
		t.Fatalf("expected managed=true desired=[example-tag], got managed=%v desired=%v", managed, desired)
	}
}

func TestServerTagsConfiguredFalseOnFreshCreateWithNoTags(t *testing.T) {
	d := rebuildDiff(t,
		map[string]string{}, // no prior state at all -- a fresh create
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
			// tags never configured
		},
	)

	if tagsConfiguredInRequest(d) {
		t.Fatal("expected tagsConfiguredInRequest to be false on a fresh create with no tags configured")
	}
}

// tagsCurrentlyTracked regression: using tagsConfiguredInRequest (i.e.
// GetRawConfig) for the READ-time decision was a real bug found live, not
// just a hypothetical -- GetRawConfig() is only ever populated when a real
// diff is attached (Create/Update), never during a plain refresh (confirmed
// against the vendored SDK's ReadResource handler, which builds
// ResourceData from prior state only). Using it in setServerTagsState made
// Read treat tags as unmanaged on EVERY refresh, unconditionally --
// confirmed live: an out-of-band (portal) tag added while tags was
// genuinely configured and unchanged went completely undetected by
// `terraform plan` ("No changes"). rebuildDiff's ResourceData has no
// RawConfig attached either (same reason rebuildDiffWithRawConfigTags
// exists), which is exactly why these assert against prior state via
// tagsCurrentlyTracked instead of needing a real diff at all.
func TestTagsCurrentlyTrackedTrueWhenStateHasRealValue(t *testing.T) {
	d := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001", "tags": "example-tag",
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
			"tags": "example-tag",
		},
	)

	if !tagsCurrentlyTracked(d) {
		t.Fatal("expected tagsCurrentlyTracked to be true when prior state already has a real tags value -- this is what setServerTagsState needs during a plain refresh, where GetRawConfig is never populated")
	}
}

func TestTagsCurrentlyTrackedFalseWhenStateHasNoValue(t *testing.T) {
	d := rebuildDiff(t,
		map[string]string{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location": "RDU", "location_id": "236",
			"image": "Debian 12 x64 (20241016)", "image_id": "5794",
			"ssh_key_id": "2001",
			// tags never tracked
		},
		map[string]interface{}{
			"hostname": "server.example.test", "plan": "VR1x1x25",
			"location_id": 236, "image_id": 5794, "ssh_key_id": 2001,
		},
	)

	if tagsCurrentlyTracked(d) {
		t.Fatal("expected tagsCurrentlyTracked to be false when prior state never had a tags value")
	}
}
