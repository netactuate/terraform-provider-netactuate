package netactuate

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

var locationCatalogCache = struct {
	sync.RWMutex
	index map[string]int
}{}

func setCachedLocationCatalog(locations []gona.Location) {
	index := make(map[string]int, len(locations)*2)
	for _, loc := range locations {
		if loc.Name != "" {
			index[strings.ToLower(loc.Name)] = loc.ID
		}
		if loc.IATACode != "" {
			index[strings.ToLower(loc.IATACode)] = loc.ID
		}
	}

	locationCatalogCache.Lock()
	locationCatalogCache.index = index
	locationCatalogCache.Unlock()
}

func resolveCachedLocationID(location string) (int, bool) {
	locationCatalogCache.RLock()
	defer locationCatalogCache.RUnlock()
	if len(locationCatalogCache.index) == 0 {
		return 0, false
	}
	id, ok := locationCatalogCache.index[strings.ToLower(location)]
	return id, ok
}

func locationIATA(location string) string {
	parts := strings.Fields(location)
	if len(parts) == 0 {
		return ""
	}
	return strings.ToUpper(parts[0])
}

func sameLocation(a, b string) bool {
	ia, ib := locationIATA(a), locationIATA(b)
	return ia != "" && ib != "" && ia == ib
}

func setLocationPreserveFormat(apiLocation string, d *schema.ResourceData, diags *diag.Diagnostics) {
	current := d.Get("location").(string)
	if current != "" && sameLocation(current, apiLocation) {
		// same location - keep the user's format in state
		return
	}
	setValue("location", apiLocation, d, diags)
}

// unchangedLocationID reports whether the configured location resolves to the
// existing resource's current location_id.
func unchangedLocationID(configuredLocation string, d *schema.ResourceData) bool {
	if d == nil || d.Id() == "" {
		return false
	}
	oldID, _ := d.GetChange("location_id")
	currentID := oldID.(int)
	if currentID == 0 {
		return false
	}

	if configuredID, ok := resolveCachedLocationID(configuredLocation); ok {
		return configuredID == currentID
	}

	// Deliberate fallback: without a cached catalog answer, suppress like the
	// old unchanged location_id behavior. Failing to suppress can replace live
	// infrastructure, while suppressing wrongly leaves the resource visibly in
	// its existing location.
	return true
}

// suppressLocationDiff suppresses diffs between two spellings of one location.
// A configuration may pin either the catalog name or the IATA code, since
// getLocationID accepts both, while the API always reports the display name --
// and a display name does not always lead with its IATA code. location is
// ForceNew, so a spelling difference that survives to the plan destroys and
// recreates the resource.
func suppressLocationDiff(k, old, new string, d *schema.ResourceData) bool {
	if new == "" {
		return true
	}
	if sameLocation(old, new) {
		return true
	}
	return unchangedLocationID(new, d)
}

// suppressStorageLocationDiff carries the same contract as
// suppressLocationDiff; storage resources resolve locations through their own
// catalog (see getStorageLocationID) but spell them the same way.
func suppressStorageLocationDiff(k, old, new string, d *schema.ResourceData) bool {
	return suppressLocationDiff(k, old, new, d)
}

// unchangedImageID/suppressImageDiff were netactuate_server's own image/
// image_id guard, superseded by CatalogRef (catalogref.go), which server now
// uses via serverImagePair -- CatalogRef.suppressNameDiff generalizes this
// exact logic (and CatalogRef.Hydrate fixes the hydration gap that defeated
// suppressImageDiff's unchangedImageID fallback in practice).

func setValue(key string, value interface{}, d *schema.ResourceData, diags *diag.Diagnostics) {
	err := d.Set(key, value)
	if err != nil {
		*diags = append(*diags, diag.Diagnostic{Severity: diag.Error, Summary: err.Error()})
	}
}

func setIntPtr(key string, val *int, d *schema.ResourceData, diags *diag.Diagnostics) {
	if val != nil {
		setValue(key, *val, d, diags)
	}
}

func nullableString(val *string) interface{} {
	if val == nil {
		return nil
	}
	return *val
}

func nullableInt(val *int) interface{} {
	if val == nil {
		return nil
	}
	return *val
}

func getLocationID(d *schema.ResourceData, c *gona.Client) (int, *diag.Diagnostic) {
	if v, ok := d.GetOk("location_id"); ok {
		return v.(int), nil
	}

	locationName := d.Get("location").(string)
	if locationName == "" {
		return 0, &diag.Errorf("Please provide a location or location_id")[0]
	}

	locations, err := c.GetLocations()
	if err != nil {
		return 0, &diag.FromErr(err)[0]
	}

	for _, loc := range locations {
		if strings.EqualFold(loc.Name, locationName) || strings.EqualFold(loc.IATACode, locationName) {
			return loc.ID, nil
		}
	}

	return 0, &diag.Errorf("location %q not found", locationName)[0]
}

func getStorageLocationID(d *schema.ResourceData, c *gona.V3Client, v2 *gona.Client) (int, *diag.Diagnostic) {
	if v, ok := d.GetOk("location_id"); ok {
		return v.(int), nil
	}

	locationName := d.Get("location").(string)
	if locationName == "" {
		return 0, &diag.Errorf("Please provide a location or location_id")[0]
	}

	locations, err := c.ListStorageLocations()
	if err != nil {
		return 0, &diag.FromErr(err)[0]
	}

	for _, loc := range locations {
		if strings.EqualFold(loc.Location.Name, locationName) {
			return loc.Location.ID, nil
		}
	}

	if v2 != nil {
		v2Locations, err := v2.GetLocations()
		if err == nil {
			for _, v2Loc := range v2Locations {
				if strings.EqualFold(v2Loc.IATACode, locationName) {
					for _, loc := range locations {
						if strings.EqualFold(loc.Location.Name, v2Loc.Name) {
							return loc.Location.ID, nil
						}
					}
				}
			}
		}
	}

	return 0, &diag.Errorf("storage location %q not found", locationName)[0]
}

// getPackageID resolves a plan name string to its integer package ID via the V2 GetPlans API.
func getPackageID(planName string, c *gona.Client) (int, *diag.Diagnostic) {
	plans, err := c.GetPlans()
	if err != nil {
		d := diag.FromErr(err)[0]
		return 0, &d
	}

	for _, plan := range plans {
		if strings.EqualFold(plan.Name, planName) {
			return plan.ID, nil
		}
	}

	notFound := diag.Errorf("plan %q not found", planName)[0]
	return 0, &notFound
}

// NetActuate VR plan names encode size as VR{MEM}x{CPU}x{DISK}
// (mem in GB, cpu cores, disk in GB), e.g. VR2x4x80.
var planSpecRe = regexp.MustCompile(`(?i)^\s*VR(\d+)x(\d+)x(\d+)\s*$`)

type planSpec struct{ Mem, CPU, Disk int }

// parsePlanSpec extracts (mem,cpu,disk) from a VR{MEM}x{CPU}x{DISK} plan name.
// ok=false for names that don't match (dedicated/custom plans) so callers can
// be conservative rather than guess a direction.
func parsePlanSpec(name string) (planSpec, bool) {
	m := planSpecRe.FindStringSubmatch(name)
	if m == nil {
		return planSpec{}, false
	}
	mem, _ := strconv.Atoi(m[1])
	cpu, _ := strconv.Atoi(m[2])
	disk, _ := strconv.Atoi(m[3])
	return planSpec{Mem: mem, CPU: cpu, Disk: disk}, true
}

// planChangeKind classifies a plan change from old (live) to new (config):
//   - "same":      identical name or identical mem/cpu/disk
//   - "upgrade":    every dimension >= and at least one >
//   - "downgrade":  ANY dimension decreases (a shrink in mem/cpu/disk - disk
//     especially - is destructive/reboot-prone, so a mixed change counts here)
//   - "unknown":    a non-VR plan name on either side (cannot prove it's an upgrade)
func planChangeKind(oldName, newName string) string {
	if strings.EqualFold(strings.TrimSpace(oldName), strings.TrimSpace(newName)) {
		return "same"
	}
	o, ok1 := parsePlanSpec(oldName)
	n, ok2 := parsePlanSpec(newName)
	if !ok1 || !ok2 {
		return "unknown"
	}
	if o == n {
		return "same"
	}
	if n.Mem < o.Mem || n.CPU < o.CPU || n.Disk < o.Disk {
		return "downgrade"
	}
	return "upgrade"
}

// NetActuate tag resource-type tokens returned by the tag API.
const (
	resourceNameVirtualServer    = "virtual-server"
	resourceNameVirtualServerVPC = "virtual-server-vpc"
)

// serverResourceName returns the tag resource_name token for a server. A server
// that is a VPC member is "virtual-server-vpc"; a standalone server is
// "virtual-server".
func serverResourceName(d *schema.ResourceData) string {
	if v, ok := d.GetOk("vpc_id"); ok && v.(int) != 0 {
		return resourceNameVirtualServerVPC
	}
	return resourceNameVirtualServer
}

// getOrCreateTagIDByName resolves a tag name to its id, creating the tag if it
// does not exist yet. Tags are global/shared; an unknown name is created on
// demand so the server resource can be the only Terraform-facing tag API.
func getOrCreateTagIDByName(name string, c *gona.Client) (int, *diag.Diagnostic) {
	if id, found, dg := findTagIDByName(name, c); dg != nil {
		return 0, dg
	} else if found {
		return id, nil
	}

	// Not found - create it (name only; the API assigns default icon/color).
	created, cerr := c.CreateTag(&gona.CreateTagRequest{Name: name})
	if cerr != nil {
		dd := diag.FromErr(cerr)[0]
		return 0, &dd
	}
	if created != nil && created.ID != 0 {
		return created.ID, nil
	}

	// Create response carried no id, or a concurrent create won the race -
	// re-resolve by name.
	if id, found, dg := findTagIDByName(name, c); dg != nil {
		return 0, dg
	} else if found {
		return id, nil
	}
	nf := diag.Errorf("tag %q could not be created or resolved", name)[0]
	return 0, &nf
}

func findTagIDByName(name string, c *gona.Client) (int, bool, *diag.Diagnostic) {
	tags, err := c.GetTags()
	if err != nil {
		dd := diag.FromErr(err)[0]
		return 0, false, &dd
	}
	for _, t := range tags {
		if strings.EqualFold(t.Name, name) {
			return t.ID, true, nil
		}
	}
	return 0, false, nil
}

func splitTags(value string) []string {
	parts := strings.Split(value, ",")
	names := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	return names
}

func tagsCSV(names []string) string {
	return strings.Join(splitTags(strings.Join(names, ",")), ", ")
}

func tagsStateFunc(val interface{}) string {
	return strings.Join(splitTags(val.(string)), ", ")
}

func suppressTagsDiff(k, old, new string, d *schema.ResourceData) bool {
	return tagsStateFunc(old) == tagsStateFunc(new)
}

// tagsConfiguredInRequest reports whether the CURRENT apply's raw config
// actually declares tags, via GetRawConfig rather than the deprecated
// d.GetOkExists. GetOkExists reads through the merged
// state/config/diff/set stack and can't distinguish "explicitly configured
// as an empty string" from "omitted, therefore diffed down to the schema's
// zero value": once a diff exists for tags (e.g. because it was configured
// before and the config line was removed), GetOkExists reports exists=true
// regardless, which broke the "unmanaged when omitted" contract below --
// confirmed live: removing tags after it was set deleted every live tag
// instead of leaving them alone. GetRawConfig reflects the actual submitted
// HCL, so an omitted attribute is a true cty null here, not a schema
// zero-value in disguise.
//
// Used only where a real diff/RawConfig is guaranteed to be attached:
// Create/Update (reconcileServerTags, deciding the desired tag set to
// enforce). Do NOT use this during Read/refresh -- see tagsCurrentlyTracked.
func tagsConfiguredInRequest(d *schema.ResourceData) bool {
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() || !raw.Type().HasAttribute("tags") {
		return false
	}
	return !raw.GetAttr("tags").IsNull()
}

// tagsCurrentlyTracked reports whether tags is being managed as of the last
// known state -- the signal setServerTagsState (Read) needs, since a plain
// refresh (terraform plan's refresh step, or `terraform refresh`) never has
// a new config/diff to consult at all: confirmed against the vendored SDK's
// ReadResource handler (helper/schema/grpc_provider.go), which builds the
// ResourceData from prior state only and attaches no RawConfig, so
// GetRawConfig() always returns a NullVal there regardless of what the
// user's config says. Using tagsConfiguredInRequest here would make Read
// treat tags as unmanaged on every refresh, unconditionally -- confirmed
// live: an out-of-band (portal) tag added while tags was genuinely
// configured and unchanged went completely undetected by `terraform plan`
// ("No changes"), because Read stopped refreshing state's tags from live
// reality at all. A non-empty persisted tags value means tags was
// configured as of the last apply, so keep syncing it with live reality so
// a later plan can compare the refreshed value against whatever the new
// config says (including catching tags having been removed from config
// since). This can't distinguish "genuinely never managed" from "managed
// but explicitly emptied," an acceptable narrow tradeoff for the refresh
// path only -- actual enforcement (reconcileServerTags, via
// tagsConfiguredInRequest) is unaffected and stays precise.
func tagsCurrentlyTracked(d *schema.ResourceData) bool {
	return d.Get("tags").(string) != ""
}

// serverDesiredTagNames returns the configured tag names and whether tags are
// managed. They are unmanaged only when tags is omitted. If tags is configured,
// even as an empty string, Terraform forces the portal/API tag set to match it.
func serverDesiredTagNames(d *schema.ResourceData) ([]string, bool) {
	if !tagsConfiguredInRequest(d) {
		return nil, false
	}
	return splitTags(d.Get("tags").(string)), true
}

// reconcileServerTags makes the server's tag set match tags (authoritative)
// when tags are managed; it is a no-op otherwise. Safe to call on every
// create/update - it also re-applies tags after a rebuild wipes them.
func reconcileServerTags(d *schema.ResourceData, c *gona.Client) diag.Diagnostics {
	desired, managed := serverDesiredTagNames(d)
	if !managed {
		return nil
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	rn := serverResourceName(d)

	desiredIDs := make(map[int]bool, len(desired))
	for _, name := range desired {
		tid, dg := getOrCreateTagIDByName(name, c)
		if dg != nil {
			return diag.Diagnostics{*dg}
		}
		desiredIDs[tid] = true
	}

	current, err := c.GetResourceTags(rn, id)
	if err != nil {
		return diag.FromErr(err)
	}
	currentIDs := make(map[int]bool, len(current))
	for _, t := range current {
		currentIDs[t.ID] = true
	}

	for tid := range desiredIDs {
		if !currentIDs[tid] {
			if err := c.AssignTagResource(tid, rn, id); err != nil {
				return diag.Errorf("failed to assign tag %d to %s/%d: %s", tid, rn, id, err)
			}
		}
	}
	for tid := range currentIDs {
		if !desiredIDs[tid] {
			if err := c.RemoveTagResource(tid, rn, id); err != nil {
				return diag.Errorf("failed to remove tag %d from %s/%d: %s", tid, rn, id, err)
			}
		}
	}
	return nil
}

// setServerTagsState writes tags back from the API, but only when
// tagsCurrentlyTracked (i.e. state already has a real tags value from a
// prior apply). When unmanaged it leaves tag state untouched so portal tags
// do not create a perpetual diff -- see tagsCurrentlyTracked's doc comment
// for why this must NOT use tagsConfiguredInRequest/GetRawConfig here.
func setServerTagsState(d *schema.ResourceData, c *gona.Client, diags *diag.Diagnostics) {
	if !tagsCurrentlyTracked(d) {
		return
	}
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return
	}
	tags, err := c.GetResourceTags(serverResourceName(d), id)
	if err != nil {
		return
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	setValue("tags", tagsCSV(names), d, diags)
}
