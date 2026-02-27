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

func resourceRouterVRFSNATRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFSNATRuleCreate,
		ReadContext:   resourceRouterVRFSNATRuleRead,
		UpdateContext: resourceRouterVRFSNATRuleUpdate,
		DeleteContext: resourceRouterVRFSNATRuleDelete,
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
			"snat_rule_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The SNAT rule ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
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
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Start of the match port range",
			},
			"match_port_end": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "End of the match port range",
			},
			"translation_network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP network you want to translate traffic to",
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
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"start", "end", "after"}, false)),
				Description:      "Place this rule at 'start', 'end', or 'after' another rule",
			},
			"priority_after_snat_rule_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Place this rule after the given SNAT rule ID",
			},
		},
	}
}

func buildSNATPortRange(d *schema.ResourceData, startKey, endKey string) *gona.VPCPortRange {
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

func buildSNATMatchConfig(d *schema.ResourceData) *struct {
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
		Port:        buildSNATPortRange(d, "match_port_start", "match_port_end"),
	}
}

func buildSNATTranslationConfig(d *schema.ResourceData) *struct {
	Network string           `json:"network"`
	Port    *gona.VPCPortRange `json:"port,omitempty"`
} {
	return &struct {
		Network string           `json:"network"`
		Port    *gona.VPCPortRange `json:"port,omitempty"`
	}{
		Network: d.Get("translation_network").(string),
		Port:    buildSNATPortRange(d, "translation_port_start", "translation_port_end"),
	}
}

func buildSNATPriorityConfig(d *schema.ResourceData) *struct {
	Location        string `json:"location,omitempty"`
	AfterSnatRuleId *int    `json:"afterSnatRuleId,omitempty"`
} {

	var afterSnatRuleID *int
	if v, ok := d.GetOk("priority_after_snat_rule_id"); ok {
		val := v.(int)
		afterSnatRuleID = &val
	}
	location := d.Get("priority_location").(string)

	return &struct {
		Location        string `json:"location,omitempty"`
		AfterSnatRuleId *int    `json:"afterSnatRuleId,omitempty"`
	}{
		Location:        location,
		AfterSnatRuleId: afterSnatRuleID,
	}

}

func resourceRouterVRFSNATRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	req := &gona.CreateRouterVRFSNATRuleRequest{
		IPVersion:   d.Get("ip_version").(int),
		Protocol:    d.Get("protocol").(string),
		Match:       buildSNATMatchConfig(d),
		Translation: buildSNATTranslationConfig(d),
		Priority:    buildSNATPriorityConfig(d),
	}

	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	rule, err := c.CreateRouterVRFSNATRule(routerID, vrfID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	d.SetId(strconv.Itoa(rule.SNATRuleID))
	setValue("ip_version_computed", ipVersion, d, &diags)
	return resourceRouterVRFSNATRuleRead(ctx, d, m)
}

func resourceRouterVRFSNATRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version_computed").(int)
	snatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid SNAT rule ID: %w", err))
	}

	rule, err := c.GetRouterVRFSNATRule(routerID, vrfID, snatRuleID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ip_version", ipVersion, d, &diags)
	setValue("snat_rule_id", rule.SNATRuleID, d, &diags)
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
		setValue("priority_after_snat_rule_id", rule.Priority.AfterSnatRuleId, d, &diags)
	}

	return diags
}

func resourceRouterVRFSNATRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipVersion := d.Get("ip_version_computed").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	snatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid SNAT rule ID: %w", err))
	}

	req := &gona.UpdateRouterVRFSNATRuleRequest{
		IPVersion: ipVersion,
		Protocol:    d.Get("protocol").(string),
		Description: d.Get("description").(string),
		Match:       buildSNATMatchConfig(d),
		Translation: buildSNATTranslationConfig(d),
		Priority:    buildSNATPriorityConfig(d),
	}

	_, err = c.UpdateRouterVRFSNATRule(routerID, vrfID, snatRuleID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ip_version_computed", ipVersion, d, &diags)
	return resourceRouterVRFSNATRuleRead(ctx, d, m)
}

func resourceRouterVRFSNATRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	snatRuleID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid SNAT rule ID: %w", err))
	}

	err = c.DeleteRouterVRFSNATRule(routerID, vrfID, snatRuleID)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}