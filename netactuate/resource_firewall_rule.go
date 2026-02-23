package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"firewallSetMu serializes draft/publish operations per firewall set.
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

// Mutex to prevent multidraft
var firewallSetMu sync.Map

func lockFirewallSet(setID int) func() {
	val, _ := firewallSetMu.LoadOrStore(setID, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallRuleCreate,
		ReadContext:   resourceFirewallRuleRead,
		UpdateContext: resourceFirewallRuleUpdate,
		DeleteContext: resourceFirewallRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceFirewallRuleImport,
		},
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the parent firewall set",
			},
			"rule_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The firewall rule ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"IPv4", "IPv6"}, false)),
				Description:      "IP version for this rule (IPv4 or IPv6)",
			},
			"direction": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "IN",
				Description: "Traffic direction (always IN per backend)",
			},
			"action": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"ACCEPT", "DROP"}, false)),
				Description:      "Rule action (ACCEPT or DROP)",
			},
			"protocol": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"tcp", "udp", "icmp"}, false)),
				Description:      "Protocol to match (tcp, udp, or icmp)",
			},
			"icmp_type": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"echo-request", "echo-reply"}, false)),
				Description:      "ICMP type (echo-request or echo-reply), only when protocol is icmp",
			},
			"source_port_start": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 65535),
				Description:  "Start of source port range",
			},
			"source_port_end": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 65535),
				Description:  "End of source port range",
			},
			"destination_port_start": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 65535),
				Description:  "Start of destination port range",
			},
			"destination_port_end": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 65535),
				Description:  "End of destination port range",
			},
			"source_net": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Source CIDRs to match",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"destination_net": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Destination CIDRs to match",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"admin_comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Comment for this rule",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether this rule is active",
			},
			"rule_priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Rule priority (auto-assigned if not set)",
			},
			"sync_after_publish": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to sync rules to attached VMs after publishing",
			},
		},
	}
}

func buildFirewallRuleRequest(d *schema.ResourceData) *gona.CreateFirewallRuleRequest {
	req := &gona.CreateFirewallRuleRequest{
		IPVersion: d.Get("ip_version").(string),
		Action:    d.Get("action").(string),
		Enabled:   d.Get("enabled").(bool),
	}

	if v, ok := d.GetOk("direction"); ok {
		req.Direction = v.(string)
	}

	if v, ok := d.GetOk("admin_comment"); ok {
		req.AdminComment = v.(string)
	}

	if v, ok := d.GetOk("rule_priority"); ok {
		p := v.(int)
		req.RulePriority = &p
	}

	// Build match_criteria
	mc := &gona.FirewallMatchCriteria{}

	if v, ok := d.GetOk("protocol"); ok {
		mc.Protocol = v.(string)
	}

	if v, ok := d.GetOk("source_port_start"); ok {
		p := v.(int)
		mc.SourcePortStart = &p
	}
	if v, ok := d.GetOk("source_port_end"); ok {
		p := v.(int)
		mc.SourcePortEnd = &p
	}
	if v, ok := d.GetOk("destination_port_start"); ok {
		p := v.(int)
		mc.DestinationPortStart = &p
	}
	if v, ok := d.GetOk("destination_port_end"); ok {
		p := v.(int)
		mc.DestinationPortEnd = &p
	}

	if v, ok := d.GetOk("source_net"); ok {
		raw := v.([]interface{})
		nets := make([]string, len(raw))
		for i, n := range raw {
			nets[i] = n.(string)
		}
		mc.SourceNet = nets
	}

	if v, ok := d.GetOk("destination_net"); ok {
		raw := v.([]interface{})
		nets := make([]string, len(raw))
		for i, n := range raw {
			nets[i] = n.(string)
		}
		mc.DestinationNet = nets
	}

	if v, ok := d.GetOk("icmp_type"); ok {
		mc.Options = &gona.FirewallMatchOptions{
			ICMPType: v.(string),
		}
	}

	// Always send match_criteria (API requires it, even if empty)
	req.MatchCriteria = mc

	return req
}

// getOrCreateDraft returns the draft set ID, creating one if necessary. (clean up draft on error)
func getOrCreateDraft(c *gona.Client, setID int) (int, error) {
	set, err := c.GetFirewallSet(setID)
	if err != nil {
		return 0, fmt.Errorf("reading firewall set: %w", err)
	}

	if set.DraftFirewallSetID != nil && *set.DraftFirewallSetID > 0 {
		log.Printf("[DEBUG] Using existing draft %d for set %d", *set.DraftFirewallSetID, setID)
		return *set.DraftFirewallSetID, nil
	}

	draft, err := c.CreateDraftFirewallSet(setID)
	if err != nil {
		return 0, fmt.Errorf("creating draft: %w", err)
	}

	log.Printf("[DEBUG] Created draft %d for set %d", draft.ID, setID)
	return draft.ID, nil
}

func findRuleByPriority(c *gona.Client, setID, priority int) (*gona.FirewallRule, error) {
	rules, err := c.GetFirewallRules(setID)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		if r.RulePriority == priority {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("rule with priority %d not found", priority)
}

func resourceFirewallRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID := d.Get("firewall_set_id").(int)
	unlock := lockFirewallSet(setID)
	defer unlock()

	//Get or create a draft
	draftID, err := getOrCreateDraft(c, setID)
	if err != nil {
		return diag.FromErr(err)
	}

	//Create rule on the draft
	req := buildFirewallRuleRequest(d)
	createdRule, err := c.CreateFirewallRule(draftID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating rule: %w", err))
	}
	log.Printf("[DEBUG] Created rule %d (priority %d) on draft %d", createdRule.ID, createdRule.RulePriority, draftID)

	//Publish the draft
	if _, err := c.PublishDraftFirewallSet(draftID); err != nil {
		return diag.FromErr(fmt.Errorf("publishing draft: %w", err))
	}

	//Find the new rule on the original set by its priority
	newRule, err := findRuleByPriority(c, setID, createdRule.RulePriority)
	if err != nil {
		return diag.FromErr(fmt.Errorf("rule not found after publish: %w", err))
	}

	// Set composite ID
	d.SetId(fmt.Sprintf("%d/%d", setID, newRule.ID))
	log.Printf("[DEBUG] Firewall rule created: set=%d rule=%d priority=%d", setID, newRule.ID, newRule.RulePriority)

	//Sync if requested
	if d.Get("sync_after_publish").(bool) {
		if err := c.SyncFirewallSetRules(setID); err != nil {
			return diag.Errorf("sync failed after create: %s", err)
		}
	}

	return resourceFirewallRuleRead(ctx, d, m)
}

func resourceFirewallRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID, ruleID, err := parseFirewallSetRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Try direct lookup by stored rule ID
	rule, err := c.GetFirewallRule(setID, ruleID)
	if err != nil {
		// Rule ID may be stale after another rule's draft/publish cycle
		// (publish replaces all rule IDs on the set). Fall back to priority.
		priority := d.Get("rule_priority").(int)
		found, err2 := findRuleByPriority(c, setID, priority)
		if err2 != nil {
			log.Printf("[WARN] Firewall rule %d (priority %d) in set %d not found, removing from state", ruleID, priority, setID)
			d.SetId("")
			return nil
		}
		rule = *found
		d.SetId(fmt.Sprintf("%d/%d", setID, rule.ID))
		log.Printf("[DEBUG] Firewall rule ID refreshed: %d -> %d (matched by priority %d)", ruleID, rule.ID, priority)
	}

	var diags diag.Diagnostics

	setValue("firewall_set_id", setID, d, &diags)
	setValue("rule_id", rule.ID, d, &diags)
	setValue("ip_version", rule.IPVersion, d, &diags)
	setValue("direction", rule.Direction, d, &diags)
	setValue("action", rule.Action, d, &diags)
	setValue("enabled", rule.Enabled, d, &diags)
	setValue("admin_comment", rule.AdminComment, d, &diags)
	setValue("rule_priority", rule.RulePriority, d, &diags)

	if mc := rule.MatchCriteria; mc != nil {
		setValue("protocol", mc.Protocol, d, &diags)
		setValue("source_net", mc.SourceNet, d, &diags)
		setValue("destination_net", mc.DestinationNet, d, &diags)
		setIntPtr("source_port_start", mc.SourcePortStart, d, &diags)
		setIntPtr("source_port_end", mc.SourcePortEnd, d, &diags)
		setIntPtr("destination_port_start", mc.DestinationPortStart, d, &diags)
		setIntPtr("destination_port_end", mc.DestinationPortEnd, d, &diags)
		if mc.Options != nil {
			setValue("icmp_type", mc.Options.ICMPType, d, &diags)
		}
	}

	return diags
}

func resourceFirewallRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID, _, err := parseFirewallSetRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	unlock := lockFirewallSet(setID)
	defer unlock()

	//Get the current rule's priority for finding it on the draft
	currentPriority := d.Get("rule_priority").(int)

	//Create draft
	draftID, err := getOrCreateDraft(c, setID)
	if err != nil {
		return diag.FromErr(err)
	}

	//Find the corresponding rule on the draft by priority
	draftRule, err := findRuleByPriority(c, draftID, currentPriority)
	if err != nil {
		return diag.FromErr(fmt.Errorf("rule not found on draft: %w", err))
	}

	//Update the rule on the draft
	req := buildFirewallRuleRequest(d)
	if _, err := c.UpdateFirewallRule(draftID, draftRule.ID, req); err != nil {
		return diag.FromErr(fmt.Errorf("updating rule: %w", err))
	}

	//Publish
	if _, err := c.PublishDraftFirewallSet(draftID); err != nil {
		return diag.FromErr(fmt.Errorf("publishing draft: %w", err))
	}

	//Find the updated rule on the original set by its new priority
	newPriority := currentPriority
	if req.RulePriority != nil {
		newPriority = *req.RulePriority
	}
	updatedRule, err := findRuleByPriority(c, setID, newPriority)
	if err != nil {
		return diag.FromErr(fmt.Errorf("rule not found after publish: %w", err))
	}

	//Update composite ID with new rule ID
	d.SetId(fmt.Sprintf("%d/%d", setID, updatedRule.ID))

	//Sync if requested
	if d.Get("sync_after_publish").(bool) {
		if err := c.SyncFirewallSetRules(setID); err != nil {
			return diag.Errorf("sync failed after update: %s", err)
		}
	}

	return resourceFirewallRuleRead(ctx, d, m)
}

func resourceFirewallRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID, _, err := parseFirewallSetRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	unlock := lockFirewallSet(setID)
	defer unlock()

	//Get the current rule's priority
	currentPriority := d.Get("rule_priority").(int)

	//Create draft
	draftID, err := getOrCreateDraft(c, setID)
	if err != nil {
		return diag.FromErr(err)
	}

	//Find the draft rule
	draftRule, err := findRuleByPriority(c, draftID, currentPriority)
	if err != nil {
		log.Printf("[WARN] Rule with priority %d not found on draft %d, may already be deleted", currentPriority, draftID)
		// Delete the draft if we didn't modify it
		_ = c.DeleteDraftFirewallSet(draftID)
		return nil
	}

	//Delete the rule from the draft
	if err := c.DeleteFirewallRule(draftID, draftRule.ID); err != nil {
		return diag.FromErr(fmt.Errorf("deleting rule: %w", err))
	}

	//Publish
	if _, err := c.PublishDraftFirewallSet(draftID); err != nil {
		return diag.FromErr(fmt.Errorf("publishing draft: %w", err))
	}

	//Sync if requested
	if d.Get("sync_after_publish").(bool) {
		if err := c.SyncFirewallSetRules(setID); err != nil {
			return diag.Errorf("sync failed after delete: %s", err)
		}
	}

	return nil
}

func resourceFirewallRuleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected import ID format: firewall_set_id/rule_id, got: %s", d.Id())
	}

	setID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid firewall_set_id %q: %w", parts[0], err)
	}

	ruleID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid rule_id %q: %w", parts[1], err)
	}

	d.SetId(fmt.Sprintf("%d/%d", setID, ruleID))
	if err := d.Set("firewall_set_id", setID); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func parseFirewallSetRuleID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid firewall rule ID %q, expected \"setId/ruleId\"", id)
	}
	setID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid firewall_set_id %q: %w", parts[0], err)
	}
	ruleID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid rule_id %q: %w", parts[1], err)
	}
	return setID, ruleID, nil
}
