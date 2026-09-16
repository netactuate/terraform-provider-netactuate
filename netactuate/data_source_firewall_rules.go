package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceFirewallRules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallRulesRead,
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Firewall set ID whose rules will be listed",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of rules in the firewall set",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule ID",
						},
						"firewall_set_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall set ID",
						},
						"ip_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP version for this rule",
						},
						"direction": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Traffic direction for this rule",
						},
						"action": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule action",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this rule is enabled",
						},
						"match_criteria": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Match criteria for this rule",
							Elem: &schema.Resource{
								Schema: firewallMatchCriteriaDataSourceSchema(),
							},
						},
						"admin_comment": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Administrative comment for this rule",
						},
						"rule_priority": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall rule priority",
						},
						"created": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule creation timestamp",
						},
						"last_updated": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Firewall rule last update timestamp",
						},
					},
				},
			},
		},
	}
}

func firewallMatchCriteriaDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"protocol": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Protocol to match",
		},
		"source_net": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Source networks to match",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"destination_net": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Destination networks to match",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"source_port_start": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Start of source port range, or 0 when absent",
		},
		"source_port_end": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "End of source port range, or 0 when absent",
		},
		"destination_port_start": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Start of destination port range, or 0 when absent",
		},
		"destination_port_end": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "End of destination port range, or 0 when absent",
		},
		"ip_version_number": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "IP version number, or 0 when absent",
		},
		"options": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Additional match options",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"icmp_type": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "ICMP type option",
					},
				},
			},
		},
	}
}

func dataSourceFirewallRulesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	setID := d.Get("firewall_set_id").(int)

	rules, err := c.GetFirewallRules(setID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	rulesList := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		rulesList[i] = flattenFirewallRule(rule)
	}

	setValue("rules", rulesList, d, &diags)
	d.SetId(fmt.Sprintf("firewall-set-%d-rules", setID))

	return diags
}

func flattenFirewallRule(rule gona.FirewallRule) map[string]interface{} {
	return map[string]interface{}{
		"id":              rule.ID,
		"firewall_set_id": rule.FirewallSetID,
		"ip_version":      rule.IPVersion,
		"direction":       rule.Direction,
		"action":          rule.Action,
		"enabled":         rule.Enabled,
		"match_criteria":  flattenFirewallMatchCriteria(rule.MatchCriteria),
		"admin_comment":   rule.AdminComment,
		"rule_priority":   rule.RulePriority,
		"created":         rule.Created,
		"last_updated":    rule.LastUpdated,
	}
}

func flattenFirewallMatchCriteria(criteria *gona.FirewallMatchCriteria) []map[string]interface{} {
	if criteria == nil {
		return []map[string]interface{}{}
	}

	sourcePortStart := 0
	if criteria.SourcePortStart != nil {
		sourcePortStart = *criteria.SourcePortStart
	}
	sourcePortEnd := 0
	if criteria.SourcePortEnd != nil {
		sourcePortEnd = *criteria.SourcePortEnd
	}
	destinationPortStart := 0
	if criteria.DestinationPortStart != nil {
		destinationPortStart = *criteria.DestinationPortStart
	}
	destinationPortEnd := 0
	if criteria.DestinationPortEnd != nil {
		destinationPortEnd = *criteria.DestinationPortEnd
	}
	ipVersionNumber := 0
	if criteria.IPVersionNumber != nil {
		ipVersionNumber = *criteria.IPVersionNumber
	}

	return []map[string]interface{}{
		{
			"protocol":               criteria.Protocol,
			"source_net":             criteria.SourceNet,
			"destination_net":        criteria.DestinationNet,
			"source_port_start":      sourcePortStart,
			"source_port_end":        sourcePortEnd,
			"destination_port_start": destinationPortStart,
			"destination_port_end":   destinationPortEnd,
			"ip_version_number":      ipVersionNumber,
			"options":                flattenFirewallMatchOptions(criteria.Options),
		},
	}
}

func flattenFirewallMatchOptions(options *gona.FirewallMatchOptions) []map[string]interface{} {
	if options == nil {
		return []map[string]interface{}{}
	}

	return []map[string]interface{}{
		{
			"icmp_type": options.ICMPType,
		},
	}
}
