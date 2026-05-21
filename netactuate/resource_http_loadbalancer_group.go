package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceHTTPLoadbalancerGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHTTPLBGroupCreate,
		ReadContext:   resourceHTTPLBGroupRead,
		UpdateContext: resourceHTTPLBGroupUpdate,
		DeleteContext: resourceHTTPLBGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceHTTPLBGroupImport,
		},
		Schema: map[string]*schema.Schema{
			"http_loadbalancer_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the HTTP load balancer",
			},
			"http_group_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The group ID",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the HTTP LB group",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the HTTP LB group",
			},
			"algorithm": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Load balancing algorithm (round-robin or least-connections)",
			},
			"sticky_sessions_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to enable sticky sessions",
			},
			"ssl_to_backend_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to use SSL when connecting to backends",
			},
			"internal_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The port to connect to on the backends",
			},
			"match_address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Public IP address to match incoming traffic",
			},
			"match_ports": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Ports to match (80, 443, or 80+443)",
			},
			"is_online": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the group is online",
			},
			"health_check_active_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether active health checks are enabled",
			},
			"health_check_active_interval": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Active health check interval in seconds",
			},
			"health_check_active_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Active health check retries",
			},
			"health_check_active_delay": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Active health check delay in seconds",
			},
			"health_check_active_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Active health check timeout in seconds",
			},
			"health_check_active_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Active health check HTTP path",
			},
			"health_check_passive_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether passive health checks are enabled",
			},
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Domain/path routing rules",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"http_rule_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The rule ID assigned by the API",
						},
						"match_domain": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Domain to match",
						},
						"match_path": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Path to match",
						},
						"ssl_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether SSL is enabled for this rule",
						},
						"ssl_certificate_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "SSL certificate ID (omit for auto SSL)",
						},
						"https_redirect_enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							// The API forces httpsRedirectEnabled=true whenever ssl_enabled=true,
							// regardless of the requested value. Suppress the diff so that
							// setting https_redirect_enabled=false in config with ssl_enabled=true
							// doesn't produce a perpetual no-op plan change.
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								prefix := k[:strings.LastIndex(k, ".")]
								ssl, _ := d.GetOk(prefix + ".ssl_enabled")
								return ssl.(bool)
							},
							Description: "Whether to redirect HTTP to HTTPS",
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
						"http_backend_id": {
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

func resourceHTTPLBGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	httpLbID := d.Get("http_loadbalancer_id").(int)

	req := &gona.CreateHTTPLBGroupRequest{
		Name:                  d.Get("name").(string),
		Description:           d.Get("description").(string),
		Algorithm:             d.Get("algorithm").(string),
		StickySessionsEnabled: d.Get("sticky_sessions_enabled").(bool),
		SSLToBackendEnabled:   d.Get("ssl_to_backend_enabled").(bool),
		InternalPort:          d.Get("internal_port").(int),
		Match: gona.HTTPLBGroupMatch{
			Address: d.Get("match_address").(string),
			Ports:   d.Get("match_ports").(string),
		},
		HealthCheck: expandHTTPLBHealthCheck(d),
		Rules:       expandHTTPLBRules(d.Get("rule").([]interface{})),
		Backends:    expandHTTPLBBackends(d.Get("backend").([]interface{})),
	}

	group, err := c.CreateHTTPLBGroup(httpLbID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", httpLbID, group.HTTPGroupID))
	log.Printf("[DEBUG] HTTP LB group %d created in LB %d", group.HTTPGroupID, httpLbID)

	return resourceHTTPLBGroupRead(ctx, d, m)
}

func resourceHTTPLBGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	httpLbID, groupID, err := parseHTTPLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	group, err := c.GetHTTPLBGroup(httpLbID, groupID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] HTTP LB group %d in LB %d not found, removing from state", groupID, httpLbID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("http_loadbalancer_id", httpLbID, d, &diags)
	setValue("http_group_id", group.HTTPGroupID, d, &diags)
	setValue("name", group.Name, d, &diags)
	setValue("description", group.Description, d, &diags)
	setValue("algorithm", group.Algorithm, d, &diags)
	setValue("sticky_sessions_enabled", group.StickySessionsEnabled, d, &diags)
	setValue("ssl_to_backend_enabled", group.SSLToBackendEnabled, d, &diags)
	setValue("internal_port", group.InternalPort, d, &diags)
	setValue("match_address", group.Match.Address, d, &diags)
	setValue("match_ports", group.Match.Ports, d, &diags)
	setValue("is_online", group.IsOnline, d, &diags)

	// Health check
	setValue("health_check_active_enabled", group.HealthCheck.Active.Enabled, d, &diags)
	setValue("health_check_active_interval", group.HealthCheck.Active.Interval, d, &diags)
	setValue("health_check_active_retries", group.HealthCheck.Active.Retries, d, &diags)
	setValue("health_check_active_delay", group.HealthCheck.Active.Delay, d, &diags)
	if group.HealthCheck.Active.Timeout != nil {
		setValue("health_check_active_timeout", *group.HealthCheck.Active.Timeout, d, &diags)
	}
	setValue("health_check_active_path", group.HealthCheck.Active.Path, d, &diags)
	setValue("health_check_passive_enabled", group.HealthCheck.Passive.Enabled, d, &diags)

	setValue("rule", flattenHTTPLBRules(group.Rules), d, &diags)
	setValue("backend", flattenHTTPLBBackends(group.Backends), d, &diags)

	return diags
}

func resourceHTTPLBGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	httpLbID, groupID, err := parseHTTPLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.ReplaceHTTPLBGroupRequest{
		Name:                  d.Get("name").(string),
		Description:           d.Get("description").(string),
		Algorithm:             d.Get("algorithm").(string),
		StickySessionsEnabled: d.Get("sticky_sessions_enabled").(bool),
		SSLToBackendEnabled:   d.Get("ssl_to_backend_enabled").(bool),
		InternalPort:          d.Get("internal_port").(int),
		Match: gona.HTTPLBGroupMatch{
			Address: d.Get("match_address").(string),
			Ports:   d.Get("match_ports").(string),
		},
		HealthCheck: expandHTTPLBHealthCheck(d),
		Rules:       expandHTTPLBRules(d.Get("rule").([]interface{})),
		Backends:    expandHTTPLBBackends(d.Get("backend").([]interface{})),
	}

	if _, err := c.ReplaceHTTPLBGroup(httpLbID, groupID, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceHTTPLBGroupRead(ctx, d, m)
}

func resourceHTTPLBGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	httpLbID, groupID, err := parseHTTPLBGroupID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting HTTP LB group %d from LB %d", groupID, httpLbID)

	if err := c.DeleteHTTPLBGroup(httpLbID, groupID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] HTTP LB group %d in LB %d already deleted", groupID, httpLbID)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceHTTPLBGroupImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	httpLbID, groupID, err := parseHTTPLBGroupID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"httpLbId/groupId\": %w", d.Id(), err)
	}

	d.SetId(fmt.Sprintf("%d/%d", httpLbID, groupID))
	d.Set("http_loadbalancer_id", httpLbID)

	return []*schema.ResourceData{d}, nil
}

// --- Expand helpers (TF → SDK) ---

func expandHTTPLBHealthCheck(d *schema.ResourceData) gona.HTTPLBGroupHealthCheck {
	hc := gona.HTTPLBGroupHealthCheck{
		Active: gona.HTTPLBGroupHealthCheckActive{
			Enabled:  d.Get("health_check_active_enabled").(bool),
			Interval: d.Get("health_check_active_interval").(int),
			Retries:  d.Get("health_check_active_retries").(int),
			Delay:    d.Get("health_check_active_delay").(int),
			Path:     d.Get("health_check_active_path").(string),
		},
		Passive: gona.HTTPLBGroupHealthCheckPassive{
			Enabled: d.Get("health_check_passive_enabled").(bool),
		},
	}

	if v, ok := d.GetOk("health_check_active_timeout"); ok {
		timeout := v.(int)
		hc.Active.Timeout = &timeout
	}

	return hc
}

func expandHTTPLBRules(raw []interface{}) []gona.HTTPLBGroupRule {
	rules := make([]gona.HTTPLBGroupRule, len(raw))
	for i, r := range raw {
		rule := r.(map[string]interface{})
		gr := gona.HTTPLBGroupRule{
			HTTPSRedirectEnabled: rule["https_redirect_enabled"].(bool),
			Match: gona.HTTPLBGroupRuleMatch{
				Domain: rule["match_domain"].(string),
				Path:   rule["match_path"].(string),
			},
			SSL: gona.HTTPLBGroupRuleSSL{
				Enabled: rule["ssl_enabled"].(bool),
			},
		}
		if v, ok := rule["ssl_certificate_id"]; ok && v.(int) > 0 {
			certID := v.(int)
			gr.SSL.SSLCertificateID = &certID
		}
		rules[i] = gr
	}
	return rules
}

func expandHTTPLBBackends(raw []interface{}) []gona.HTTPLBGroupBackend {
	backends := make([]gona.HTTPLBGroupBackend, len(raw))
	for i, b := range raw {
		backend := b.(map[string]interface{})
		backends[i] = gona.HTTPLBGroupBackend{
			Name:            backend["name"].(string),
			InternalAddress: backend["internal_address"].(string),
		}
	}
	return backends
}

// --- Flatten helpers (SDK → TF) ---

func flattenHTTPLBRules(rules []gona.HTTPLBGroupRule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(rules))
	for i, r := range rules {
		certID := 0
		if r.SSL.SSLCertificateID != nil {
			certID = *r.SSL.SSLCertificateID
		}
		result[i] = map[string]interface{}{
			"http_rule_id":           r.HTTPRuleID,
			"match_domain":           r.Match.Domain,
			"match_path":             r.Match.Path,
			"ssl_enabled":            r.SSL.Enabled,
			"ssl_certificate_id":     certID,
			"https_redirect_enabled": r.HTTPSRedirectEnabled,
		}
	}
	return result
}

func flattenHTTPLBBackends(backends []gona.HTTPLBGroupBackend) []map[string]interface{} {
	result := make([]map[string]interface{}, len(backends))
	for i, b := range backends {
		result[i] = map[string]interface{}{
			"http_backend_id":  b.HTTPBackendID,
			"name":             b.Name,
			"internal_address": b.InternalAddress,
			"is_online":        b.IsOnline,
		}
	}
	return result
}

// --- ID parsing ---

func parseHTTPLBGroupID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid HTTP LB group ID %q, expected \"httpLbId/groupId\"", id)
	}
	httpLbID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid http_loadbalancer_id %q: %w", parts[0], err)
	}
	groupID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid http_group_id %q: %w", parts[1], err)
	}
	return httpLbID, groupID, nil
}
