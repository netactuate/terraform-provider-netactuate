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

func resourceUsageContract() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUsageContractCreate,
		ReadContext:   resourceUsageContractRead,
		DeleteContext: resourceUsageContractDelete,
		Description:   "Creates an account usage contract. The API has no delete endpoint for usage contracts, so deleting this resource removes it from Terraform state only and leaves the contract on the account.",
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: contractUsageSchema(map[string]*schema.Schema{
			"mb_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Account ID that should receive the usage contract.",
			},
		}),
	}
}

func resourceCloudIPv4ReverseDNS() *schema.Resource {
	return reverseDNSResource("IPv4", func(c *gona.Client, addressID int, reverse string) error {
		return c.UpdateCloudIPv4ReverseDNS(addressID, reverse)
	}, func(c *gona.Client, mbpkgID int) ([]gona.ServerIPAddress, error) {
		return c.GetServerIPv4(mbpkgID)
	})
}

func resourceCloudIPv6ReverseDNS() *schema.Resource {
	return reverseDNSResource("IPv6", func(c *gona.Client, addressID int, reverse string) error {
		return c.UpdateCloudIPv6ReverseDNS(addressID, reverse)
	}, func(c *gona.Client, mbpkgID int) ([]gona.ServerIPAddress, error) {
		return c.GetServerIPv6(mbpkgID)
	})
}

func resourceCloudFloatingIPv4() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudFloatingIPv4Create,
		ReadContext:   resourceCloudFloatingIPv4Read,
		DeleteContext: resourceCloudFloatingIPv4Delete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"floating_ipv4_id": computedInt("Floating IPv4 ID."),
			"ptr_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "PTR domain to assign when the floating IPv4 address is created.",
			},
			"vlan_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "VLAN ID to associate when the floating IPv4 address is created.",
			},
			"assigned_on": computedString("Floating IPv4 assignment timestamp. The API field is spelled AssignedOn."),
			"address":     computedString("Floating IPv4 address."),
		},
	}
}

func resourceCloudFloatingIPv4VMGrant() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudFloatingIPv4VMGrantCreate,
		ReadContext:   resourceCloudFloatingIPv4VMGrantRead,
		UpdateContext: resourceCloudFloatingIPv4VMGrantUpdate,
		DeleteContext: resourceCloudFloatingIPv4VMGrantDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceCloudFloatingIPv4VMGrantImport},
		Schema: map[string]*schema.Schema{
			"floating_ipv4_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Floating IPv4 ID whose VM allow list should be managed.",
			},
			"mbpkgids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Server package IDs allowed to access the floating IPv4 address.",
			},
			"revoke_existing": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "When set, controls whether the grant call revokes existing VM access before applying mbpkgids. When omitted, the API default is used.",
			},
		},
	}
}

func resourceServerOptions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerOptionsApply,
		ReadContext:   resourceServerOptionsRead,
		UpdateContext: resourceServerOptionsApply,
		DeleteContext: resourceServerOptionsDelete,
		Description:   "Updates mutable options on an existing cloud server. The API has no delete endpoint for server options, so deleting this resource removes it from Terraform state only and leaves the server options unchanged.",
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Server package ID whose options should be managed.",
			},
			"fqdn":        optionalString("Server hostname to set."),
			"autorescue":  optionalInt("Autorescue value to set."),
			"description": optionalString("Server description to set."),
			"vcpus":       optionalInt("Server vCPU count to set."),
			"boot":        optionalString("Server boot option to set."),
			"kernel_id":   optionalInt("Server kernel ID to set."),
			"raw_json":    computedString("Raw JSON payload returned by the last update call."),
		},
	}
}

func resourceCloudFirewallSetBinding() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudFirewallSetBindingCreate,
		ReadContext:   resourceCloudFirewallSetBindingRead,
		DeleteContext: resourceCloudFirewallSetBindingDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceCloudFirewallSetBindingImport},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Cloud server package ID to bind to the firewall set.",
			},
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Firewall set ID to bind to the cloud server.",
			},
			"interface_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Server interface ID sent in the bind request.",
			},
			"set_priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Firewall set priority sent in the bind request.",
			},
			"raw_json": computedString("Raw JSON payload returned by the bind call."),
		},
	}
}

func resourceUsageContractCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	contract, err := m.(*ProviderClients).V2.CreateUsageContract(&gona.CreateUsageContractRequest{MBID: d.Get("mb_id").(int)})
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setContractUsageState(contract, d, &diags)
	if contract.ID != nil && *contract.ID != 0 {
		d.SetId(strconv.Itoa(*contract.ID))
	} else if contract.ContractMBPkgID != nil && *contract.ContractMBPkgID != 0 {
		d.SetId(strconv.Itoa(*contract.ContractMBPkgID))
	} else {
		d.SetId(strconv.Itoa(d.Get("mb_id").(int)))
	}
	return diags
}

func resourceUsageContractRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	contract, err := m.(*ProviderClients).V2.GetContractUsage()
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setContractUsageState(contract, d, &diags)
	return diags
}

func resourceUsageContractDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Usage contract remains on the account",
		Detail:   "The NetActuate API has no delete endpoint for usage contracts. Terraform removed usage contract " + id + " from state, but the contract remains on the account.",
	}}
}

func reverseDNSResource(family string, update func(*gona.Client, int, string) error, read func(*gona.Client, int) ([]gona.ServerIPAddress, error)) *schema.Resource {
	return &schema.Resource{
		CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return reverseDNSApply(d, m, update, read)
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return reverseDNSRead(d, m, read)
		},
		UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return reverseDNSApply(d, m, update, read)
		},
		DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			c := m.(*ProviderClients).V2
			_, addressID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "address_id")
			if err != nil {
				return diag.FromErr(err)
			}
			if err := update(c, addressID, ""); err != nil {
				return diag.FromErr(err)
			}
			d.SetId("")
			return nil
		},
		Importer: &schema.ResourceImporter{StateContext: reverseDNSImport},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Server package ID that owns the " + family + " address.",
			},
			"address_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: family + " address ID.",
			},
			"reverse": {
				Type:        schema.TypeString,
				Required:    true,
				Description: family + " reverse DNS value.",
			},
			"ip": computedString(family + " address."),
		},
	}
}

func reverseDNSApply(d *schema.ResourceData, m interface{}, update func(*gona.Client, int, string) error, read func(*gona.Client, int) ([]gona.ServerIPAddress, error)) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgID := d.Get("mbpkgid").(int)
	addressID := d.Get("address_id").(int)
	if err := update(c, addressID, d.Get("reverse").(string)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", mbpkgID, addressID))
	return reverseDNSRead(d, m, read)
}

func reverseDNSRead(d *schema.ResourceData, m interface{}, read func(*gona.Client, int) ([]gona.ServerIPAddress, error)) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgID, addressID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "address_id")
	if err != nil {
		return diag.FromErr(err)
	}
	addresses, err := read(c, mbpkgID)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	for _, address := range addresses {
		if address.ID == addressID {
			var diags diag.Diagnostics
			setValue("mbpkgid", mbpkgID, d, &diags)
			setValue("address_id", addressID, d, &diags)
			setValue("ip", address.IP, d, &diags)
			setValue("reverse", stringPtrOrEmpty(address.Reverse), d, &diags)
			return diags
		}
	}
	d.SetId("")
	return nil
}

func reverseDNSImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	mbpkgID, addressID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "address_id")
	if err != nil {
		return nil, err
	}
	if err := d.Set("mbpkgid", mbpkgID); err != nil {
		return nil, err
	}
	if err := d.Set("address_id", addressID); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceCloudFloatingIPv4Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	req := &gona.CreateCloudFloatingIPv4Request{}
	if v, ok := d.GetOk("ptr_domain"); ok {
		value := v.(string)
		req.PTRDomain = &value
	}
	if v, ok := d.GetOk("vlan_id"); ok {
		value := v.(int)
		req.VLANID = &value
	}
	floatingIP, err := m.(*ProviderClients).V3.CreateCloudFloatingIPv4(req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(floatingIP.FloatingIPv4ID))
	return setCloudFloatingIPv4State(floatingIP, d)
}

func resourceCloudFloatingIPv4Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	floatingIPs, err := m.(*ProviderClients).V3.ListCloudFloatingIPv4()
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	for _, floatingIP := range floatingIPs {
		if floatingIP.FloatingIPv4ID == id {
			return setCloudFloatingIPv4State(floatingIP, d)
		}
	}
	d.SetId("")
	return nil
}

func resourceCloudFloatingIPv4Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := m.(*ProviderClients).V3.DeleteCloudFloatingIPv4(id); err != nil && !gona.IsNotFound(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func setCloudFloatingIPv4State(floatingIP gona.CloudFloatingIPv4, d *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics
	setValue("floating_ipv4_id", floatingIP.FloatingIPv4ID, d, &diags)
	setValue("ptr_domain", stringPtrOrEmpty(floatingIP.PTRDomain), d, &diags)
	setValue("vlan_id", floatingIP.VLANID, d, &diags)
	setValue("assigned_on", floatingIP.AssignedOn, d, &diags)
	setValue("address", floatingIP.Address, d, &diags)
	return diags
}

func resourceCloudFloatingIPv4VMGrantCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if diags := grantCloudFloatingIPv4VMs(d, m); diags.HasError() {
		return diags
	}
	d.SetId(strconv.Itoa(d.Get("floating_ipv4_id").(int)))
	return resourceCloudFloatingIPv4VMGrantRead(ctx, d, m)
}

func resourceCloudFloatingIPv4VMGrantRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	vms, err := m.(*ProviderClients).V3.ListCloudFloatingIPv4VMs(id)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	ids := make([]int, len(vms))
	for i, vm := range vms {
		ids[i] = vm.MBPkgID
	}
	var diags diag.Diagnostics
	setValue("floating_ipv4_id", id, d, &diags)
	setValue("mbpkgids", ids, d, &diags)
	return diags
}

func resourceCloudFloatingIPv4VMGrantUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if diags := grantCloudFloatingIPv4VMs(d, m); diags.HasError() {
		return diags
	}
	return resourceCloudFloatingIPv4VMGrantRead(ctx, d, m)
}

func resourceCloudFloatingIPv4VMGrantDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	req := &gona.RevokeCloudFloatingIPv4VMsRequest{VMs: cloudFloatingIPv4VMRefs(d.Get("mbpkgids").(*schema.Set))}
	if err := m.(*ProviderClients).V3.RevokeCloudFloatingIPv4VMs(id, req); err != nil && !gona.IsNotFound(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func resourceCloudFloatingIPv4VMGrantImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid floating_ipv4_id %q: %s", d.Id(), err)
	}
	if err := d.Set("floating_ipv4_id", id); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func grantCloudFloatingIPv4VMs(d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("floating_ipv4_id").(int)
	req := &gona.GrantCloudFloatingIPv4VMsRequest{VMs: cloudFloatingIPv4VMRefs(d.Get("mbpkgids").(*schema.Set))}
	if v, ok := d.GetOkExists("revoke_existing"); ok {
		value := v.(bool)
		req.RevokeExisting = &value
	}
	if err := m.(*ProviderClients).V3.GrantCloudFloatingIPv4VMs(id, req); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func cloudFloatingIPv4VMRefs(set *schema.Set) []gona.CloudFloatingIPv4VMRef {
	values := set.List()
	refs := make([]gona.CloudFloatingIPv4VMRef, len(values))
	for i, value := range values {
		refs[i] = gona.CloudFloatingIPv4VMRef{MBPkgID: value.(int)}
	}
	return refs
}

func resourceServerOptionsApply(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	req := &gona.UpdateServerOptionsRequest{}
	if v, ok := d.GetOk("fqdn"); ok {
		value := v.(string)
		req.FQDN = &value
	}
	if v, ok := d.GetOk("autorescue"); ok {
		value := v.(int)
		req.Autorescue = &value
	}
	if v, ok := d.GetOk("description"); ok {
		value := v.(string)
		req.Description = &value
	}
	if v, ok := d.GetOk("vcpus"); ok {
		value := v.(int)
		req.VCPUs = &value
	}
	if v, ok := d.GetOk("boot"); ok {
		value := v.(string)
		req.Boot = &value
	}
	if v, ok := d.GetOk("kernel_id"); ok {
		value := v.(int)
		req.KernelID = &value
	}
	mbpkgID := d.Get("mbpkgid").(int)
	raw, err := m.(*ProviderClients).V2.UpdateServerOptions(mbpkgID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(mbpkgID))
	var diags diag.Diagnostics
	setValue("raw_json", string(raw), d, &diags)
	return diags
}

func resourceServerOptionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbpkgID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	server, err := m.(*ProviderClients).V2.GetServer(mbpkgID)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("mbpkgid", mbpkgID, d, &diags)
	setValue("fqdn", server.Name, d, &diags)
	return diags
}

func resourceServerOptionsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Server options remain on the server",
		Detail:   "The NetActuate API has no delete endpoint for server options. Terraform removed server options " + id + " from state, but the configured server options remain on the server.",
	}}
}

func resourceCloudFirewallSetBindingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbpkgID := d.Get("mbpkgid").(int)
	firewallSetID := d.Get("firewall_set_id").(int)
	req := &gona.BindFirewallSetRequest{FirewallSetID: firewallSetID}
	if v, ok := d.GetOk("interface_id"); ok {
		req.InterfaceID = v.(int)
	}
	if v, ok := d.GetOk("set_priority"); ok {
		req.SetPriority = v.(int)
	}
	raw, err := m.(*ProviderClients).V2.BindCloudFirewallSet(mbpkgID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", mbpkgID, firewallSetID))
	var diags diag.Diagnostics
	setValue("raw_json", string(raw), d, &diags)
	return diags
}

func resourceCloudFirewallSetBindingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbpkgID, firewallSetID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "firewall_set_id")
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("mbpkgid", mbpkgID, d, &diags)
	setValue("firewall_set_id", firewallSetID, d, &diags)
	return diags
}

func resourceCloudFirewallSetBindingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbpkgID, firewallSetID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "firewall_set_id")
	if err != nil {
		return diag.FromErr(err)
	}
	if err := m.(*ProviderClients).V2.UnbindCloudFirewallSet(mbpkgID, strconv.Itoa(firewallSetID)); err != nil && !gona.IsNotFound(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func resourceCloudFirewallSetBindingImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	mbpkgID, firewallSetID, err := parseTwoPartIntID(d.Id(), "mbpkgid", "firewall_set_id")
	if err != nil {
		return nil, err
	}
	if err := d.Set("mbpkgid", mbpkgID); err != nil {
		return nil, err
	}
	if err := d.Set("firewall_set_id", firewallSetID); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func parseTwoPartIntID(id, leftName, rightName string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid ID format %q, expected {%s}/{%s}", id, leftName, rightName)
	}
	left, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s %q: %s", leftName, parts[0], err)
	}
	right, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s %q: %s", rightName, parts[1], err)
	}
	return left, right, nil
}
