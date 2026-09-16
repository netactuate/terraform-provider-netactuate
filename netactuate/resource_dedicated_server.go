package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceDedicatedServerBuy() *schema.Resource {
	return dedicatedServerResource(resourceDedicatedServerBuyCreate, "Purchases a dedicated server without deploying an operating system.")
}

func resourceDedicatedServerDeployment() *schema.Resource {
	return dedicatedServerResource(resourceDedicatedServerDeploymentCreate, "Deploys an operating system on an existing dedicated server package.")
}

func resourceDedicatedServerBuyBuild() *schema.Resource {
	return dedicatedServerResource(resourceDedicatedServerBuyBuildCreate, "Purchases and deploys a dedicated server.")
}

func dedicatedServerResource(create schema.CreateContextFunc, description string) *schema.Resource {
	return &schema.Resource{
		CreateContext: create,
		ReadContext:   resourceDedicatedServerRead,
		DeleteContext: resourceDedicatedServerDelete,
		Description:   description,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: dedicatedServerResourceSchema(),
	}
}

func dedicatedServerResourceSchema() map[string]*schema.Schema {
	s := dedicatedServerBuildInputSchema()
	s["device_id"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		ForceNew:    true,
		Description: "Dedicated device ID to purchase.",
	}
	s["mbpkgid"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
		ForceNew:    true,
		Description: "Dedicated server package ID.",
	}
	for key, value := range dedicatedServerReadSchema() {
		if key != "mbpkgid" {
			s[key] = value
		}
	}
	s["build_id"] = &schema.Schema{
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "Build job ID returned by the API.",
	}
	s["build_status"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Build status returned by the API.",
	}
	return s
}

func dedicatedServerBuildInputSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"fqdn": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Fully qualified domain name to assign during deployment.",
		},
		"profile": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Dedicated OS profile ID to deploy.",
		},
		"disklayout": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "Dedicated disk layout ID.",
		},
		"root_password": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			ForceNew:    true,
			Description: "Root password to set during deployment.",
		},
		"ssh_key": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "SSH public key content to install during deployment.",
		},
		"ssh_key_id": {
			Type:        schema.TypeInt,
			Optional:    true,
			ForceNew:    true,
			Description: "SSH key ID to install during deployment.",
		},
		"build_script": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Build script content to run during deployment.",
		},
	}
}

func resourceDedicatedServerBuyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	deviceID := d.Get("device_id").(int)
	if deviceID == 0 {
		return diag.Errorf("device_id is required")
	}

	build, err := c.BuyDedicatedServer(deviceID)
	if err != nil {
		return diag.FromErr(err)
	}
	setDedicatedBuildState(build, d)
	return resourceDedicatedServerRead(ctx, d, m)
}

func resourceDedicatedServerDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbPkgID := d.Get("mbpkgid").(int)
	if mbPkgID == 0 {
		return diag.Errorf("mbpkgid is required")
	}

	build, err := c.DeployDedicatedServer(mbPkgID, dedicatedServerBuildRequest(d))
	if err != nil {
		return diag.FromErr(err)
	}
	setDedicatedBuildState(build, d)
	return resourceDedicatedServerRead(ctx, d, m)
}

func resourceDedicatedServerBuyBuildCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	deviceID := d.Get("device_id").(int)
	if deviceID == 0 {
		return diag.Errorf("device_id is required")
	}

	build, err := c.BuyAndDeployDedicatedServer(deviceID, dedicatedServerBuildRequest(d))
	if err != nil {
		return diag.FromErr(err)
	}
	setDedicatedBuildState(build, d)
	return resourceDedicatedServerRead(ctx, d, m)
}

func resourceDedicatedServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbPkgID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	server, err := c.GetMetal(mbPkgID)
	if err != nil {
		if isNotFoundOrUnprocessable(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if server.Canceling == 1 {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics
	for key, value := range flattenDedicatedServer(server) {
		setValue(key, value, d, &diags)
	}
	return diags
}

func resourceDedicatedServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbPkgID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteDedicatedServer(mbPkgID, nil); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func dedicatedServerBuildRequest(d *schema.ResourceData) *gona.DedicatedServerBuildRequest {
	req := &gona.DedicatedServerBuildRequest{}
	if v, ok := d.GetOk("fqdn"); ok {
		req.FQDN = v.(string)
	}
	if v, ok := d.GetOk("profile"); ok {
		req.Profile = v.(int)
	}
	if v, ok := d.GetOk("disklayout"); ok {
		value := v.(int)
		req.DiskLayout = &value
	}
	if v, ok := d.GetOk("root_password"); ok {
		value := v.(string)
		req.RootPassword = &value
	}
	if v, ok := d.GetOk("ssh_key"); ok {
		value := v.(string)
		req.SSHKey = &value
	}
	if v, ok := d.GetOk("ssh_key_id"); ok {
		value := v.(int)
		req.SSHKeyID = &value
	}
	if v, ok := d.GetOk("build_script"); ok {
		value := v.(string)
		req.BuildScript = &value
	}
	return req
}

func setDedicatedBuildState(build gona.MetalBuild, d *schema.ResourceData) {
	d.SetId(strconv.Itoa(build.MBPKGID))
	_ = d.Set("mbpkgid", build.MBPKGID)
	_ = d.Set("build_id", build.Build)
	_ = d.Set("build_status", build.Status)
}

func resourceDedicatedServerIPv4Reverse() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDedicatedServerIPv4ReverseUpsert,
		ReadContext:   resourceDedicatedServerIPv4ReverseRead,
		UpdateContext: resourceDedicatedServerIPv4ReverseUpsert,
		DeleteContext: resourceDedicatedServerIPv4ReverseDelete,
		Description:   "Manages reverse DNS for one dedicated server IPv4 address.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Dedicated server package ID.",
			},
			"ipv4_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Dedicated server IPv4 address ID.",
			},
			"reverse": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reverse DNS hostname.",
			},
		},
	}
}

func resourceDedicatedServerIPv4ReverseUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	req := &gona.DedicatedIPv4ReverseRequest{
		MBPkgID: d.Get("mbpkgid").(int),
		ID:      d.Get("ipv4_id").(int),
		Reverse: d.Get("reverse").(string),
	}
	if err := c.UpdateDedicatedServerIPv4Reverse(req); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", req.MBPkgID, req.ID))
	return resourceDedicatedServerIPv4ReverseRead(ctx, d, m)
}

func resourceDedicatedServerIPv4ReverseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbPkgID, ipv4ID, err := parseDedicatedIPv4ReverseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("mbpkgid", mbPkgID, d, &diags)
	setValue("ipv4_id", ipv4ID, d, &diags)
	return diags
}

func resourceDedicatedServerIPv4ReverseDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Dedicated IPv4 reverse DNS remains on the account",
		Detail:   "The NetActuate API has no endpoint to clear dedicated server IPv4 reverse DNS. Terraform removed " + id + " from state, but the reverse DNS value remains on the account.",
	}}
}

func parseDedicatedIPv4ReverseID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid dedicated IPv4 reverse ID %q, expected {mbpkgid}/{ipv4_id}", id)
	}
	mbPkgID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid mbpkgid %q: %s", parts[0], err)
	}
	ipv4ID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid IPv4 ID %q: %s", parts[1], err)
	}
	return mbPkgID, ipv4ID, nil
}
