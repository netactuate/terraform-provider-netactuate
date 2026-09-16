package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceFirewallSets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallSetsRead,
		Schema: map[string]*schema.Schema{
			"firewall_sets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of firewall sets visible to the account",
				Elem: &schema.Resource{
					Schema: firewallSetDataSourceSchema(),
				},
			},
		},
	}
}

func firewallSetDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Firewall set ID",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Firewall set name",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Firewall set description",
		},
		"enabled": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the firewall set is enabled",
		},
		"is_draft": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the firewall set is a draft",
		},
		"draft_firewall_set_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Draft firewall set ID, or 0 when absent",
		},
		"created": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Firewall set creation timestamp",
		},
		"last_updated": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Firewall set last update timestamp",
		},
	}
}

func dataSourceFirewallSetsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	sets, err := c.GetFirewallSets()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setsList := make([]map[string]interface{}, len(sets))
	for i, set := range sets {
		setsList[i] = flattenFirewallSet(set)
	}

	setValue("firewall_sets", setsList, d, &diags)
	d.SetId("firewall-sets")

	return diags
}

func flattenFirewallSet(set gona.FirewallSet) map[string]interface{} {
	draftFirewallSetID := 0
	if set.DraftFirewallSetID != nil {
		draftFirewallSetID = *set.DraftFirewallSetID
	}

	return map[string]interface{}{
		"id":                    set.ID,
		"name":                  set.Name,
		"description":           set.Description,
		"enabled":               set.Enabled,
		"is_draft":              set.IsDraft,
		"draft_firewall_set_id": draftFirewallSetID,
		"created":               set.Created,
		"last_updated":          set.LastUpdated,
	}
}
