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

func resourceVPCGatewayDNATRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCGatewayDNATRuleCreate,
		ReadContext:   resourceVPCGatewayDNATRuleRead,
		UpdateContext: resourceVPCGatewayDNATRuleUpdate,
		DeleteContext: resourceVPCGatewayDNATRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCGatewayDNATRuleImport,
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
				Description: "The DNAT rule ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "IP version for this rule (4 or 6)",
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
			"match_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Public IP address to match on inbound traffic",
			},
			"match_port_start": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Start of the match port range",
			},
			"match_port_end": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "End of the match port range",
			},
			"translation_address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Private IP address to forward traffic to",
			},
			"translation_port_start": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Start of the translation port range",
			},
			"translation_port_end": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "End of the translation port range",
			},
			"priority_location": {
				Type:             schema.TypeString,
				Optional:         true,
				ConflictsWith:    []string{"priority_after_rule_id"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"start", "end"}, false)),
				Description:      "Place this rule at \"start\" or \"end\" of the rule list",
			},
			"priority_after_rule_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"priority_location"},
				Description:   "Place this rule after the given DNAT rule ID",
			},
		},
	}
}

func resourceVPCGatewayDNATRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)

	req := &gona.CreateVPCDNATRuleRequest{
		IPVersion: d.Get("ip_version").(int),
	}

	if v, ok := d.GetOk("protocol"); ok {
		req.Protocol = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	matchAddr, hasMatchAddr := d.GetOk("match_address")
	matchPS, hasMatchPS := d.GetOk("match_port_start")
	matchPE, hasMatchPE := d.GetOk("match_port_end")
	if hasMatchAddr || hasMatchPS || hasMatchPE {
		req.Match = &struct {
			Address string            `json:"address,omitempty"`
			Port    *gona.VPCPortRange `json:"port,omitempty"`
		}{}
		if hasMatchAddr {
			req.Match.Address = matchAddr.(string)
		}
		if hasMatchPS || hasMatchPE {
			req.Match.Port = &gona.VPCPortRange{}
			if hasMatchPS {
				req.Match.Port.Start = matchPS.(int)
			}
			if hasMatchPE {
				req.Match.Port.End = matchPE.(int)
			}
		}
	}

	req.Translation = &struct {
		Address string            `json:"address"`
		Port    *gona.VPCPortRange `json:"port,omitempty"`
	}{
		Address: d.Get("translation_address").(string),
	}
	transPS, hasTransPS := d.GetOk("translation_port_start")
	transPE, hasTransPE := d.GetOk("translation_port_end")
	if hasTransPS || hasTransPE {
		req.Translation.Port = &gona.VPCPortRange{}
		if hasTransPS {
			req.Translation.Port.Start = transPS.(int)
		}
		if hasTransPE {
			req.Translation.Port.End = transPE.(int)
		}
	}

	if v, ok := d.GetOk("priority_location"); ok {
		req.Priority = &struct {
			Location        string `json:"location,omitempty"`
			AfterDnatRuleId int    `json:"afterDnatRuleId,omitempty"`
		}{Location: v.(string)}
	} else if v, ok := d.GetOk("priority_after_rule_id"); ok {
		req.Priority = &struct {
			Location        string `json:"location,omitempty"`
			AfterDnatRuleId int    `json:"afterDnatRuleId,omitempty"`
		}{AfterDnatRuleId: v.(int)}
	}

	rule, err := c.CreateVPCDNATRule(vpcID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, rule.DNATRuleID))
	log.Printf("[DEBUG] DNAT rule %d created in VPC %d, applying changes...", rule.DNATRuleID, vpcID)

	if err := c.ApplyVPCDNATChanges(vpcID); err != nil {
		return diag.Errorf("DNAT rule created but apply-changes failed: %s", err)
	}

	return resourceVPCGatewayDNATRuleRead(ctx, d, m)
}

func resourceVPCGatewayDNATRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseDNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ipVersion := d.Get("ip_version").(int)
	if ipVersion == 0 {
		ipVersion = 4
	}

	rule, err := c.GetVPCDNATRule(vpcID, ruleID, ipVersion)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] DNAT rule %d in VPC %d not found, removing from state", ruleID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("rule_id", rule.DNATRuleID, d, &diags)
	setValue("ip_version", rule.IPVersion, d, &diags)
	setValue("protocol", rule.Protocol, d, &diags)
	setValue("description", rule.Description, d, &diags)

	if rule.Match != nil {
		setValue("match_address", rule.Match.Address, d, &diags)
		if rule.Match.Port != nil {
			setValue("match_port_start", rule.Match.Port.Start, d, &diags)
			matchEnd := rule.Match.Port.End
			if matchEnd == 0 && rule.Match.Port.Start != 0 {
				matchEnd = rule.Match.Port.Start
			}
			setValue("match_port_end", matchEnd, d, &diags)
		}
	}

	if rule.Translation != nil {
		setValue("translation_address", rule.Translation.Address, d, &diags)
		if rule.Translation.Port != nil {
			setValue("translation_port_start", rule.Translation.Port.Start, d, &diags)
			transEnd := rule.Translation.Port.End
			if transEnd == 0 && rule.Translation.Port.Start != 0 {
				transEnd = rule.Translation.Port.Start
			}
			setValue("translation_port_end", transEnd, d, &diags)
		}
	}

	return diags
}

func resourceVPCGatewayDNATRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseDNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateVPCDNATRuleRequest{}

	if d.HasChange("protocol") {
		req.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}

	if d.HasChanges("match_address", "match_port_start", "match_port_end") {
		req.Match = &struct {
			Address string            `json:"address,omitempty"`
			Port    *gona.VPCPortRange `json:"port,omitempty"`
		}{
			Address: d.Get("match_address").(string),
		}
		ps := d.Get("match_port_start").(int)
		pe := d.Get("match_port_end").(int)
		if ps != 0 || pe != 0 {
			req.Match.Port = &gona.VPCPortRange{Start: ps, End: pe}
		}
	}

	if d.HasChanges("translation_address", "translation_port_start", "translation_port_end") {
		req.Translation = &struct {
			Address string            `json:"address,omitempty"`
			Port    *gona.VPCPortRange `json:"port,omitempty"`
		}{
			Address: d.Get("translation_address").(string),
		}
		ps := d.Get("translation_port_start").(int)
		pe := d.Get("translation_port_end").(int)
		if ps != 0 || pe != 0 {
			req.Translation.Port = &gona.VPCPortRange{Start: ps, End: pe}
		}
	}

	if d.HasChanges("priority_location", "priority_after_rule_id") {
		if v, ok := d.GetOk("priority_location"); ok {
			req.Priority = &struct {
				Location        string `json:"location,omitempty"`
				AfterDnatRuleId int    `json:"afterDnatRuleId,omitempty"`
			}{Location: v.(string)}
		} else if v, ok := d.GetOk("priority_after_rule_id"); ok {
			req.Priority = &struct {
				Location        string `json:"location,omitempty"`
				AfterDnatRuleId int    `json:"afterDnatRuleId,omitempty"`
			}{AfterDnatRuleId: v.(int)}
		}
	}

	if _, err := c.UpdateVPCDNATRule(vpcID, ruleID, req); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] DNAT rule %d updated in VPC %d, applying changes...", ruleID, vpcID)
	if err := c.ApplyVPCDNATChanges(vpcID); err != nil {
		return diag.Errorf("DNAT rule updated but apply-changes failed: %s", err)
	}

	return resourceVPCGatewayDNATRuleRead(ctx, d, m)
}

func resourceVPCGatewayDNATRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseDNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting DNAT rule %d from VPC %d", ruleID, vpcID)

	if err := c.DeleteVPCDNATRule(vpcID, ruleID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] DNAT rule %d in VPC %d already deleted", ruleID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	if err := c.ApplyVPCDNATChanges(vpcID); err != nil {
		return diag.Errorf("DNAT rule deleted but apply-changes failed: %s", err)
	}

	return nil
}

func resourceVPCGatewayDNATRuleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
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

func parseDNATRuleID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid DNAT rule ID %q, expected \"vpcId/ruleId\"", id)
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
