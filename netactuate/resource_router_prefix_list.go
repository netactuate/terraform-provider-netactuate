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

func resourceRouterPrefixList() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterPrefixListCreate,
		ReadContext:   resourceRouterPrefixListRead,
		UpdateContext: resourceRouterPrefixListUpdate,
		DeleteContext: resourceRouterPrefixListDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterPrefixListImport,
		},
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"prefix_list_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the prefix list.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name your prefix list something you'll use to refer to it in other configurations.",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "Whether the prefix list refers to IPv4 or IPv6 networks (4 or 6).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the prefix list.",
			},
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The list of rules for the prefix list, matched in order of priority (first elements have the highest priority).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"permit", "deny"}, false)),
							Description:      "Whether to permit or deny matching prefixes.",
						},
						"prefix": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The network prefix in CIDR notation (e.g. 10.0.0.0/8).",
						},
					},
				},
			},
		},
	}
}

func resourceRouterPrefixListCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	req := &gona.CreateRouterPrefixListRequest{
		Name:      d.Get("name").(string),
		IPVersion: d.Get("ip_version").(int),
		Rules:     expandPrefixListRules(d.Get("rule").([]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	resp, err := c.CreateRouterPrefixList(routerID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, resp.PrefixListID))

	var diags diag.Diagnostics
	setValue("prefix_list_id", resp.PrefixListID, d, &diags)

	readDiags := resourceRouterPrefixListRead(ctx, d, m)
	diags = append(diags, readDiags...)
	return diags
}

func resourceRouterPrefixListRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, prefixListID, err := parsePrefixListID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	prefixList, err := c.GetRouterPrefixList(routerID, prefixListID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("[WARN] Prefix list %d not found on router %d, removing from state", prefixListID, routerID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("prefix_list_id", prefixList.PrefixListID, d, &diags)
	setValue("name", prefixList.Name, d, &diags)
	setValue("ip_version", prefixList.IPVersion, d, &diags)
	setValue("description", prefixList.Description, d, &diags)
	setValue("rule", flattenPrefixListRules(prefixList.Rules), d, &diags)

	return diags
}

func resourceRouterPrefixListUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, prefixListID, err := parsePrefixListID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(routerID)
	defer unlock()

	req := &gona.UpdateRouterPrefixListRequest{
		Name:      d.Get("name").(string),
		IPVersion: d.Get("ip_version").(int),
		Rules:     expandPrefixListRules(d.Get("rule").([]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}

	_, err = c.UpdateRouterPrefixList(routerID, prefixListID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterPrefixListRead(ctx, d, m)
}

func resourceRouterPrefixListDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, prefixListID, err := parsePrefixListID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(routerID)
	defer unlock()

	err = c.DeleteRouterPrefixList(routerID, prefixListID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceRouterPrefixListImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, prefixListID, err := parsePrefixListID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"routerId/prefixListId\" (e.g. \"42/10\")", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, prefixListID))
	d.Set("router_id", routerID)

	return []*schema.ResourceData{d}, nil
}

func parsePrefixListID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid prefix list ID %q, expected \"routerId/prefixListId\"", id)
	}
	routerID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid router_id %q: %w", parts[0], err)
	}
	prefixListID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid prefix_list_id %q: %w", parts[1], err)
	}
	return routerID, prefixListID, nil
}

func expandPrefixListRules(raw []interface{}) []gona.PrefixListRule {
	rules := make([]gona.PrefixListRule, len(raw))
	for i, r := range raw {
		ruleMap := r.(map[string]interface{})
		rules[i] = gona.PrefixListRule{
			Action: ruleMap["action"].(string),
			Prefix: ruleMap["prefix"].(string),
		}
	}
	return rules
}

func flattenPrefixListRules(rules []gona.PrefixListRule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		result[i] = map[string]interface{}{
			"action": rule.Action,
			"prefix": rule.Prefix,
		}
	}
	return result
}
