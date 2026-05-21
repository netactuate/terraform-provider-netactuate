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

func resourceVPCGatewaySNATRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCGatewaySNATRuleCreate,
		ReadContext:   resourceVPCGatewaySNATRuleRead,
		UpdateContext: resourceVPCGatewaySNATRuleUpdate,
		DeleteContext: resourceVPCGatewaySNATRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCGatewaySNATRuleImport,
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
				Description: "The SNAT rule ID assigned by the API",
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
				Description: "Human-readable description of the rule",
			},
			"match_internal_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Internal VPC subnet CIDR to apply SNAT to. If not set, the API defaults to ::/0 or 0.0.0.0/0 (match all).",
			},
			"translation_address_start": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Lower bound of the public IP range for SNAT translation. If not set, the API defaults to the bastion IP.",
			},
			"translation_address_end": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Upper bound of the public IP range for SNAT translation. If not set, the API defaults to the bastion IP.",
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
				Description:   "Place this rule after the given SNAT rule ID",
			},
		},
	}
}

func resourceVPCGatewaySNATRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)

	req := &gona.CreateVPCSNATRuleRequest{
		IPVersion: d.Get("ip_version").(int),
	}

	if v, ok := d.GetOk("protocol"); ok {
		req.Protocol = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	// Match
	if v, ok := d.GetOk("match_internal_cidr"); ok {
		req.Match = &struct {
			InternalCidr string `json:"internalCidr,omitempty"`
		}{
			InternalCidr: v.(string),
		}
	}

	// Translation
	addrStart, hasAddrStart := d.GetOk("translation_address_start")
	addrEnd, hasAddrEnd := d.GetOk("translation_address_end")
	portStart, hasPortStart := d.GetOk("translation_port_start")
	portEnd, hasPortEnd := d.GetOk("translation_port_end")

	if hasAddrStart || hasAddrEnd || hasPortStart || hasPortEnd {
		req.Translation = &struct {
			Address *struct {
				Start string `json:"start,omitempty"`
				End   string `json:"end,omitempty"`
			} `json:"address,omitempty"`
			Port *gona.VPCPortRange `json:"port,omitempty"`
		}{}

		if hasAddrStart || hasAddrEnd {
			req.Translation.Address = &struct {
				Start string `json:"start,omitempty"`
				End   string `json:"end,omitempty"`
			}{}
			if hasAddrStart {
				req.Translation.Address.Start = addrStart.(string)
			}
			if hasAddrEnd {
				req.Translation.Address.End = addrEnd.(string)
			}
		}

		if hasPortStart || hasPortEnd {
			req.Translation.Port = &gona.VPCPortRange{}
			if hasPortStart {
				req.Translation.Port.Start = portStart.(int)
			}
			if hasPortEnd {
				req.Translation.Port.End = portEnd.(int)
			}
		}
	}

	if v, ok := d.GetOk("priority_location"); ok {
		req.Priority = &struct {
			Location        string `json:"location,omitempty"`
			AfterSnatRuleId int    `json:"afterSnatRuleId,omitempty"`
		}{Location: v.(string)}
	} else if v, ok := d.GetOk("priority_after_rule_id"); ok {
		req.Priority = &struct {
			Location        string `json:"location,omitempty"`
			AfterSnatRuleId int    `json:"afterSnatRuleId,omitempty"`
		}{AfterSnatRuleId: v.(int)}
	}

	rule, err := c.CreateVPCSNATRule(vpcID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, rule.SNATRuleID))
	log.Printf("[DEBUG] SNAT rule %d created in VPC %d, applying changes...", rule.SNATRuleID, vpcID)

	if err := c.ApplyVPCSNATChanges(vpcID); err != nil {
		return diag.Errorf("SNAT rule created but apply-changes failed: %s", err)
	}

	return resourceVPCGatewaySNATRuleRead(ctx, d, m)
}

func resourceVPCGatewaySNATRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseSNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ipVersion := d.Get("ip_version").(int)
	if ipVersion == 0 {
		ipVersion = 4
	}

	rule, err := c.GetVPCSNATRule(vpcID, ruleID, ipVersion)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SNAT rule %d in VPC %d not found, removing from state", ruleID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("rule_id", rule.SNATRuleID, d, &diags)
	setValue("ip_version", rule.IPVersion, d, &diags)
	setValue("protocol", rule.Protocol, d, &diags)
	setValue("description", rule.Description, d, &diags)

	if rule.Match != nil {
		setValue("match_internal_cidr", rule.Match.InternalCidr, d, &diags)
	}

	if rule.Translation != nil {
		if rule.Translation.Address != nil {
			setValue("translation_address_start", rule.Translation.Address.Start, d, &diags)
			// API may omit end when it is equivalent to start for single-IP translation.
			addrEnd := rule.Translation.Address.End
			if addrEnd == "" && rule.Translation.Address.Start != "" {
				addrEnd = rule.Translation.Address.Start
			}
			setValue("translation_address_end", addrEnd, d, &diags)
		}
		if rule.Translation.Port != nil {
			setValue("translation_port_start", rule.Translation.Port.Start, d, &diags)
			// API returns null for end when it equals start
			portEnd := rule.Translation.Port.End
			if portEnd == 0 && rule.Translation.Port.Start != 0 {
				portEnd = rule.Translation.Port.Start
			}
			setValue("translation_port_end", portEnd, d, &diags)
		}
	}

	return diags
}

func resourceVPCGatewaySNATRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseSNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateVPCSNATRuleRequest{}

	if d.HasChange("protocol") {
		req.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}

	if d.HasChange("match_internal_cidr") {
		req.Match = &struct {
			InternalCidr string `json:"internalCidr,omitempty"`
		}{
			InternalCidr: d.Get("match_internal_cidr").(string),
		}
	}

	if d.HasChanges("translation_address_start", "translation_address_end", "translation_port_start", "translation_port_end") {
		req.Translation = &struct {
			Address *struct {
				Start string `json:"start,omitempty"`
				End   string `json:"end,omitempty"`
			} `json:"address,omitempty"`
			Port *gona.VPCPortRange `json:"port,omitempty"`
		}{}

		addrStart := d.Get("translation_address_start").(string)
		addrEnd := d.Get("translation_address_end").(string)
		if addrStart != "" || addrEnd != "" {
			req.Translation.Address = &struct {
				Start string `json:"start,omitempty"`
				End   string `json:"end,omitempty"`
			}{
				Start: addrStart,
				End:   addrEnd,
			}
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
				AfterSnatRuleId int    `json:"afterSnatRuleId,omitempty"`
			}{Location: v.(string)}
		} else if v, ok := d.GetOk("priority_after_rule_id"); ok {
			req.Priority = &struct {
				Location        string `json:"location,omitempty"`
				AfterSnatRuleId int    `json:"afterSnatRuleId,omitempty"`
			}{AfterSnatRuleId: v.(int)}
		}
	}

	if _, err := c.UpdateVPCSNATRule(vpcID, ruleID, req); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] SNAT rule %d updated in VPC %d, applying changes...", ruleID, vpcID)
	if err := c.ApplyVPCSNATChanges(vpcID); err != nil {
		return diag.Errorf("SNAT rule updated but apply-changes failed: %s", err)
	}

	return resourceVPCGatewaySNATRuleRead(ctx, d, m)
}

func resourceVPCGatewaySNATRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, ruleID, err := parseSNATRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting SNAT rule %d from VPC %d", ruleID, vpcID)

	if err := c.DeleteVPCSNATRule(vpcID, ruleID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SNAT rule %d in VPC %d already deleted", ruleID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	if err := c.ApplyVPCSNATChanges(vpcID); err != nil {
		return diag.Errorf("SNAT rule deleted but apply-changes failed: %s", err)
	}

	return nil
}

func resourceVPCGatewaySNATRuleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
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

func parseSNATRuleID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid SNAT rule ID %q, expected \"vpcId/ruleId\"", id)
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
