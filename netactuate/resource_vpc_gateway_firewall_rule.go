package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceVPCGatewayFirewallRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCGatewayFirewallRuleCreate,
		ReadContext:   resourceVPCGatewayFirewallRuleRead,
		UpdateContext: resourceVPCGatewayFirewallRuleUpdate,
		DeleteContext: resourceVPCGatewayFirewallRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCGatewayFirewallRuleImport,
		},
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the parent VPC",
			},
			"rule_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The firewall rule ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "IP version for this rule (4 or 6)",
			},
			"direction": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"inbound", "outbound"}, false)),
				Description:      "Traffic direction (inbound or outbound)",
			},
			"protocol": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"TCP", "UDP", "ICMP"}, false)),
				Description:      "Protocol to match (TCP, UDP, or ICMP)",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the rule",
			},
			"network": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network CIDR to match (e.g. 172.16.0.0/12)",
			},
			"port_start": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Start of the port range",
			},
			"port_end": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "End of the port range",
			},
		},
	}
}

func resourceVPCGatewayFirewallRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)

	req := &gona.CreateVPCFirewallRuleRequest{
		IPVersion: d.Get("ip_version").(int),
		Direction: d.Get("direction").(string),
	}

	if v, ok := d.GetOk("protocol"); ok {
		req.Protocol = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}
	if v, ok := d.GetOk("network"); ok {
		req.Network = v.(string)
	}

	portStart, hasPS := d.GetOk("port_start")
	portEnd, hasPE := d.GetOk("port_end")
	if hasPS || hasPE {
		req.Port = &gona.VPCPortRange{}
		if hasPS {
			req.Port.Start = portStart.(int)
		}
		if hasPE {
			req.Port.End = portEnd.(int)
		}
	}

	result, err := c.CreateVPCFirewallRule(vpcID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, result.FirewallRuleID))
	log.Printf("[DEBUG] Firewall rule %d created in VPC %d, applying changes...", result.FirewallRuleID, vpcID)

	if err := c.ApplyVPCFirewallChanges(vpcID); err != nil {
		return diag.Errorf("firewall rule created but apply-changes failed: %s", err)
	}

	return resourceVPCGatewayFirewallRuleRead(ctx, d, m)
}

func resourceVPCGatewayFirewallRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseFirewallRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ipVersion := d.Get("ip_version").(int)
	if ipVersion == 0 {
		ipVersion = 4
	}

	rule, err := c.GetVPCFirewallRule(vpcID, ruleID, ipVersion)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Firewall rule %d in VPC %d not found, removing from state", ruleID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("rule_id", rule.FirewallRuleID, d, &diags)
	setValue("ip_version", rule.IPVersion, d, &diags)
	setValue("direction", rule.Direction, d, &diags)
	setValue("protocol", rule.Protocol, d, &diags)
	setValue("description", rule.Description, d, &diags)
	setValue("network", rule.Network, d, &diags)

	if rule.Port != nil {
		setValue("port_start", rule.Port.Start, d, &diags)
		portEnd := rule.Port.End
		if portEnd == 0 && rule.Port.Start != 0 {
			portEnd = rule.Port.Start
		}
		setValue("port_end", portEnd, d, &diags)
	}

	return diags
}

func resourceVPCGatewayFirewallRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseFirewallRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateVPCFirewallRuleRequest{}

	if d.HasChange("direction") {
		req.Direction = d.Get("direction").(string)
	}
	if d.HasChange("protocol") {
		req.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}
	if d.HasChange("network") {
		req.Network = d.Get("network").(string)
	}

	if d.HasChanges("port_start", "port_end") {
		ps := d.Get("port_start").(int)
		pe := d.Get("port_end").(int)
		if ps != 0 || pe != 0 {
			req.Port = &gona.VPCPortRange{Start: ps, End: pe}
		}
	}

	if _, err := c.UpdateVPCFirewallRule(vpcID, ruleID, req); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Firewall rule %d updated in VPC %d, applying changes...", ruleID, vpcID)
	if err := c.ApplyVPCFirewallChanges(vpcID); err != nil {
		return diag.Errorf("firewall rule updated but apply-changes failed: %s", err)
	}

	return resourceVPCGatewayFirewallRuleRead(ctx, d, m)
}

func resourceVPCGatewayFirewallRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseFirewallRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting firewall rule %d from VPC %d", ruleID, vpcID)

	if err := c.DeleteVPCFirewallRule(vpcID, ruleID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Firewall rule %d in VPC %d already deleted", ruleID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	if err := c.ApplyVPCFirewallChanges(vpcID); err != nil {
		return diag.Errorf("firewall rule deleted but apply-changes failed: %s", err)
	}

	return nil
}

func resourceVPCGatewayFirewallRuleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid import ID %q, expected \"vpcId/ruleId/ipVersion\" (e.g. \"276/171/4\")", d.Id())
	}
	vpcID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid vpc_id %q: %w", parts[0], err)
	}
	ruleID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid rule_id %q: %w", parts[1], err)
	}
	ipVersion, err := strconv.Atoi(parts[2])
	if err != nil || (ipVersion != 4 && ipVersion != 6) {
		return nil, fmt.Errorf("invalid ip_version %q, must be 4 or 6", parts[2])
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, ruleID))
	d.Set("vpc_id", vpcID)
	d.Set("ip_version", ipVersion)

	return []*schema.ResourceData{d}, nil
}

func parseFirewallRuleID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid firewall rule ID %q, expected \"vpcId/ruleId\"", id)
	}
	vpcID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vpc_id %q: %w", parts[0], err)
	}
	ruleID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid rule_id %q: %w", parts[1], err)
	}
	return vpcID, ruleID, nil
}
