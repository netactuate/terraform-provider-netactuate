package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceVPC() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCCreate,
		ReadContext:   resourceVPCRead,
		UpdateContext: resourceVPCUpdate,
		DeleteContext: resourceVPCDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			if d.HasChange("bastion_port") {
				_, newVal := d.GetChange("bastion_port")
				port := newVal.(int)
				d.SetNew("bastion_enabled", port > 0)
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the VPC",
			},
			"label": {
				Type:     schema.TypeString,
				Required: true,
				// The 32 character limit was already DOCUMENTED here and not enforced, so
				// an over-long label was accepted by the plan and rejected by the API at
				// apply time with
				//   400 label: "String must be at most 32 characters in length."
				// after the plan had already been approved. Validated now, so the failure
				// happens at plan with the customer's own value named.
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 32)),
				Description:      "The name of the VPC shown in the portal (max 32 chars)",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A description of the VPC (max 255 chars)",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location_id", "location"},
				Description:  "The location ID to deploy the VPC into",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress location_id drift when location (name) is set in config
					_, hasLocation := d.GetOk("location")
					return hasLocation && new == "0"
				},
			},
			"location": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"location_id", "location"},
				DiffSuppressFunc: suppressLocationDiff,
				Description:      "The location name to deploy the VPC",
			},
			"network_ipv4": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The IPv4 network CIDR available within the VPC (e.g. 10.0.0.0/24)",
			},
			"network_ipv6": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The IPv6 network CIDR available within the VPC",
			},
			"nameservers_ipv4": {
				Type:     schema.TypeList,
				ForceNew: true,
				// Optional AND Computed. The platform assigns nameservers when the VPC is
				// created and Read hydrates them, so Optional alone means any config that
				// omits the field gets a permanent "remove these" diff on every plan.
				// Create then plan can propose removing both entries. Computed makes
				// an omitted field adopt the API's value.
				Optional:    true,
				Computed:    true,
				Description: "IPv4 nameservers for DHCP",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"nameservers_ipv6": {
				Type:        schema.TypeList,
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				Description: "IPv6 nameservers for DHCP",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"enable_default_snat": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				Description: "If true, requests a default SNAT rule at creation time for outbound " +
					"internet connectivity. This is a create-time convenience only -- the API has no " +
					"persistent representation of it, so it can never be verified or recovered by " +
					"Read/import; state simply keeps whatever value was last configured or defaulted. " +
					"To manage SNAT rules as real, importable/driftable state, use " +
					"netactuate_vpc_gateway_snat_rule instead.",
			},
			"firewall_ipv4_inbound": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable IPv4 inbound firewall",
			},
			"firewall_ipv4_outbound": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable IPv4 outbound firewall",
			},
			"firewall_ipv6_inbound": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable IPv6 inbound firewall",
			},
			"firewall_ipv6_outbound": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable IPv6 outbound firewall",
			},
			"bastion_ipv4": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bastion IPv4 address",
			},
			"bastion_ipv6": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bastion IPv6 address",
			},
			"bastion_port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The bastion SSH port",
			},
			"bastion_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the bastion is enabled",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The overall status of the VPC",
			},
			"network_loadbalancer_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The network load balancer ID for this VPC. Empty when the API reports balancers as counts rather than identifiers.",
			},
			"http_loadbalancer_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The HTTP load balancer ID for this VPC. Empty when the API reports balancers as counts rather than identifiers.",
			},
			"network_loadbalancer_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "How many network load balancers exist in this VPC.",
			},
			"http_loadbalancer_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "How many HTTP load balancers exist in this VPC.",
			},
		},
	}
}

func resourceVPCCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	locationID, locDiag := getLocationID(d, m.(*ProviderClients).V2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	req := &gona.CreateVPCRequest{
		Label:       d.Get("label").(string),
		Description: d.Get("description").(string),
		LocationID:  locationID,
	}

	// Network
	netV4 := d.Get("network_ipv4").(string)
	netV6 := d.Get("network_ipv6").(string)
	if netV4 != "" || netV6 != "" {
		req.Network = &gona.VPCNetwork{
			IPv4: netV4,
			IPv6: netV6,
		}
	}

	// Nameservers
	nsV4Raw := d.Get("nameservers_ipv4").([]interface{})
	nsV6Raw := d.Get("nameservers_ipv6").([]interface{})
	if len(nsV4Raw) > 0 || len(nsV6Raw) > 0 {
		ns := &gona.VPCNameservers{}
		for _, v := range nsV4Raw {
			ns.IPv4 = append(ns.IPv4, gona.VPCNameserver{Server: v.(string)})
		}
		for _, v := range nsV6Raw {
			ns.IPv6 = append(ns.IPv6, gona.VPCNameserver{Server: v.(string)})
		}
		req.Nameservers = ns
	}

	// Default SNAT
	if d.Get("enable_default_snat").(bool) {
		enableSnat := true
		req.Defaults = &gona.VPCDefaults{EnableDefaultSnatRule: &enableSnat}
	}

	// Firewalls
	fwV4In := d.Get("firewall_ipv4_inbound").(bool)
	fwV4Out := d.Get("firewall_ipv4_outbound").(bool)
	fwV6In := d.Get("firewall_ipv6_inbound").(bool)
	fwV6Out := d.Get("firewall_ipv6_outbound").(bool)
	if fwV4In || fwV4Out || fwV6In || fwV6Out {
		req.Firewalls = &gona.VPCFirewalls{
			IPv4: &gona.VPCFirewallDirections{Inbound: &fwV4In, Outbound: &fwV4Out},
			IPv6: &gona.VPCFirewallDirections{Inbound: &fwV6In, Outbound: &fwV6Out},
		}
	}

	vpc, err := c.CreateVPC(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(vpc.VPCID))
	log.Printf("[DEBUG] VPC created with ID: %d, waiting for ready...", vpc.VPCID)

	if err := c.WaitForVPCReady(vpc.VPCID); err != nil {
		return diag.Errorf("VPC %d created but failed to become ready: %s", vpc.VPCID, err)
	}

	// Apply bastion port if specified (not part of the create request)
	if port := d.Get("bastion_port").(int); port > 0 {
		sshReq := &gona.UpdateVPCSSHSettingsRequest{Port: &port}
		if _, err := c.UpdateVPCSSHSettings(vpc.VPCID, sshReq); err != nil {
			return diag.FromErr(fmt.Errorf("set VPC %d bastion port after create: %w", vpc.VPCID, err))
		}
	}

	return resourceVPCRead(ctx, d, m)
}

func resourceVPCRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	vpc, err := c.GetVPC(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] VPC %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpc.VPCID, d, &diags)
	setValue("bastion_ipv4", vpc.Bastion.Addresses.IPv4, d, &diags)
	setValue("bastion_ipv6", vpc.Bastion.Addresses.IPv6, d, &diags)
	setValue("bastion_enabled", vpc.Bastion.Enabled, d, &diags)
	if vpc.Bastion.Port != nil {
		setValue("bastion_port", *vpc.Bastion.Port, d, &diags)
	} else {
		setValue("bastion_port", 0, d, &diags)
	}
	setValue("status", vpc.Metadata.Status, d, &diags)

	// The payload reports each balancer class either as a list carrying identifiers or as a
	// count. Both are set from whichever arrived, so the id is empty rather than wrong when the
	// platform answered with counts.
	if vpc.LoadBalancers != nil {
		if n := vpc.LoadBalancers.Network; n != nil {
			setValue("network_loadbalancer_count", n.Total, d, &diags)
			if len(n.IDs) > 0 {
				setValue("network_loadbalancer_id", n.IDs[0], d, &diags)
			}
		}
		if h := vpc.LoadBalancers.HTTP; h != nil {
			setValue("http_loadbalancer_count", h.Total, d, &diags)
			if len(h.IDs) > 0 {
				setValue("http_loadbalancer_id", h.IDs[0], d, &diags)
			}
		}
	}

	setValue("label", vpc.Metadata.Label, d, &diags)
	setValue("description", vpc.Metadata.Description, d, &diags)
	setValue("location_id", vpc.Location.ID, d, &diags)
	setLocationPreserveFormat(vpc.Location.Name, d, &diags)

	if vpc.InternalNetwork != nil {
		setValue("network_ipv4", vpc.InternalNetwork.IPv4, d, &diags)
		setValue("network_ipv6", vpc.InternalNetwork.IPv6, d, &diags)
	}

	if vpc.DHCP != nil && vpc.DHCP.Nameservers != nil {
		setValue("nameservers_ipv4", stripHostCIDRSuffixes(vpc.DHCP.Nameservers.IPv4), d, &diags)
		setValue("nameservers_ipv6", stripHostCIDRSuffixes(vpc.DHCP.Nameservers.IPv6), d, &diags)
	}

	if vpc.Firewalls != nil {
		if vpc.Firewalls.IPv4 != nil {
			if vpc.Firewalls.IPv4.Inbound != nil {
				setValue("firewall_ipv4_inbound", vpc.Firewalls.IPv4.Inbound.Enabled, d, &diags)
			}
			if vpc.Firewalls.IPv4.Outbound != nil {
				setValue("firewall_ipv4_outbound", vpc.Firewalls.IPv4.Outbound.Enabled, d, &diags)
			}
		}
		if vpc.Firewalls.IPv6 != nil {
			if vpc.Firewalls.IPv6.Inbound != nil {
				setValue("firewall_ipv6_inbound", vpc.Firewalls.IPv6.Inbound.Enabled, d, &diags)
			}
			if vpc.Firewalls.IPv6.Outbound != nil {
				setValue("firewall_ipv6_outbound", vpc.Firewalls.IPv6.Outbound.Enabled, d, &diags)
			}
		}
	}

	return diags
}

func resourceVPCUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateVPCRequest{}
	changed := false

	if d.HasChange("label") {
		req.Label = d.Get("label").(string)
		changed = true
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
		changed = true
	}

	if d.HasChanges("firewall_ipv4_inbound", "firewall_ipv4_outbound", "firewall_ipv6_inbound", "firewall_ipv6_outbound") {
		fwV4In := d.Get("firewall_ipv4_inbound").(bool)
		fwV4Out := d.Get("firewall_ipv4_outbound").(bool)
		fwV6In := d.Get("firewall_ipv6_inbound").(bool)
		fwV6Out := d.Get("firewall_ipv6_outbound").(bool)
		req.Firewalls = &gona.VPCFirewalls{
			IPv4: &gona.VPCFirewallDirections{Inbound: &fwV4In, Outbound: &fwV4Out},
			IPv6: &gona.VPCFirewallDirections{Inbound: &fwV6In, Outbound: &fwV6Out},
		}
		changed = true
	}

	if changed {
		_, err := c.UpdateVPC(id, req)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Setting bastion_port = 0 (or removing it) disables the bastion by sending port: null
	if d.HasChange("bastion_port") {
		req := &gona.UpdateVPCSSHSettingsRequest{}
		port := d.Get("bastion_port").(int)
		if port > 0 {
			req.Port = &port
		}
		if _, err := c.UpdateVPCSSHSettings(id, req); err != nil {
			return diag.FromErr(fmt.Errorf("update VPC %d bastion port: %w", id, err))
		}
	}

	// Update nameservers if changed
	if d.HasChanges("nameservers_ipv4", "nameservers_ipv6") {
		ns := &gona.VPCNameservers{}
		for _, v := range d.Get("nameservers_ipv4").([]interface{}) {
			ns.IPv4 = append(ns.IPv4, gona.VPCNameserver{Server: v.(string)})
		}
		for _, v := range d.Get("nameservers_ipv6").([]interface{}) {
			ns.IPv6 = append(ns.IPv6, gona.VPCNameserver{Server: v.(string)})
		}
		if _, err := c.UpdateVPCNameservers(id, ns); err != nil {
			return diag.FromErr(fmt.Errorf("update VPC %d nameservers: %w", id, err))
		}
	}

	return resourceVPCRead(ctx, d, m)
}

func resourceVPCDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting VPC %d", id)

	// A VPC whose VMs are still being torn down refuses deletion with
	//   400 "You currently have virtual machines in this VPC so you cannot delete it."
	// That is TRANSIENT: server deletion is a job, so the VM can be gone from Terraform's
	// view while the platform is still detaching it. Failing immediately leaves the VPC
	// behind and require manual cleanup.
	//
	// Retried for a bounded time, and only on this specific message. Any other 400 is a
	// real error and still fails at once.
	const vpcDeleteRetries = 12
	const vpcDeleteInterval = 10 * time.Second
	for attempt := 0; ; attempt++ {
		err := c.DeleteVPC(id)
		if err == nil {
			return nil
		}
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] VPC %d already deleted", id)
			return nil
		}
		if !strings.Contains(err.Error(), "virtual machines in this VPC") {
			return diag.FromErr(err)
		}
		if attempt >= vpcDeleteRetries-1 {
			return diag.Errorf(
				"VPC %d still has virtual machines after %d attempts over %s. "+
					"Delete or detach them first: %s",
				id, vpcDeleteRetries, time.Duration(vpcDeleteRetries)*vpcDeleteInterval, err)
		}
		log.Printf("[DEBUG] VPC %d still has VMs attached, retry %d/%d",
			id, attempt+1, vpcDeleteRetries)
		time.Sleep(vpcDeleteInterval)
	}
}

// stripHostCIDRSuffixes drops a trailing "/32" (IPv4) or "/128" (IPv6) from
// each address. The API returns nameservers CIDR-suffixed
// (e.g. "192.0.2.53/32") even though a bare address like
// "192.0.2.53" is what's configured -- setting the raw API value verbatim
// would produce a permanent, unresolvable diff on every plan.
func stripHostCIDRSuffixes(addrs []string) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		a = strings.TrimSuffix(a, "/32")
		a = strings.TrimSuffix(a, "/128")
		out[i] = a
	}
	return out
}
