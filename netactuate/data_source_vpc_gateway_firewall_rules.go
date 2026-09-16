package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCGatewayFirewallRules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCGatewayFirewallRulesRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VPC ID whose gateway firewall rules will be listed",
			},
			"ip_version": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "IP version whose gateway firewall rules will be listed",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of gateway firewall rules attached to the VPC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"firewall_rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule ID",
						},
						"ip_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "IP version for the firewall rule",
						},
						"direction": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule direction",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule protocol",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule description",
						},
						"network": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule network",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule address",
						},
						"prefix_length": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule prefix length",
						},
						"port_start": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule port range start, or 0 when no port is set",
						},
						"port_end": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule port range end, matching port_start for a single port rule, or 0 when no port is set",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCGatewayFirewallRulesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	vpcID := d.Get("vpc_id").(int)
	ipVersion := d.Get("ip_version").(int)

	rules, err := c.ListVPCFirewallRules(vpcID, ipVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	rulesList := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		portStart := 0
		portEnd := 0
		if rule.Port != nil {
			portStart = rule.Port.Start
			portEnd = rule.Port.End
			if portEnd == 0 {
				portEnd = portStart
			}
		}

		rulesList[i] = map[string]interface{}{
			"firewall_rule_id": rule.FirewallRuleID,
			"ip_version":       rule.IPVersion,
			"direction":        rule.Direction,
			"protocol":         rule.Protocol,
			"description":      rule.Description,
			"network":          rule.Network,
			"address":          rule.Address,
			"prefix_length":    rule.PrefixLength,
			"port_start":       portStart,
			"port_end":         portEnd,
		}
	}

	setValue("rules", rulesList, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-firewall-rules-ipv%d", vpcID, ipVersion))

	return diags
}
