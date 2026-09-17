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
				Type:     schema.TypeString,
				Required: true,
				// Only inbound is honored. Outbound is present in some API docs but
				// never takes effect, confirmed against the platform and the portal,
				// so accepting it would let a config request filtering that silently
				// does nothing.
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"inbound"}, false)),
				Description:      "Traffic direction. Only inbound is supported.",
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

	result, err := createVPCFirewallRuleWithRetry(c, vpcID, req)
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
	// Don't store default network value to avoid drifting
	isDefault := rule.Network == "0.0.0.0/0" || rule.Network == "::/0"
	currentNetwork, _ := d.GetOk("network")
	hasNetworkInState := currentNetwork != nil && currentNetwork.(string) != ""
	if !isDefault || hasNetworkInState {
		setValue("network", rule.Network, d, &diags)
	}

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

// createVPCFirewallRuleWithRetry creates a gateway firewall rule, retrying a 5xx safely.
//
// This endpoint returns HTTP 500 intermittently, measured at roughly one run in five across
// fourteen runs of scenarios/vpc-private-networking, with the platform's own message being
// "An error occurred while processing your request. Please try again later."
//
// It was first thought to be contention, because Terraform creates independent rules in
// parallel. That was tested by serialising them with depends_on and it was REFUTED: the
// serialised configuration failed the same way on the first rule in the chain, with nothing
// running alongside it. Parallel failed 2 of 8, serialised 1 of 6. So a customer cannot avoid
// this by ordering their configuration, and the mitigation has to be here.
//
// The retry is not blind, and that matters. A 500 can mean the rule was created and the
// response was lost, so repeating the POST risks a duplicate rule. Before each retry the rule
// list is re-read and a rule matching this request is adopted if one is already there. This is
// the same shape as resourceFirewallRuleCreate's findRuleByRequest for the VM firewall.
func createVPCFirewallRuleWithRetry(c *gona.V3Client, vpcID int, req *gona.CreateVPCFirewallRuleRequest) (*gona.CreateVPCFirewallRuleResponse, error) {
	const attempts = 3

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := c.CreateVPCFirewallRule(vpcID, req)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Only a server side fault is worth repeating. A 4xx means the request is wrong and
		// will stay wrong.
		if !isVPCFirewallRetryable(err) {
			return nil, err
		}

		// The write may have landed even though the response did not come back. Adopt it
		// rather than creating a second copy.
		if existing, findErr := findVPCFirewallRuleByRequest(c, vpcID, req); findErr == nil && existing != nil {
			log.Printf("[WARN] VPC %d firewall rule create returned %v, but rule %d matching the request already exists; adopting it", vpcID, err, existing.FirewallRuleID)
			return &gona.CreateVPCFirewallRuleResponse{FirewallRuleID: existing.FirewallRuleID}, nil
		}

		if attempt < attempts {
			log.Printf("[WARN] VPC %d firewall rule create attempt %d/%d failed (%v), retrying", vpcID, attempt, attempts, err)
			time.Sleep(time.Duration(attempt) * 3 * time.Second)
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

// isVPCFirewallRetryable reports whether an error is a server side fault worth repeating.
func isVPCFirewallRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, marker := range []string{"HTTP 500", "HTTP 502", "HTTP 503", "HTTP 504"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// findVPCFirewallRuleByRequest looks for a rule already matching what we tried to create, so a
// retry after a lost response adopts it instead of duplicating it. Matched on the fields the
// request actually sets, since the API assigns the id.
func findVPCFirewallRuleByRequest(c *gona.V3Client, vpcID int, req *gona.CreateVPCFirewallRuleRequest) (*gona.VPCFirewallRule, error) {
	rules, err := c.ListVPCFirewallRules(vpcID, req.IPVersion)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if vpcFirewallRuleMatchesRequest(&rules[i], req) {
			return &rules[i], nil
		}
	}
	return nil, nil
}

// vpcFirewallRuleMatchesRequest reports whether an existing rule is the one this request was
// trying to create. Kept pure and separate from the client call so it can be unit tested: it is
// the part of the retry most likely to be wrong, and getting it wrong means either duplicating
// a rule or adopting the wrong one.
func vpcFirewallRuleMatchesRequest(r *gona.VPCFirewallRule, req *gona.CreateVPCFirewallRuleRequest) bool {
	if r.Direction != req.Direction || r.Protocol != req.Protocol || r.Network != req.Network {
		return false
	}
	if req.Description != "" && r.Description != req.Description {
		return false
	}
	if req.Port == nil {
		return r.Port == nil
	}
	if r.Port == nil || r.Port.Start != req.Port.Start {
		return false
	}
	// A single port rule comes back from the API with end null, which unmarshals to zero and
	// means the same as start.
	gotEnd := r.Port.End
	if gotEnd == 0 {
		gotEnd = r.Port.Start
	}
	wantEnd := req.Port.End
	if wantEnd == 0 {
		wantEnd = req.Port.Start
	}
	return gotEnd == wantEnd
}
