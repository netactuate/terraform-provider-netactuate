package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFDNATRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFDNATRuleCreate,
		ReadContext:   resourceRouterVRFDNATRuleRead,
		UpdateContext: resourceRouterVRFDNATRuleUpdate,
		DeleteContext: resourceRouterVRFDNATRuleDelete,
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
            matchPortStart, hasMatchStart := diff.GetOk("match_port_start")
            matchPortEnd, hasMatchEnd := diff.GetOk("match_port_end")
            if hasMatchStart && hasMatchEnd {
                startVal := matchPortStart.(int)
                endVal := matchPortEnd.(int)
                if startVal == endVal {
                    return fmt.Errorf("match_port_start and match_port_end have the same value (%d). For a single port, only set match_port_start and omit match_port_end", startVal)
                }
            }

            translationPortStart, hasTranslationStart := diff.GetOk("translation_port_start")
            translationPortEnd, hasTranslationEnd := diff.GetOk("translation_port_end")
            if hasTranslationStart && hasTranslationEnd {
                startVal := translationPortStart.(int)
                endVal := translationPortEnd.(int)
                if startVal == endVal {
                    return fmt.Errorf("translation_port_start and translation_port_end have the same value (%d). For a single port, only set translation_port_start and omit translation_port_end", startVal)
                }
            }
            return nil
        },
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router",
			},
			"vrf_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VRF",
			},
			"dnat_rule_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The DNAT rule ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:        true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "IP version for this rule (4 or 6)",
			},
			"ip_version_computed": {
				Type:             schema.TypeInt,
				Computed:    	  true,
				Description:      "The value is computed based on the 'ip_version' field",
			},
			"protocol": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"TCP", "UDP", "ICMP"}, false)),
				Description:      "Protocol to match (TCP, UDP, or ICMP)",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Human-readable description of the rule",
			},
			"match_interface_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The interface in which the traffic is going out",
			},
			"match_network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP network to match against for outbound traffic",
			},
			"match_port_start": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "Start of the match port range (1-32000)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 32000)),
			},
			"match_port_end": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "End of the match port range (1-32000)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 32000)),
			},
			"translation_network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP network you want to translate traffic to",
			},
			"translation_port_start": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "Start of the translation port range (1-32000)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 32000)),
			},
			"translation_port_end": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "End of the translation port range (1-32000)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 32000)),
			},
			"priority_location": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"start", "end", "after"}, false)),
				Description:      "Place this rule at 'start', 'end', or 'after' another rule",
			},
			"priority_after_dnat_rule_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Place this rule after the given DNAT rule ID",
			},
		},
	}
}

func buildDNATPortRange(d *schema.ResourceData, startKey, endKey string) *gona.VPCPortRange {
	start, hasStart := d.GetOk(startKey)
	end, hasEnd := d.GetOk(endKey)

	portRange := &gona.VPCPortRange{}
	if hasStart {
		portRange.Start = start.(int)
	}
	if hasEnd {
		portRange.End = end.(int)
	}
	return portRange
}

func buildDNATMatchConfig(d *schema.ResourceData) *struct {
	InterfaceID int              `json:"interfaceId"`
	Network     string           `json:"network"`
	Port        *gona.VPCPortRange `json:"port,omitempty"`
} {
	return &struct {
		InterfaceID int              `json:"interfaceId"`
		Network     string           `json:"network"`
		Port        *gona.VPCPortRange `json:"port,omitempty"`
	}{
		InterfaceID: d.Get("match_interface_id").(int),
		Network:     d.Get("match_network").(string),
		Port:        buildDNATPortRange(d, "match_port_start", "match_port_end"),
	}
}

func buildDNATTranslationConfig(d *schema.ResourceData) *struct {
	Network string           `json:"network"`
	Port    *gona.VPCPortRange `json:"port,omitempty"`
} {
	return &struct {
		Network string           `json:"network"`
		Port    *gona.VPCPortRange `json:"port,omitempty"`
	}{
		Network: d.Get("translation_network").(string),
		Port:    buildDNATPortRange(d, "translation_port_start", "translation_port_end"),
	}
}

func buildDNATPriorityConfig(d *schema.ResourceData) *struct {
	Location        string `json:"location,omitempty"`
	AfterDnatRuleId *int    `json:"afterDnatRuleId,omitempty"`
} {

	var afterDnatRuleID *int
	if v, ok := d.GetOk("priority_after_dnat_rule_id"); ok {
		val := v.(int)
		afterDnatRuleID = &val
	}
	location := d.Get("priority_location").(string)

	return &struct {
		Location        string `json:"location,omitempty"`
		AfterDnatRuleId *int    `json:"afterDnatRuleId,omitempty"`
	}{
		Location:        location,
		AfterDnatRuleId: afterDnatRuleID,
	}

}

func resourceRouterVRFDNATRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	req := &gona.CreateRouterVRFDNATRuleRequest{
		IPVersion:   ipVersion,
		Protocol:    d.Get("protocol").(string),
		Match:       buildDNATMatchConfig(d),
		Translation: buildDNATTranslationConfig(d),
		Priority:    buildDNATPriorityConfig(d),
	}

	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	rule, err := c.CreateRouterVRFDNATRule(routerID, vrfID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	d.SetId(strconv.Itoa(rule.DNATRuleID))
	setValue("ip_version_computed", ipVersion, d, &diags)
	return resourceRouterVRFDNATRuleRead(ctx, d, m)
}

func resourceRouterVRFDNATRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version_computed").(int)
	dnatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid DNAT rule ID: %w", err))
	}

	rule, err := c.GetRouterVRFDNATRule(routerID, vrfID, dnatRuleID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ip_version", ipVersion, d, &diags)
	setValue("dnat_rule_id", rule.DNATRuleID, d, &diags)
	setValue("protocol", rule.Protocol, d, &diags)
	setValue("description", rule.Description, d, &diags)

	if rule.Match != nil {
		setValue("match_interface_id", rule.Match.InterfaceID, d, &diags)
		setValue("match_network", rule.Match.Network, d, &diags)
		if rule.Match.Port != nil {
			setValue("match_port_start", rule.Match.Port.Start, d, &diags)
			setValue("match_port_end", rule.Match.Port.End, d, &diags)
		}
	}

	if rule.Translation != nil {
		setValue("translation_network", rule.Translation.Network, d, &diags)
		if rule.Translation.Port != nil {
			setValue("translation_port_start", rule.Translation.Port.Start, d, &diags)
			setValue("translation_port_end", rule.Translation.Port.End, d, &diags)
		}
	}

	if rule.Priority != nil {
		setValue("priority_location", rule.Priority.Location, d, &diags)
		setValue("priority_after_dnat_rule_id", rule.Priority.AfterDnatRuleId, d, &diags)
	}

	return diags
}

func resourceRouterVRFDNATRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version_computed").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	dnatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid DNAT rule ID: %w", err))
	}

	req := &gona.UpdateRouterVRFDNATRuleRequest{
		IPVersion: ipVersion,
		Protocol:    d.Get("protocol").(string),
		Description: d.Get("description").(string),
		Match:       buildDNATMatchConfig(d),
		Translation: buildDNATTranslationConfig(d),
		Priority:    buildDNATPriorityConfig(d),
	}

	_, err = c.UpdateRouterVRFDNATRule(routerID, vrfID, dnatRuleID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ip_version_computed", ipVersion, d, &diags)
	return resourceRouterVRFDNATRuleRead(ctx, d, m)
}

func resourceRouterVRFDNATRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	dnatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid DNAT rule ID: %w", err))
	}

	err = c.DeleteRouterVRFDNATRule(routerID, vrfID, dnatRuleID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}