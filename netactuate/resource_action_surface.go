package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceRouterRoutingView() *schema.Resource {
	return &schema.Resource{
		Description:   "Beta. The cloud router family is in beta: behaviour and schema may change. Requests routing views from a VRF.",
		CreateContext: resourceRouterRoutingViewReadPost,
		ReadContext:   resourceRouterRoutingViewReadPost,
		UpdateContext: resourceRouterRoutingViewReadPost,
		DeleteContext: resourceRouterRoutingViewDelete,
		// No Importer: routing_view is a query action whose required "view" request
		// list has no server-side identity to import from, so a passthrough import
		// would leave Read unable to reconstruct the request. Import is unsupported.
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Cloud router ID."},
			"vrf_id":    {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "VRF ID."},
			"view": {Type: schema.TypeList, Required: true, Description: "Routing views to request from the router.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"view_id":    {Type: schema.TypeString, Optional: true, Description: "Optional caller supplied view ID."},
				"ip_version": {Type: schema.TypeInt, Required: true, Description: "IP version for the routing view."},
				"name":       {Type: schema.TypeString, Required: true, Description: "Routing view name."},
				"filter":     {Type: schema.TypeString, Optional: true, Description: "Optional routing view filter."},
			}}},
			"result_json": rawJSONDataSourceSchema("Requested routing views as compact JSON."),
		},
	}
}

func resourceRouterRoutingViewReadPost(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	req := &gona.RouterRoutingViewRequest{}
	for _, raw := range d.Get("view").([]interface{}) {
		view := raw.(map[string]interface{})
		req.Views = append(req.Views, gona.RouterRoutingViewSelector{ID: view["view_id"].(string), IPVersion: view["ip_version"].(int), Name: view["name"].(string), Filter: view["filter"].(string)})
	}
	result, err := m.(*ProviderClients).V3.GetRouterRoutingViews(routerID, vrfID, req)
	if err != nil {
		// On refresh (the resource already has an id) a 404 means the router or VRF
		// is gone, so the routing view is gone too: clear it from state. On create or
		// update (no id yet) a 404 is a real failure and must surface.
		if d.Id() != "" && gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	out, err := compactSurfaceJSON(result)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("result_json", out, d, &diags)
	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))
	return diags
}

func resourceRouterRoutingViewDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "Routing view remains query-only", Detail: "The routing view endpoint returns live router state and has no delete operation. Terraform removed routing view " + id + " from state."}}
}

func resourceFirewallSetVMDetach() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallSetVMDetachCreate,
		ReadContext:   resourceFirewallSetVMDetachRead,
		DeleteContext: resourceFirewallSetVMDetachDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"relation_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Firewall set VM relation ID to detach."},
		},
	}
}

func resourceFirewallSetVMDetachCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	relationID := d.Get("relation_id").(int)
	if err := m.(*ProviderClients).V2.DetachFirewallSetVMRelation(relationID); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(relationID))
	return nil
}

func resourceFirewallSetVMDetachRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	relationID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("relation_id", relationID, d, &diags)
	return diags
}

func resourceFirewallSetVMDetachDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "Firewall set VM relation remains detached", Detail: "The relation detach endpoint has no inverse operation. Terraform removed detach marker " + id + " from state."}}
}

func resourceFirewallSetRuleOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallSetRuleOrderApply,
		ReadContext:   resourceFirewallSetRuleOrderRead,
		UpdateContext: resourceFirewallSetRuleOrderApply,
		DeleteContext: resourceFirewallSetRuleOrderDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Firewall set ID whose rule order is managed."},
			"move_id":         {Type: schema.TypeInt, Required: true, Description: "Firewall rule ID to move."},
			"after_id":        {Type: schema.TypeInt, Optional: true, Description: "Move the rule after this firewall rule ID."},
			"before_id":       {Type: schema.TypeInt, Optional: true, Description: "Move the rule before this firewall rule ID."},
		},
	}
}

func resourceFirewallSetRuleOrderApply(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	setID := d.Get("firewall_set_id").(int)
	moveID := d.Get("move_id").(int)
	req := &gona.ReorderFirewallRulesRequest{MoveID: moveID}
	if v, ok := d.GetOk("after_id"); ok {
		afterID := v.(int)
		req.AfterID = &afterID
	}
	if v, ok := d.GetOk("before_id"); ok {
		beforeID := v.(int)
		req.BeforeID = &beforeID
	}
	if err := m.(*ProviderClients).V2.ReorderFirewallRules(setID, req); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", setID, moveID))
	return resourceFirewallSetRuleOrderRead(ctx, d, m)
}

func resourceFirewallSetRuleOrderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	setID, moveID, err := parseSurfaceTwoPartID(d.Id(), "firewallSetId/moveId")
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("firewall_set_id", setID, d, &diags)
	setValue("move_id", moveID, d, &diags)
	return diags
}

func resourceFirewallSetRuleOrderDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "Firewall rule order is unchanged", Detail: "The firewall reorder endpoint has no delete operation. Terraform removed rule order marker " + id + " from state without changing the firewall set."}}
}

func resourceFirewallSetDetachAllVMs() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallSetDetachAllVMsCreate,
		ReadContext:   resourceFirewallSetDetachAllVMsRead,
		DeleteContext: resourceFirewallSetDetachAllVMsDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Firewall set ID whose VM attachments are detached."},
		},
	}
}

func resourceFirewallSetDetachAllVMsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	setID := d.Get("firewall_set_id").(int)
	if err := m.(*ProviderClients).V2.DetachAllFirewallSetVMs(setID); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(setID))
	return nil
}

func resourceFirewallSetDetachAllVMsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	setID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("firewall_set_id", setID, d, &diags)
	return diags
}

func resourceFirewallSetDetachAllVMsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "Firewall set VMs remain detached", Detail: "The detach-all endpoint has no inverse operation. Terraform removed detach-all marker " + id + " from state."}}
}
