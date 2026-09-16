package netactuate

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	tries       = 200
	intervalSec = 1
)

var (
	credentialKeys = []string{"password", "ssh_key_id", "ssh_key"}
	billingKeys    = []string{"package_billing_contract_id", "package_billing_opt_in"}

	hostnameRegex = regexp.MustCompile(fmt.Sprintf("^(%[1]s\\.)*%[1]s$", fmt.Sprintf("(%[1]s|%[1]s%[2]s*%[1]s)", "[a-zA-Z0-9]", "[a-zA-Z0-9\\-]")))

	// serverLocationPair/serverImagePair replace the old locationKeys/
	// imageKeys ExactlyOneOf pairs plus their bespoke suppress*Diff/
	// unchanged*ID/get* helpers -- see catalogref.go.
	serverLocationPair = CatalogRef{
		NameField:     "location",
		IDField:       "location_id",
		NameStateFunc: func(val interface{}) string { return strings.ToUpper(val.(string)) },
		SameName:      sameLocation, // reuse helper.go's existing IATA-aware matcher
	}
	serverImagePair = CatalogRef{
		NameField: "image",
		IDField:   "image_id",
		// SameName nil -> strings.EqualFold, matching the old suppressImageDiff behavior
	}
)

func resourceServer() *schema.Resource {
	locationSchema, locationIDSchema := serverLocationPair.Schemas()
	locationSchema.Description = "Deployment location: display code (e.g. TOR or AMS), full catalog name, or API IATA code. Names are matched case-insensitively."
	imageSchema, imageIDSchema := serverImagePair.Schemas()

	return &schema.Resource{
		CreateContext: resourceServerCreate,
		ReadContext:   resourceServerRead,
		UpdateContext: resourceServerUpdate,
		DeleteContext: resourceServerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceServerImport,
		},
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				ForceNew: false,
				Required: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					if !hostnameRegex.MatchString(i.(string)) {
						return diag.Errorf("%q is not a valid hostname", i)
					}
					return nil
				},
			},
			"plan": {
				Type:     schema.TypeString,
				ForceNew: false,
				Required: true,
			},
			"allow_downsize_reboot": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Single opt-in for disruptive scaling. Defaults to false: in-place upgrades that need no reboot happen automatically, but any scale that DOWNSIZES (reduces mem/cpu/disk) or otherwise requires a REBOOT is rejected during terraform apply before the scale API call instead of silently downsizing/rebooting a running server (a downsize always reboots on this platform). Set true to permit downsize+reboot. This guards against an out-of-band portal scale-up being silently reverted.",
			},
			"package_billing": {
				Type:        schema.TypeString,
				ForceNew:    false,
				Optional:    true,
				Default:     "usage",
				Description: "Not recoverable on import: the API's server-read endpoint has no field reflecting the current billing mode, so this stays at its default (\"usage\") after `terraform import` regardless of the live server's actual billing configuration.",
			},
			"package_billing_opt_in": {
				Type:         schema.TypeString,
				ExactlyOneOf: billingKeys,
				ForceNew:     false,
				Optional:     true,
			},
			"package_billing_contract_id": {
				Type:         schema.TypeString,
				ExactlyOneOf: billingKeys,
				ForceNew:     false,
				Optional:     true,
			},
			"cloud_pool_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Cloud pool ID",
			},
			"vpc_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "VPC ID to deploy the server into",
			},
			"location":    locationSchema,
			"location_id": locationIDSchema,
			"image":       imageSchema,
			"image_id":    imageIDSchema,
			"password": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Sensitive:    true,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
				Description:  "Not recoverable on import: the API has no endpoint that returns which credential was used to build a server, so this stays unset after `terraform import` regardless of the live server's actual configuration. Set it explicitly after import if you intend Terraform to manage it.",
			},
			"ssh_key_id": {
				Type:         schema.TypeInt,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
				Description:  "Not recoverable on import: the API has no endpoint that returns which SSH key is associated with a server, so this stays unset after `terraform import` regardless of the live server's actual configuration. Set it explicitly after import if you intend Terraform to manage it.",
			},
			"ssh_key": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
				Description:  "Not recoverable on import: the API has no endpoint that returns which credential was used to build a server, so this stays unset after `terraform import` regardless of the live server's actual configuration. Set it explicitly after import if you intend Terraform to manage it.",
			},
			"cloud_config": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"user_data_base64": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"primary_ipv4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_ipv6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_reserved_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The private IP address reserved for this server within its VPC.",
			},
			"private_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Alias for vpc_reserved_network for easier private IP discovery.",
			},
			"params": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Additional JSON formatted parameters to be passed to the server creation and management API",
			},
			"tags": {
				Type:             schema.TypeString,
				Optional:         true,
				StateFunc:        tagsStateFunc,
				DiffSuppressFunc: suppressTagsDiff,
				Description: "Tag name or comma-separated tag names to associate with the server, e.g. \"kube\" or \"kube, sjc, cluster\". " +
					"Tags are created automatically if they do not exist. When configured, Terraform is authoritative and forces the " +
					"portal/API tag set to match this value. When omitted, the server's tags are left unmanaged.",
			},
		},
		CustomizeDiff: customdiff.Sequence(
			// Plan diff cleanup: suppress cosmetic no-op plan diffs (same plan,
			// possibly different name casing) so `terraform plan` shows "no
			// changes" and we never scale for nothing. The downgrade/unknown
			// POLICY check is in resourceServerUpdate (apply-time), NOT here:
			// CustomizeDiff also fires on `terraform refresh`, and a policy
			// error during refresh would block the very operation that is meant
			// to surface portal drift. Plan-time will still SHOW the diff (e.g.
			// `~ plan: "VR2x1x25" -> "VR1x1x25"`); apply will reject it.
			func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
				if d.Id() == "" || !d.HasChange("plan") {
					return nil
				}
				oldV, newV := d.GetChange("plan")
				if planChangeKind(oldV.(string), newV.(string)) == "same" {
					return d.Clear("plan")
				}
				return nil
			},
			// "image"/"location" use CatalogRef.NameReallyChanged instead of a
			// raw d.HasChange: CustomizeDiff runs before each field's own
			// DiffSuppressFunc is applied, so a raw HasChange fires even on a
			// purely cosmetic spelling difference (case, IATA vs full name)
			// that suppressNameDiff will suppress in the final plan -- but by
			// then this ComputedIf's "mark it unknown" side effect has
			// already been locked in, showing primary_ipv4/6 as "known after
			// apply" even though nothing will actually change. location_id/
			// image_id/hostname have no DiffSuppressFunc, so raw HasChange is
			// already accurate for them.
			customdiff.ComputedIf("primary_ipv4", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location_id") || serverImagePair.NameReallyChanged(d) || d.HasChange("image_id") || d.HasChange("hostname")
			}),
			customdiff.ComputedIf("primary_ipv6", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location_id") || serverImagePair.NameReallyChanged(d) || d.HasChange("image_id") || d.HasChange("hostname")
			}),
		),
	}
}

func resourceServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	locationId, imageId, diags := getParams(d, c)
	if diags.HasError() {
		return diags
	}
	diags = diag.Diagnostics{}
	req := &gona.CreateServerRequest{
		Plan:                     d.Get("plan").(string),
		Location:                 locationId,
		Image:                    imageId,
		FQDN:                     d.Get("hostname").(string),
		SSHKey:                   d.Get("ssh_key").(string),
		SSHKeyID:                 d.Get("ssh_key_id").(int),
		Password:                 d.Get("password").(string),
		PackageBilling:           d.Get("package_billing").(string),
		PackageBillingContractId: d.Get("package_billing_contract_id").(string),
		CloudConfig:              base64.StdEncoding.EncodeToString([]byte(d.Get("cloud_config").(string))),
		ScriptContent:            base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string))),
		Params:                   d.Get("params").(string), // Handle the new params field
	}

	if userData64, ok := d.GetOk("user_data_base64"); ok {
		req.ScriptContent = userData64.(string)
	}

	if v, ok := d.GetOk("cloud_pool_id"); ok {
		poolID := v.(int)
		req.CloudPoolID = &poolID
	}

	if v, ok := d.GetOk("vpc_id"); ok {
		vpcID := v.(int)
		req.VpcID = &vpcID
	}

	var packageValue = d.Get("package_billing")
	if packageValue == "package" {
		optIn, ok := d.GetOk("package_billing_opt_in")
		if !ok {
			return diag.Errorf("when package_billing is set to package, package_billing_opt_in must be set to yes")
		}

		if optIn.(string) != "yes" {
			return diag.Errorf("when package_billing is set to package, package_billing_opt_in must be set to yes")
		}
	}

	if packageValue == "usage" {
		contractID, ok := d.GetOk("package_billing_contract_id")
		if !ok {
			return diag.Errorf("package_billing_contract_id must be set to your contract ID with NetActuate")
		}

		if len(contractID.(string)) == 0 {
			return diag.Errorf("package_billing_contract_id must be set to your contract ID with NetActuate")
		}
	}

	s, err := c.CreateServer(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(s.ServerID))
	d.Set("params", req.Params) // Store params in the state file

	if _, diags := wait4Status(s.ServerID, "RUNNING", c); diags.HasError() {
		return diags
	}

	server, err := c.GetServer(s.ServerID)
	if err != nil {
		return diag.FromErr(err)
	}
	setValue("primary_ipv4", server.PrimaryIPv4, d, &diags)
	setValue("primary_ipv6", server.PrimaryIPv6, d, &diags)
	setValue("vpc_reserved_network", server.VpcReservedNetwork, d, &diags)
	setValue("private_ip", server.VpcReservedNetwork, d, &diags)

	if td := reconcileServerTags(d, c); td.HasError() {
		return td
	}

	return resourceServerRead(ctx, d, m)
}

// hydrateServerBillingFields fixes an import gap: resourceServerRead never
// read package_billing_contract_id back from the API at all. gona.Server's
// response DOES carry the underlying value, but under the API's actual
// response field name, "contract_id" -- gona.Server was tagged
// `json:"package_billing_contract_id"`, a field the live response never
// actually has (confirmed against the raw API response: its real billing
// field is "contract_id", found by inspecting cloud/server?mbpkgid=...
// directly), so PackageBillingContractId silently unmarshaled to "" every
// time regardless of what the provider did with it. Fixed in gona itself
// (servers.go) alongside this. Without either fix, a server imported via
// schema.ImportStatePassthroughContext (which only sets the ID and calls
// Read) left package_billing_contract_id null forever, and a subsequent
// apply would silently accept whatever config said into state without ever
// calling any API for it (it is not in resourceServerUpdate's
// fieldsToRebuild, and the generic update path makes no call for it).
//
// package_billing (usage vs. package billing mode) has NO live counterpart
// in this response at all -- confirmed by listing every key the endpoint
// actually returns, no "billing"-related key exists besides contract_id.
// Deliberately NOT hydrated: there is nothing true to hydrate it from, and
// writing a fabricated value would be worse than leaving the known gap
// visible. See findings/server_import_gaps.md.
//
// allow_downsize_reboot has no live API concept at all either -- it's a
// pure local policy gate -- but unlike package_billing its Go zero value
// (false) already matches its schema default, and d.Get returns that zero
// value whether or not the underlying map has an entry, so its import gap
// is cosmetic (a one-time harmless diff), not functional. Left untouched.
func hydrateServerBillingFields(d *schema.ResourceData, server gona.Server, diags *diag.Diagnostics) {
	// gona.Server.PackageBillingContractId is an int (the API's "contract_id"
	// is a JSON number), but the Terraform schema field is a string (the
	// create/build request sends it as a string url param, a separate,
	// asymmetric write-side convention) -- convert, and treat 0 (no contract,
	// e.g. an opt_in-billed server) as "" rather than the literal string "0".
	contractID := ""
	if server.PackageBillingContractId != 0 {
		contractID = strconv.Itoa(server.PackageBillingContractId)
	}
	setValue("package_billing_contract_id", contractID, d, diags)
}

func resourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := c.GetServer(id)
	if err != nil {
		if gona.IsNotFound(err) {
			// Deleted out of band. Drop it from state so a plan can recreate it,
			// rather than erroring forever or hydrating zeros over real state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	if server.Installed == 0 {
		setValue("hostname", "", d, &diags)
		serverImagePair.Hydrate(d, 0, "", &diags)
	} else {
		setValue("hostname", server.Name, d, &diags)
		serverImagePair.Hydrate(d, server.OSID, server.OS, &diags)
	}
	setValue("plan", server.Package, d, &diags)
	serverLocationPair.Hydrate(d, server.LocationID, server.Location, &diags)
	hydrateServerBillingFields(d, server, &diags)
	setIntPtr("cloud_pool_id", server.CloudPoolID, d, &diags)
	setIntPtr("vpc_id", server.VpcID, d, &diags)

	setValue("primary_ipv4", server.PrimaryIPv4, d, &diags)
	setValue("primary_ipv6", server.PrimaryIPv6, d, &diags)
	setValue("vpc_reserved_network", server.VpcReservedNetwork, d, &diags)
	setValue("private_ip", server.VpcReservedNetwork, d, &diags)

	setServerTagsState(d, c, &diags)

	return diags
}

// resourceServerImport replaces the former schema.ImportStatePassthroughContext.
// Read's own tag hydration (setServerTagsState, above) deliberately skips
// fetching tags unless state already has a tracked value -- correct for an
// ordinary refresh (see tagsCurrentlyTracked's doc comment), but wrong for
// import, which always starts from a blank state regardless of whether the
// live server actually carries tags. So import needs its own real fetch,
// not a reuse of Read's refresh-oriented gate: this hydrates vpc_id first
// (serverResourceName needs it to pick the right tag-resource type), then
// unconditionally reads and sets the server's actual live tags. See
// findings/server_tags_import_gap.md.
func resourceServerImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, err
	}

	server, err := c.GetServer(id)
	if err != nil {
		return nil, err
	}
	if server.VpcID != nil {
		if err := d.Set("vpc_id", *server.VpcID); err != nil {
			return nil, err
		}
	}

	tags, err := c.GetResourceTags(serverResourceName(d), id)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	if err := d.Set("tags", tagsCSV(names)); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func resourceServerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	fieldsToRebuild := []string{
		"location",
		"location_id",
		"image",
		"image_id",
		"hostname",
		"params",
	}

	rebuildRequired := false
	for _, f := range fieldsToRebuild {
		if d.HasChange(f) {
			rebuildRequired = true
			break
		}
	}

	planChanged := d.HasChange("plan")

	// Autoscale when only the plan changes (no rebuild fields changed)
	if planChanged && !rebuildRequired {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		oldV, newV := d.GetChange("plan")
		oldPlan, newPlan := oldV.(string), newV.(string)
		allowReboot := d.Get("allow_downsize_reboot").(bool)

		// Policy gate: classify the change at APPLY time (not in CustomizeDiff,
		// which would also fire on `terraform refresh` and block drift sync).
		// A disallowed downgrade or unverifiable change returns an error here
		// BEFORE any API call, so the server is never stopped or rebooted.
		switch planChangeKind(oldPlan, newPlan) {
		case "downgrade":
			if !allowReboot {
				return diag.Errorf(
					"refusing to downgrade server plan %q -> %q: this downsizes "+
						"and reboots a running server. The live server may have been "+
						"scaled in the portal. To intentionally downsize, set "+
						"allow_downsize_reboot = true; otherwise change the config "+
						"plan to match the live server.",
					oldPlan, newPlan)
			}
		case "unknown":
			if !allowReboot {
				return diag.Errorf(
					"cannot verify plan change %q -> %q is a no-reboot upgrade (a "+
						"non-VR{mem}x{cpu}x{disk} plan name on one side); refusing to "+
						"auto-change the plan to avoid an accidental downsize/reboot. "+
						"Set allow_downsize_reboot = true to override.",
					oldPlan, newPlan)
			}
		}

		log.Printf("[DEBUG] Scaling server %d to plan %q (allow_downsize_reboot=%v)", id, newPlan, allowReboot)

		jobID, err := c.ScaleServer(id, &gona.ScaleServerRequest{
			PkgName:     newPlan,
			AllowReboot: allowReboot,
		})
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[DEBUG] Scale job started with jobID: %d", jobID)

		if d := wait4JobStatus("scale_vm", jobID, c); d != nil {
			return d
		}

		log.Printf("[DEBUG] Scale job %d completed, waiting for server to be RUNNING", jobID)

		if _, diags := wait4Status(id, "RUNNING", c); diags.HasError() {
			return diags
		}

		if td := reconcileServerTags(d, c); td.HasError() {
			return td
		}

		return resourceServerRead(ctx, d, m)
	}

	// Rebuild on these property changes
	if rebuildRequired {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		oldHost_r, _ := d.GetChange("hostname")
		oldHost := oldHost_r.(string)

		if oldHost != "" {
			// delete
			jobID, err := c.DeleteServer(id, false)
			if err != nil {
				return diag.FromErr(err)
			}
			log.Printf("[DEBUG] Delete job started with jobID: %d", jobID)

			if d := wait4JobStatus("delete", jobID, c); d != nil {
				return d
			}
			log.Printf("[DEBUG] Server deletion job %d completed", jobID)
		}

		// unlink if changing locationID
		unlinkRequired := false

		if d.HasChange("location") {
			oldLoc_r, _ := d.GetChange("location")
			oldLoc := oldLoc_r.(string)
			setValue("location_id", 0, d, &diag.Diagnostics{})
			if oldLoc != "" {
				unlinkRequired = true
				if unlinkRequired {
					err = c.UnlinkServer(id)
					if err != nil {
						return diag.FromErr(err)
					}
				}
			}
		}

		if d.HasChange("location_id") {
			oldLoc_r, _ := d.GetChange("location_id")
			oldLoc := oldLoc_r.(int)
			if oldLoc != 0 {
				unlinkRequired = true
			}

			if unlinkRequired {
				err = c.UnlinkServer(id)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}

		// Get correct build params
		locationId, imageId, diags := getParams(d, c)
		if diags.HasError() {
			return diags
		}
		req := &gona.BuildServerRequest{
			Plan:                     d.Get("plan").(string),
			Location:                 locationId,
			Image:                    imageId,
			FQDN:                     d.Get("hostname").(string),
			SSHKey:                   d.Get("ssh_key").(string),
			SSHKeyID:                 d.Get("ssh_key_id").(int),
			Password:                 d.Get("password").(string),
			PackageBilling:           d.Get("package_billing").(string),
			PackageBillingContractId: d.Get("package_billing_contract_id").(string),
			CloudConfig:              base64.StdEncoding.EncodeToString([]byte(d.Get("cloud_config").(string))),
			ScriptContent:            base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string))),
			Params:                   d.Get("params").(string),
		}

		if userData64, ok := d.GetOk("user_data_base64"); ok {
			req.ScriptContent = userData64.(string)
		}

		if v, ok := d.GetOk("cloud_pool_id"); ok {
			poolID := v.(int)
			req.CloudPoolID = &poolID
		}

		if v, ok := d.GetOk("vpc_id"); ok {
			vpcID := v.(int)
			req.VpcID = &vpcID
		}

		// Rebuild server with potentially updated params
		_, err = c.BuildServer(id, req)
		if err != nil {
			return diag.FromErr(err)
		}

		// Update the params in the state file if they were changed and server rebuilt
		if d.HasChange("params") {
			d.Set("params", req.Params)
		}

		if _, diags := wait4Status(id, "RUNNING", c); diags.HasError() {
			return diags
		}
	}

	if td := reconcileServerTags(d, c); td.HasError() {
		return td
	}

	return resourceServerRead(ctx, d, m)
}

func resourceServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Deleting server with ID: %d", id)

	// Retry delete - the API rejects requests when there are active utility
	// queues (e.g. a build or other operation still in progress).
	const deleteRetries = 10
	const deleteInterval = 20 * time.Second
	var jobID int
	for i := 0; i < deleteRetries; i++ {
		jobID, err = c.DeleteServer(id, true)
		if err == nil {
			break
		}
		if gona.IsNotFound(err) {
			// The server is already gone, which is the goal of a delete. Retrying a
			// not-found nine more times at 20 second intervals turns a success into a
			// three minute wait and then a hard error for a server deleted out of band.
			log.Printf("[DEBUG] Server %d is already gone, treating delete as done", id)
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Delete attempt %d/%d for server %d failed: %s", i+1, deleteRetries, id, err)
		if i == deleteRetries-1 {
			return diag.Errorf("failed to delete server %d after %d attempts: %s", id, deleteRetries, err)
		}
		time.Sleep(deleteInterval)
	}
	log.Printf("[DEBUG] Delete job started with jobID: %d", jobID)

	if d := wait4JobStatus("delete", jobID, c); d != nil {
		return d
	}

	log.Printf("[DEBUG] Server deletion job %d completed", jobID)

	// For VPC servers, unlink the billing package from the location after the
	// delete job completes. This releases the VPC IP reservation; without it
	// the VPC still counts the server as an active member and blocks VPC deletion.
	if vpcID, ok := d.GetOk("vpc_id"); ok && vpcID.(int) != 0 {
		log.Printf("[DEBUG] Server %d was in VPC %d, unlinking to release IP reservation", id, vpcID.(int))
		if err := c.UnlinkServer(id); err != nil {
			return diag.Errorf("server %d deleted but VPC unlink failed: %s", id, err)
		}
	}

	return nil
}

func wait4Status(serverId int, status string, client *gona.Client) (server gona.Server, d diag.Diagnostics) {
	for i := 0; i < tries; i++ {
		server, err := client.GetServer(serverId)

		// Special-case deletion: when waiting for TERMINATED, treat either a real
		// TERMINATED or a blank status (due to the 422/invalid-mbpkgid) as success.
		if status == "TERMINATED" && err == nil && (server.ServerStatus == status || server.ServerStatus == "") {
			return server, nil
		}

		if err != nil && i >= 5 {
			// Retry errors on first few attempts, since sometimes calling GetServer
			// immediately after creating a server returns an error
			// ("mbpkgid must be a valid mbpkgid").
			return server, diag.FromErr(err)
		}

		if err == nil && server.ServerStatus == status {
			return server, nil
		}

		time.Sleep(intervalSec * time.Second)
	}

	return server, diag.Errorf("Timeout of waiting the server to obtain %q status", status)
}

func wait4JobStatus(command string, jobID int, client *gona.Client) diag.Diagnostics {
	for i := 0; i < tries; i++ {
		job, err := client.GetJobStatus(command, jobID)
		if err != nil {
			return diag.FromErr(err)
		}

		if job.Status > 5 {
			return diag.Errorf("Job %s #%d failed with status: %d", command, jobID, job.Status)
		}

		// 5 = completed
		if job.Status == 5 {
			return nil
		}

		time.Sleep(intervalSec * time.Second)
	}

	return diag.Errorf("timeout waiting for job %s #%d to complete", command, jobID)
}

// getParams resolves the server's location and image config (whichever form
// -- name or id -- is set) into concrete API ids, via the shared CatalogRef
// abstraction (catalogref.go). Replaces this file's own former getLocation/
// getImageByName. locationCatalog retains the former server resolver's display
// code aliases alongside full names and API IATA codes.
func getParams(d *schema.ResourceData, client *gona.Client) (int, int, diag.Diagnostics) {
	var diags diag.Diagnostics

	locationId, ld := serverLocationPair.Resolve(d, func() ([]CatalogEntry, error) { return locationCatalog(client) })
	if ld != nil {
		diags = append(diags, *ld)
	}

	imageId, id := serverImagePair.Resolve(d, func() ([]CatalogEntry, error) { return imageCatalog(client) })
	if id != nil {
		diags = append(diags, *id)
	}

	return locationId, imageId, diags
}
