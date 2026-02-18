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

func resourceNetworkLoadbalancerGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNLBGroupCreate,
		ReadContext:   resourceNLBGroupRead,
		UpdateContext: resourceNLBGroupUpdate,
		DeleteContext: resourceNLBGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNLBGroupImport,
		},
		Schema: map[string]*schema.Schema{
			"network_loadbalancer_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the network load balancer",
			},
			"network_group_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The group ID assigned by the API",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the load balancer group",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the load balancer group",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "IP version (4 or 6)",
			},
			"algorithm": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Load balancing algorithm (e.g. round-robin)",
			},
			"match_address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Public IP address to match incoming traffic",
			},
			"is_online": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the group is online",
			},
			"health_check": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Health check configuration",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Whether health checks are enabled",
						},
						"method": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Health check method (e.g. Ping)",
						},
						"interval": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Health check interval in seconds",
						},
						"retries": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Number of retries before marking unhealthy",
						},
						"delay": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Delay between retries in seconds",
						},
						"timeout": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Health check timeout in seconds",
						},
					},
				},
			},
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Forwarding rules",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The rule ID assigned by the API",
						},
						"protocol": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Protocol (e.g. TCP, UDP)",
						},
						"port_match": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "External port to match",
						},
						"port_internal": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Internal port to forward to",
						},
					},
				},
			},
			"backend": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Backend servers",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_backend_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The backend ID assigned by the API",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the backend",
						},
						"internal_address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Internal IP address of the backend",
						},
						"is_online": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this backend is online",
						},
					},
				},
			},
		},
	}
}

func resourceNLBGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	nlbID := d.Get("network_loadbalancer_id").(int)

	req := &gona.CreateNLBGroupRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IPVersion:   d.Get("ip_version").(int),
		Algorithm:   d.Get("algorithm").(string),
		Match:       gona.NLBGroupMatch{Address: d.Get("match_address").(string)},
		HealthCheck: expandNLBHealthCheck(d.Get("health_check").([]interface{})),
		Rules:       expandNLBRules(d.Get("rule").([]interface{})),
		Backends:    expandNLBBackends(d.Get("backend").([]interface{})),
	}

	group, err := c.CreateNLBGroup(nlbID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", nlbID, group.NetworkGroupID))
	log.Printf("[DEBUG] NLB group %d created in NLB %d", group.NetworkGroupID, nlbID)

	return resourceNLBGroupRead(ctx, d, m)
}

func resourceNLBGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	nlbID, groupID, err := parseNLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	group, err := c.GetNLBGroup(nlbID, groupID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NLB group %d in NLB %d not found, removing from state", groupID, nlbID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("network_loadbalancer_id", nlbID, d, &diags)
	setValue("network_group_id", group.NetworkGroupID, d, &diags)
	setValue("name", group.Name, d, &diags)
	setValue("description", group.Description, d, &diags)
	setValue("ip_version", group.IPVersion, d, &diags)
	setValue("algorithm", group.Algorithm, d, &diags)
	setValue("match_address", group.Match.Address, d, &diags)
	setValue("is_online", group.IsOnline, d, &diags)

	setValue("health_check", flattenNLBHealthCheck(group.HealthCheck), d, &diags)
	setValue("rule", flattenNLBRules(group.Rules), d, &diags)
	setValue("backend", flattenNLBBackends(group.Backends), d, &diags)

	return diags
}

func resourceNLBGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	nlbID, groupID, err := parseNLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.ReplaceNLBGroupRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IPVersion:   d.Get("ip_version").(int),
		Algorithm:   d.Get("algorithm").(string),
		Match:       gona.NLBGroupMatch{Address: d.Get("match_address").(string)},
		HealthCheck: expandNLBHealthCheck(d.Get("health_check").([]interface{})),
		Rules:       expandNLBRules(d.Get("rule").([]interface{})),
		Backends:    expandNLBBackends(d.Get("backend").([]interface{})),
	}

	if _, err := c.ReplaceNLBGroup(nlbID, groupID, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceNLBGroupRead(ctx, d, m)
}

func resourceNLBGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	nlbID, groupID, err := parseNLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting NLB group %d from NLB %d", groupID, nlbID)

	if err := c.DeleteNLBGroup(nlbID, groupID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NLB group %d in NLB %d already deleted", groupID, nlbID)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceNLBGroupImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	nlbID, groupID, err := parseNLBGroupID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"nlbId/groupId\": %w", d.Id(), err)
	}

	d.SetId(fmt.Sprintf("%d/%d", nlbID, groupID))
	d.Set("network_loadbalancer_id", nlbID)

	return []*schema.ResourceData{d}, nil
}

// --- Expand helpers (TF → SDK) ---

func expandNLBHealthCheck(raw []interface{}) gona.NLBGroupHealthCheck {
	if len(raw) == 0 {
		return gona.NLBGroupHealthCheck{}
	}
	hc := raw[0].(map[string]interface{})
	return gona.NLBGroupHealthCheck{
		Enabled:  hc["enabled"].(bool),
		Method:   hc["method"].(string),
		Interval: hc["interval"].(int),
		Retries:  hc["retries"].(int),
		Delay:    hc["delay"].(int),
		Timeout:  hc["timeout"].(int),
	}
}

func expandNLBRules(raw []interface{}) []gona.NLBGroupRule {
	rules := make([]gona.NLBGroupRule, len(raw))
	for i, r := range raw {
		rule := r.(map[string]interface{})
		rules[i] = gona.NLBGroupRule{
			Protocol: rule["protocol"].(string),
			Ports: gona.NLBGroupRulePorts{
				Match:    rule["port_match"].(int),
				Internal: rule["port_internal"].(int),
			},
		}
	}
	return rules
}

func expandNLBBackends(raw []interface{}) []gona.NLBGroupBackend {
	backends := make([]gona.NLBGroupBackend, len(raw))
	for i, b := range raw {
		backend := b.(map[string]interface{})
		backends[i] = gona.NLBGroupBackend{
			Name:            backend["name"].(string),
			InternalAddress: backend["internal_address"].(string),
		}
	}
	return backends
}

// --- Flatten helpers (SDK → TF) ---

func flattenNLBHealthCheck(hc gona.NLBGroupHealthCheck) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"enabled":  hc.Enabled,
			"method":   hc.Method,
			"interval": hc.Interval,
			"retries":  hc.Retries,
			"delay":    hc.Delay,
			"timeout":  hc.Timeout,
		},
	}
}

func flattenNLBRules(rules []gona.NLBGroupRule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(rules))
	for i, r := range rules {
		result[i] = map[string]interface{}{
			"network_rule_id": r.NetworkRuleID,
			"protocol":        r.Protocol,
			"port_match":      r.Ports.Match,
			"port_internal":   r.Ports.Internal,
		}
	}
	return result
}

func flattenNLBBackends(backends []gona.NLBGroupBackend) []map[string]interface{} {
	result := make([]map[string]interface{}, len(backends))
	for i, b := range backends {
		result[i] = map[string]interface{}{
			"network_backend_id": b.NetworkBackendID,
			"name":               b.Name,
			"internal_address":   b.InternalAddress,
			"is_online":          b.IsOnline,
		}
	}
	return result
}

// --- ID parsing ---

func parseNLBGroupID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid NLB group ID %q, expected \"nlbId/groupId\"", id)
	}
	nlbID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid network_loadbalancer_id %q: %w", parts[0], err)
	}
	groupID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid network_group_id %q: %w", parts[1], err)
	}
	return nlbID, groupID, nil
}
