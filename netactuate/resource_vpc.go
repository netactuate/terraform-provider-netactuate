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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the VPC shown in the portal (max 32 chars)",
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
			},
			"location": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location_id", "location"},
				Description:  "The location name to deploy the VPC",
			},
			"network_ipv4": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The IPv4 network CIDR available within the VPC (e.g. 10.0.0.0/24)",
			},
			"network_ipv6": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The IPv6 network CIDR available within the VPC",
			},
			"nameservers_ipv4": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "IPv4 nameservers for DHCP",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"nameservers_ipv6": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "IPv6 nameservers for DHCP",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"enable_default_snat": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     false,
				Description: "If true, the VPC will have a default SNAT rule allowing outbound internet connectivity",
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
				Description: "The network load balancer ID for this VPC",
			},
			"http_loadbalancer_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The HTTP load balancer ID for this VPC",
			},
		},
	}
}

func resourceVPCCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	locationID, locDiag := getVPCLocation(d, m.(*ProviderClients).V2)
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

	if vpc.LoadBalancers != nil {
		if len(vpc.LoadBalancers.Network) > 0 {
			setValue("network_loadbalancer_id", vpc.LoadBalancers.Network[0].NetworkLbID, d, &diags)
		}
		if len(vpc.LoadBalancers.HTTP) > 0 {
			setValue("http_loadbalancer_id", vpc.LoadBalancers.HTTP[0].HTTPLbID, d, &diags)
		}
	}

	setValue("label", vpc.Metadata.Label, d, &diags)
	setValue("description", vpc.Metadata.Description, d, &diags)
	setValue("location_id", vpc.Location.ID, d, &diags)
	setValue("location", vpc.Location.Name, d, &diags)

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

	if err := c.DeleteVPC(id); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] VPC %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func getVPCLocation(d *schema.ResourceData, c *gona.Client) (int, *diag.Diagnostic) {
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
		log.Printf("[DEBUG] Available location: ID=%d Name=%q IATACode=%q", loc.ID, loc.Name, loc.IATACode)
		if strings.EqualFold(loc.Name, locationName) || strings.EqualFold(loc.IATACode, locationName) {
			return loc.ID, nil
		}
	}

	return 0, &diag.Errorf("VPC location %q not found", locationName)[0]
}
