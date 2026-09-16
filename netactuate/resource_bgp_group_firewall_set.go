package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceBGPGroupFirewallSet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBGPGroupFirewallSetCreate,
		ReadContext:   resourceBGPGroupFirewallSetRead,
		DeleteContext: resourceBGPGroupFirewallSetDelete,
		Description:   "Binds a firewall set to a BGP group.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"bgp_group_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "BGP group ID to bind the firewall set to.",
			},
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Firewall set ID to bind to the BGP group.",
			},
			"interface_number": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     0,
				Description: "BGP group interface number for the firewall binding.",
			},
			"set_priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     0,
				Description: "Priority of this firewall set binding.",
			},
			"binding_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "API assigned firewall set binding ID.",
			},
		},
	}
}

func resourceBGPGroupFirewallSetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	groupID := d.Get("bgp_group_id").(int)
	firewallSetID := d.Get("firewall_set_id").(int)
	req := &gona.BindBGPGroupFirewallSetRequest{
		FirewallSetID:   firewallSetID,
		InterfaceNumber: d.Get("interface_number").(int),
		SetPriority:     d.Get("set_priority").(int),
	}

	binding, err := c.BindBGPGroupFirewallSet(groupID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(formatBGPGroupFirewallSetID(groupID, firewallSetID))
	var diags diag.Diagnostics
	setValue("binding_id", binding.ID, d, &diags)
	setValue("bgp_group_id", binding.BGPGroupID, d, &diags)
	setValue("firewall_set_id", binding.FirewallSetID, d, &diags)
	setValue("interface_number", binding.InterfaceNumber, d, &diags)
	setValue("set_priority", binding.SetPriority, d, &diags)
	return diags
}

func resourceBGPGroupFirewallSetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	groupID, firewallSetID, err := parseBGPGroupFirewallSetID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("bgp_group_id", groupID, d, &diags)
	setValue("firewall_set_id", firewallSetID, d, &diags)
	return diags
}

func resourceBGPGroupFirewallSetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	groupID, firewallSetID, err := parseBGPGroupFirewallSetID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.UnbindBGPGroupFirewallSet(groupID, firewallSetID); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func formatBGPGroupFirewallSetID(groupID, firewallSetID int) string {
	return fmt.Sprintf("%d/%d", groupID, firewallSetID)
}

func parseBGPGroupFirewallSetID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid BGP group firewall set ID %q, expected {bgp_group_id}/{firewall_set_id}", id)
	}
	groupID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid BGP group ID %q: %s", parts[0], err)
	}
	firewallSetID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid firewall set ID %q: %s", parts[1], err)
	}
	return groupID, firewallSetID, nil
}
