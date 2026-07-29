package netactuate

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
