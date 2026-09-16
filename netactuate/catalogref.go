package netactuate

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// CatalogEntry is the common shape every name/id catalog this provider
// resolves configuration against is normalized into. AltNames holds any
// other spelling a config is allowed to pin instead of Name (e.g. a
// location's IATA code); it is nil for catalogs with no alternate spelling.
type CatalogEntry struct {
	ID       int
	Name     string
	AltNames []string
}

// CatalogFetcher fetches and normalizes one catalog. Each resource supplies
// its own closure bound to its own already-authenticated client, so
// CatalogRef itself never needs to know about *gona.Client vs *gona.V3Client.
type CatalogFetcher func() ([]CatalogEntry, error)

// resolveCatalogEntry finds the entry whose Name or any AltName matches
// configured, case-insensitively. This is the one matching rule for every
// name/id pair in the provider, so every resolver matches identically.
func resolveCatalogEntry(configured string, entries []CatalogEntry) (CatalogEntry, bool) {
	for _, e := range entries {
		if strings.EqualFold(e.Name, configured) {
			return e, true
		}
		for _, alt := range e.AltNames {
			if strings.EqualFold(alt, configured) {
				return e, true
			}
		}
	}
	return CatalogEntry{}, false
}

// locationCatalog normalizes the V2 locations catalog. Preserve the leading
// display code accepted by v0.3.0 as well as the API's IATA code: they can differ
// ("TOR - Toronto, CA" uses "yyz", "AMS - Amsterdam, NL" uses "ams2").
func locationCatalog(c *gona.Client) ([]CatalogEntry, error) {
	locs, err := c.GetLocations()
	if err != nil {
		return nil, err
	}
	entries := make([]CatalogEntry, len(locs))
	for i, l := range locs {
		entries[i] = CatalogEntry{ID: l.ID, Name: l.Name, AltNames: []string{l.IATACode, locationIATA(l.Name)}}
	}
	return entries, nil
}

// imageCatalog normalizes the V2 OS catalog. Images have no alternate
// spelling.
func imageCatalog(c *gona.Client) ([]CatalogEntry, error) {
	oss, err := c.GetOSs()
	if err != nil {
		return nil, err
	}
	entries := make([]CatalogEntry, len(oss))
	for i, o := range oss {
		entries[i] = CatalogEntry{ID: o.ID, Name: o.Os}
	}
	return entries, nil
}

// CatalogRef describes one name/id catalog-resolved field pair: a string
// name field and an int id field, ExactlyOneOf each other, both resolving to
// the same live-API catalog entry. It is pure data -- no client, no
// schema.ResourceData -- so one value can be shared between Schemas()
// (called once, at provider-registration time, no client available yet) and
// Resolve()/Hydrate() (called per-request, with whatever client/apiID/
// apiName the caller already has in hand).
type CatalogRef struct {
	NameField string
	IDField   string
	ForceNew  bool

	// NameStateFunc is optional (e.g. strings.ToUpper for location).
	NameStateFunc schema.SchemaStateFunc

	// SameName decides whether two spellings denote the same catalog entry
	// for DiffSuppressFunc purposes. nil defaults to strings.EqualFold. Pass
	// an IATA-aware matcher (e.g. helper.go's sameLocation) to reuse
	// existing, already-tested equivalence logic instead of reimplementing
	// it per pair.
	SameName func(old, new string) bool
}

func (r CatalogRef) exactlyOneOf() []string { return []string{r.NameField, r.IDField} }

func (r CatalogRef) sameName(a, b string) bool {
	if r.SameName != nil {
		return r.SameName(a, b)
	}
	return strings.EqualFold(a, b)
}

// unchangedID reports whether an existing resource's id field was
// EXPLICITLY reconfigured to the same value it already had, which proves
// the underlying catalog entry did not change however differently the API
// and the configuration spell its name. Generalizes
// unchangedLocationID/unchangedImageID.
//
// Requires the id field to actually appear (non-null) in d.GetRawConfig()
// before trusting oldID == newID. This is not optional: when the id field
// is simply never configured at all (the ordinary case for any name-only
// config), d.GetChange(r.IDField) trivially reports old == new anyway,
// because nothing in the diff ever touches it -- that equality is NOT
// evidence the id was reconfirmed, only that it was never examined. Changing
// `image` alone to a genuinely different, unrelated image (Ubuntu vs Debian,
// with image_id left unconfigured, the normal way a user edits a name-only config) was
// silently swallowed as "No changes" -- unchangedID incorrectly "trusted"
// image_id as unchanged solely because it was untouched, permanently
// masking any name-only image/location change from ever taking effect.
// This predates CatalogRef entirely (the original unchangedLocationID/
// unchangedImageID had the identical flaw) and was never caught because
// every existing test exercised it via a hand-built ResourceData
// (.Data(state) or hand-fed old/new parameters) that never had a real
// diff or RawConfig attached, so it could not have revealed this.
//
// GetRawConfig() is reliable here specifically because this runs during
// actual diff computation (Create/Update/plan), not a bare refresh: the
// vendored SDK's PlanResourceChange handler (helper/schema/grpc_provider.go)
// sets `priorState.RawConfig` before computing the diff, and confirmed via
// direct experiment that GetRawConfig()'s state-level fallback picks that
// up correctly -- unlike ReadResource, which never sets any RawConfig at
// all (see tagsCurrentlyTracked's doc comment for that contrasting case).
func (r CatalogRef) unchangedID(d *schema.ResourceData) bool {
	if d == nil || d.Id() == "" {
		return false
	}
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() || !raw.Type().HasAttribute(r.IDField) || raw.GetAttr(r.IDField).IsNull() {
		return false
	}
	oldID, newID := d.GetChange(r.IDField)
	return oldID.(int) != 0 && oldID == newID
}

// suppressNameDiff generalizes suppressLocationDiff/suppressImageDiff.
func (r CatalogRef) suppressNameDiff(k, old, new string, d *schema.ResourceData) bool {
	if new == "" {
		return true
	}
	if r.sameName(old, new) {
		return true
	}
	return r.unchangedID(d)
}

// NameReallyChanged mirrors suppressNameDiff's logic but is driven from a
// *schema.ResourceDiff (CustomizeDiff's type), not *schema.ResourceData.
// CustomizeDiff callbacks (e.g. resource_server.go's ComputedIf rules for
// primary_ipv4/primary_ipv6) run BEFORE the name field's own
// DiffSuppressFunc is applied, so a raw d.HasChange(r.NameField) fires even
// on a purely cosmetic spelling difference (case, IATA vs full name) that
// will end up suppressed in the final plan -- but the CustomizeDiff-stage
// side effect (marking a field Computed) has already been locked in by
// then and survives into the final plan as a misleading "known after
// apply", even though nothing will actually change. A case-only image spelling change can show primary_ipv4/6 as
// unknown, while the real apply-time d.HasChange("image") (post-
// suppression) and rebuildRequired were both false -- i.e. apply would
// have been a true no-op. Use this instead of a raw HasChange check in any
// CustomizeDiff callback that cares whether a name field's change is real.
// (Duplicates suppressNameDiff/unchangedID's logic rather than sharing it:
// ResourceDiff and ResourceData aren't the same type, and generalizing
// unchangedID to accept both via an interface risks the classic Go
// nil-interface-vs-nil-pointer footgun, since suppressNameDiff is called
// directly with a literal nil *schema.ResourceData in tests.)
func (r CatalogRef) NameReallyChanged(d *schema.ResourceDiff) bool {
	if !d.HasChange(r.NameField) {
		return false
	}
	oldV, newV := d.GetChange(r.NameField)
	oldName, _ := oldV.(string)
	newName, _ := newV.(string)
	if newName == "" {
		return false
	}
	if r.sameName(oldName, newName) {
		return false
	}
	if d.Id() == "" {
		return true
	}
	oldID, newID := d.GetChange(r.IDField)
	oid, _ := oldID.(int)
	nid, _ := newID.(int)
	return !(oid != 0 && oid == nid)
}

// Schemas produces the two *schema.Schema entries for this pair: Optional,
// ExactlyOneOf each other, DiffSuppressFunc on the name side, Computed on
// the id side. No client or live data involved -- safe to call once from a
// resource's schema map literal.
func (r CatalogRef) Schemas() (name, id *schema.Schema) {
	exactly := r.exactlyOneOf()
	name = &schema.Schema{
		Type:             schema.TypeString,
		Optional:         true,
		ForceNew:         r.ForceNew,
		ExactlyOneOf:     exactly,
		StateFunc:        r.NameStateFunc,
		DiffSuppressFunc: r.suppressNameDiff,
	}
	id = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Computed:     true,
		ForceNew:     r.ForceNew,
		ExactlyOneOf: exactly,
	}
	return name, id
}

// Resolve turns whichever of the pair's two fields is configured into a
// concrete catalog id: the id field directly if set, otherwise the name
// field resolved against fetch's catalog. Replaces per-resource resolvers
// like getLocationID/getStorageLocationID/getPackageID and resource_server.go's
// own getLocation/getImageByName/getParams.
func (r CatalogRef) Resolve(d *schema.ResourceData, fetch CatalogFetcher) (int, *diag.Diagnostic) {
	if v, ok := d.GetOk(r.IDField); ok {
		return v.(int), nil
	}
	name, _ := d.Get(r.NameField).(string)
	if name == "" {
		e := diag.Errorf("please provide %s or %s", r.NameField, r.IDField)[0]
		return 0, &e
	}
	entries, err := fetch()
	if err != nil {
		e := diag.FromErr(err)[0]
		return 0, &e
	}
	if entry, ok := resolveCatalogEntry(name, entries); ok {
		return entry.ID, nil
	}
	e := diag.Errorf("%s %q not found", r.NameField, name)[0]
	return 0, &e
}

// Hydrate unconditionally writes both fields from the authoritative API
// response on every Read -- the fix for the location_id/image_id hydration
// gap: setValue, not the old updateValue, on the id side, always. apiName's
// spelling is preserved in state when it is equivalent to what's already
// there (see sameName), generalizing setLocationPreserveFormat to any pair.
func (r CatalogRef) Hydrate(d *schema.ResourceData, apiID int, apiName string, diags *diag.Diagnostics) {
	setValue(r.IDField, apiID, d, diags)
	current, _ := d.Get(r.NameField).(string)
	if current != "" && r.sameName(current, apiName) {
		return
	}
	setValue(r.NameField, apiName, d, diags)
}
