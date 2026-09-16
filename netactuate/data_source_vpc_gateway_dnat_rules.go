package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCGatewayDNATRules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCGatewayDNATRulesRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VPC ID whose gateway DNAT rules will be listed",
			},
			"ip_version": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "IP version whose gateway DNAT rules will be listed",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of gateway DNAT rules attached to the VPC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dnat_rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DNAT rule ID",
						},
						"ip_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "IP version for the DNAT rule",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNAT rule protocol",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNAT rule description",
						},
						"match_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNAT match address",
						},
						"match_port_start": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DNAT match port range start, or 0 when no match port is set",
						},
						"match_port_end": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DNAT match port range end, matching match_port_start for a single port rule, or 0 when no match port is set",
						},
						"translation_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DNAT translation address",
						},
						"translation_port_start": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DNAT translation port range start, or 0 when no translation port is set",
						},
						"translation_port_end": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DNAT translation port range end, matching translation_port_start for a single port rule, or 0 when no translation port is set",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCGatewayDNATRulesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	vpcID := d.Get("vpc_id").(int)
	ipVersion := d.Get("ip_version").(int)

	rules, err := c.ListVPCDNATRules(vpcID, ipVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	rulesList := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		matchAddress := ""
		matchPortStart := 0
		matchPortEnd := 0
		if rule.Match != nil {
			matchAddress = rule.Match.Address
			if rule.Match.Port != nil {
				matchPortStart = rule.Match.Port.Start
				matchPortEnd = rule.Match.Port.End
				if matchPortEnd == 0 {
					matchPortEnd = matchPortStart
				}
			}
		}

		translationAddress := ""
		translationPortStart := 0
		translationPortEnd := 0
		if rule.Translation != nil {
			translationAddress = rule.Translation.Address
			if rule.Translation.Port != nil {
				translationPortStart = rule.Translation.Port.Start
				translationPortEnd = rule.Translation.Port.End
				if translationPortEnd == 0 {
					translationPortEnd = translationPortStart
				}
			}
		}

		rulesList[i] = map[string]interface{}{
			"dnat_rule_id":           rule.DNATRuleID,
			"ip_version":             rule.IPVersion,
			"protocol":               rule.Protocol,
			"description":            rule.Description,
			"match_address":          matchAddress,
			"match_port_start":       matchPortStart,
			"match_port_end":         matchPortEnd,
			"translation_address":    translationAddress,
			"translation_port_start": translationPortStart,
			"translation_port_end":   translationPortEnd,
		}
	}

	setValue("rules", rulesList, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-dnat-rules-ipv%d", vpcID, ipVersion))

	return diags
}
