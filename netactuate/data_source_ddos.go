package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceDDoSAttacks() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDDoSAttacksRead,
		Schema: map[string]*schema.Schema{
			"attacks": ddosAttackListSchema("DDoS attack records returned by the API."),
		},
	}
}

func dataSourceDDoSActiveAttacks() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDDoSActiveAttacksRead,
		Schema: map[string]*schema.Schema{
			"attacks": ddosAttackListSchema("Active DDoS attack records returned by the API."),
		},
	}
}

func ddosAttackListSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"attack_id": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "DDoS attack record ID returned by the API id field.",
				},
				"date_start": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Attack start timestamp returned by the API.",
				},
				"date_end": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Attack end timestamp returned by the API. This may be empty for an unfinished attack.",
				},
				"status": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Opaque attack status code returned by the API. The provider does not map this undocumented code to a label.",
				},
				"ip": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Target IP address returned by the API.",
				},
				"prefix": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Target prefix in CIDR notation returned by the API.",
				},
				"direction": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Attack traffic direction returned by the API.",
				},
				"pps": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Peak packets per second returned by the API.",
				},
				"rule_id": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Mitigation rule ID returned by the API for this attack.",
				},
				"rule_type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Mitigation rule type returned by the API.",
				},
				"ban_duration": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Ban duration in seconds returned by the API.",
				},
				"rule_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Mitigation rule name returned by the API.",
				},
			},
		},
	}
}

func dataSourceDDoSAttacksRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	attacks, err := c.GetDDoSAttacks()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("attacks", flattenDDoSAttacks(attacks), d, &diags)
	d.SetId("ddos-attacks")
	return diags
}

func dataSourceDDoSActiveAttacksRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	attacks, err := c.GetDDoSActiveAttacks()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	// The active attack shape is shared with all attacks because active was empty when measured.
	setValue("attacks", flattenDDoSAttacks(attacks), d, &diags)
	d.SetId("ddos-active-attacks")
	return diags
}

func dataSourceDDoSDashboard() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDDoSDashboardRead,
		Schema: map[string]*schema.Schema{
			"period": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Dashboard period in seconds. When configured, this is sent as the period query parameter.",
			},
			"include_ended": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether ended attacks should be included. When configured, this is sent as the include_ended query parameter.",
			},
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of dashboard entries requested. When configured, this is sent as the limit query parameter.",
			},
			"total_attacks": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total attack count returned by the dashboard endpoint.",
			},
			"active_rules": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Active rule count returned by the dashboard endpoint.",
			},
			"longest_attack_seconds": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Longest attack duration in seconds returned by the dashboard endpoint.",
			},
			"top_attacks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Top attack entries returned by the dashboard endpoint. The element shape is intentionally minimal because the endpoint can return an empty list.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{},
				},
			},
		},
	}
}

func dataSourceDDoSDashboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	opts := gona.DDoSDashboardOptions{}
	if value, ok := d.GetOkExists("period"); ok {
		period := value.(int)
		opts.Period = &period
	}
	if value, ok := d.GetOkExists("include_ended"); ok {
		includeEnded := value.(bool)
		opts.IncludeEnded = &includeEnded
	}
	if value, ok := d.GetOkExists("limit"); ok {
		limit := value.(int)
		opts.Limit = &limit
	}

	dashboard, err := c.GetDDoSDashboard(opts)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("total_attacks", dashboard.TotalAttacks, d, &diags)
	setValue("active_rules", dashboard.ActiveRules, d, &diags)
	setValue("longest_attack_seconds", dashboard.LongestAttackSeconds, d, &diags)
	// Top attacks were empty when measured, so the provider does not invent fields.
	setValue("top_attacks", flattenDDoSDashboardAttacks(dashboard.TopAttacks), d, &diags)
	setValue("period", dashboard.Period, d, &diags)
	d.SetId(fmt.Sprintf("ddos-dashboard-period-%s-include-ended-%s-limit-%s",
		optionalIntIDPart(opts.Period),
		optionalBoolIDPart(opts.IncludeEnded),
		optionalIntIDPart(opts.Limit)))
	return diags
}

func dataSourceDDoSRules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDDoSRulesRead,
		Schema: map[string]*schema.Schema{
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "DDoS mitigation rules returned by the API.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "DDoS mitigation rule ID returned by the API id field.",
						},
						"rule_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DDoS mitigation rule name returned by the API.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "DDoS mitigation rule description returned by the API.",
						},
						"prefixes": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Prefixes associated with this DDoS mitigation rule.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"prefix_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Prefix record ID returned by the API id field.",
									},
									"prefix": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Prefix in CIDR notation returned by the API.",
									},
									"prefix_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Prefix type returned by the API.",
									},
									"description": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Prefix description returned by the API.",
									},
									"allowed_pps": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Allowed packets per second returned by the API.",
									},
								},
							},
						},
						"rules": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Rule actions associated with this DDoS mitigation rule.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Rule action name returned by the API.",
									},
									"action_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Rule action type returned by the API.",
									},
									"run_order": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Rule action run order returned by the API.",
									},
									"action_name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Rule action display name returned by the API.",
									},
									"action_description": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Rule action description returned by the API.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceDDoSRule() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDDoSRuleRead,
		Schema: map[string]*schema.Schema{
			"rule_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "DDoS mitigation rule ID.",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "DDoS mitigation rule returned by the API.",
				Elem: &schema.Resource{
					Schema: ddosRuleSchema(),
				},
			},
		},
	}
}

func ddosRuleSchema() map[string]*schema.Schema {
	return dataSourceDDoSRules().Schema["rules"].Elem.(*schema.Resource).Schema
}

func dataSourceDDoSRulesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	rules, err := c.GetDDoSRules()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("rules", flattenDDoSRules(rules), d, &diags)
	d.SetId("ddos-rules")
	return diags
}

func dataSourceDDoSRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	ruleID := d.Get("rule_id").(int)

	rule, err := c.GetDDoSRule(ruleID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("rules", flattenDDoSRules([]gona.DDoSRule{*rule}), d, &diags)
	d.SetId(fmt.Sprintf("ddos-rule-%d", ruleID))
	return diags
}

func flattenDDoSAttacks(attacks []gona.DDoSAttack) []map[string]interface{} {
	out := make([]map[string]interface{}, len(attacks))
	for i, attack := range attacks {
		out[i] = map[string]interface{}{
			"attack_id":    attack.AttackID,
			"date_start":   attack.DateStart,
			"date_end":     attack.DateEnd,
			"status":       attack.Status,
			"ip":           attack.IP,
			"prefix":       attack.Prefix,
			"direction":    attack.Direction,
			"pps":          attack.PPS,
			"rule_id":      attack.RuleID,
			"rule_type":    attack.RuleType,
			"ban_duration": attack.BanDuration,
			"rule_name":    attack.RuleName,
		}
	}
	return out
}

func flattenDDoSDashboardAttacks(attacks []gona.DDoSDashboardAttack) []map[string]interface{} {
	out := make([]map[string]interface{}, len(attacks))
	for i := range attacks {
		out[i] = map[string]interface{}{}
	}
	return out
}

func flattenDDoSRules(rules []gona.DDoSRule) []map[string]interface{} {
	out := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		out[i] = map[string]interface{}{
			"rule_id":     rule.RuleID,
			"rule_name":   rule.RuleName,
			"description": rule.Description,
			"prefixes":    flattenDDoSRulePrefixes(rule.Prefixes),
			"rules":       flattenDDoSRuleActions(rule.Rules),
		}
	}
	return out
}

func flattenDDoSRulePrefixes(prefixes []gona.DDoSRulePrefix) []map[string]interface{} {
	out := make([]map[string]interface{}, len(prefixes))
	for i, prefix := range prefixes {
		out[i] = map[string]interface{}{
			"prefix_id":   prefix.PrefixID,
			"prefix":      prefix.Prefix,
			"prefix_type": prefix.PrefixType,
			"description": prefix.Description,
			"allowed_pps": prefix.AllowedPPS,
		}
	}
	return out
}

func flattenDDoSRuleActions(actions []gona.DDoSRuleAction) []map[string]interface{} {
	out := make([]map[string]interface{}, len(actions))
	for i, action := range actions {
		out[i] = map[string]interface{}{
			"name":               action.Name,
			"action_type":        action.ActionType,
			"run_order":          action.RunOrder,
			"action_name":        action.ActionName,
			"action_description": action.ActionDescription,
		}
	}
	return out
}

func optionalIntIDPart(value *int) string {
	if value == nil {
		return "unset"
	}
	return fmt.Sprintf("%d", *value)
}

func optionalBoolIDPart(value *bool) string {
	if value == nil {
		return "unset"
	}
	return fmt.Sprintf("%t", *value)
}
