package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCGatewaySNATRules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCGatewaySNATRulesRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VPC ID whose gateway SNAT rules will be listed",
			},
			"ip_version": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "IP version whose gateway SNAT rules will be listed",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of gateway SNAT rules attached to the VPC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"snat_rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "SNAT rule ID",
						},
						"ip_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "IP version for the SNAT rule",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SNAT rule protocol",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SNAT rule description",
						},
						"match_internal_cidr": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SNAT match internal CIDR",
						},
						"translation_address_start": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SNAT translation address range start",
						},
						"translation_address_end": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SNAT translation address range end",
						},
						"translation_port_start": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "SNAT translation port range start, or 0 when no translation port is set",
						},
						"translation_port_end": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "SNAT translation port range end, matching translation_port_start for a single port rule, or 0 when no translation port is set",
						},
					},
				},
			},
		},
	}
}

func dataSourceVPCGatewaySNATRulesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	vpcID := d.Get("vpc_id").(int)
	ipVersion := d.Get("ip_version").(int)

	rules, err := c.ListVPCSNATRules(vpcID, ipVersion)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	rulesList := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		matchInternalCIDR := ""
		if rule.Match != nil {
			matchInternalCIDR = rule.Match.InternalCidr
		}

		translationAddressStart := ""
		translationAddressEnd := ""
		translationPortStart := 0
		translationPortEnd := 0
		if rule.Translation != nil {
			if rule.Translation.Address != nil {
				translationAddressStart = rule.Translation.Address.Start
				translationAddressEnd = rule.Translation.Address.End
			}
			if rule.Translation.Port != nil {
				translationPortStart = rule.Translation.Port.Start
				translationPortEnd = rule.Translation.Port.End
				if translationPortEnd == 0 {
					translationPortEnd = translationPortStart
				}
			}
		}

		rulesList[i] = map[string]interface{}{
			"snat_rule_id":              rule.SNATRuleID,
			"ip_version":                rule.IPVersion,
			"protocol":                  rule.Protocol,
			"description":               rule.Description,
			"match_internal_cidr":       matchInternalCIDR,
			"translation_address_start": translationAddressStart,
			"translation_address_end":   translationAddressEnd,
			"translation_port_start":    translationPortStart,
			"translation_port_end":      translationPortEnd,
		}
	}

	setValue("rules", rulesList, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-snat-rules-ipv%d", vpcID, ipVersion))

	return diags
}
