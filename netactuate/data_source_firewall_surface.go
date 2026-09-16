package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func firewallSetVMSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"relation_id":     {Type: schema.TypeInt, Computed: true, Description: "Firewall set VM relation ID."},
		"mbpkgid":         {Type: schema.TypeInt, Computed: true, Description: "VM package ID."},
		"interface_id":    {Type: schema.TypeInt, Computed: true, Description: "Network interface ID."},
		"firewall_set_id": {Type: schema.TypeInt, Computed: true, Description: "Firewall set ID."},
		"set_priority":    {Type: schema.TypeInt, Computed: true, Description: "Firewall set priority for the VM."},
		"hostname":        {Type: schema.TypeString, Computed: true, Description: "VM hostname."},
		"location":        {Type: schema.TypeString, Computed: true, Description: "VM location name."},
		"iata_code":       {Type: schema.TypeString, Computed: true, Description: "VM location IATA code."},
	}
}

func flattenFirewallSetVM(vm gona.FirewallSetVM) map[string]interface{} {
	return map[string]interface{}{
		"relation_id":     vm.ID,
		"mbpkgid":         vm.Mbpkgid,
		"interface_id":    vm.InterfaceID,
		"firewall_set_id": vm.FirewallSetID,
		"set_priority":    vm.SetPriority,
		"hostname":        vm.Hostname,
		"location":        vm.Location,
		"iata_code":       vm.IATACode,
	}
}

func dataSourceFirewallExternalIPSets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallExternalIPSetsRead,
		Schema: map[string]*schema.Schema{
			"external_ipset_id": {Type: schema.TypeInt, Optional: true, Description: "External IP set ID. When omitted, all external IP sets are returned."},
			"ipsets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "External IP sets visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"external_ipset_id": {Type: schema.TypeInt, Computed: true, Description: "External IP set ID."},
					"name":              {Type: schema.TypeString, Computed: true, Description: "External IP set name."},
					"description":       {Type: schema.TypeString, Computed: true, Description: "External IP set description."},
					"raw_json":          rawJSONDataSourceSchema("Full external IP set response as compact JSON."),
				}},
			},
		},
	}
}

func dataSourceFirewallExternalIPSetsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	var sets []gona.FirewallExternalIPSet
	if v, ok := d.GetOk("external_ipset_id"); ok {
		set, err := c.GetFirewallExternalIPSet(v.(int))
		if err != nil {
			return diag.FromErr(err)
		}
		sets = []gona.FirewallExternalIPSet{set}
	} else {
		var err error
		sets, err = c.GetFirewallExternalIPSets()
		if err != nil {
			return diag.FromErr(err)
		}
	}
	out := make([]map[string]interface{}, len(sets))
	for i, set := range sets {
		raw, err := compactSurfaceJSON(set.Raw)
		if err != nil {
			return diag.FromErr(err)
		}
		out[i] = map[string]interface{}{"external_ipset_id": set.ID, "name": set.Name, "description": set.Description, "raw_json": raw}
	}
	var diags diag.Diagnostics
	setValue("ipsets", out, d, &diags)
	d.SetId("firewall-external-ipsets")
	if v, ok := d.GetOk("external_ipset_id"); ok {
		d.SetId(fmt.Sprintf("firewall-external-ipset-%d", v.(int)))
	}
	return diags
}

func dataSourceFirewallManageEnabled() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallManageEnabledRead,
		Schema: map[string]*schema.Schema{
			"enabled":  {Type: schema.TypeBool, Computed: true, Description: "Whether firewall management is available for the account."},
			"raw_json": rawJSONDataSourceSchema("Full firewall management availability response as compact JSON."),
		},
	}
}

func dataSourceFirewallManageEnabledRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	enabled, err := m.(*ProviderClients).V2.GetFirewallManageEnabled()
	if err != nil {
		return diag.FromErr(err)
	}
	raw, err := compactSurfaceJSON(enabled.Raw)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("enabled", enabled.Enabled, d, &diags)
	setValue("raw_json", raw, d, &diags)
	d.SetId("firewall-manage-enabled")
	return diags
}

func dataSourceFirewallSetRelatedVMs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallSetRelatedVMsRead,
		Schema: map[string]*schema.Schema{
			"mbpkgid":                     {Type: schema.TypeInt, Required: true, Description: "VM package ID whose firewall set relations will be listed."},
			"disable_interface_id_filter": {Type: schema.TypeBool, Optional: true, Description: "Disable interface ID filtering when querying related firewall sets."},
			"vms":                         {Type: schema.TypeList, Computed: true, Description: "Firewall set relations for the VM.", Elem: &schema.Resource{Schema: firewallSetVMSchema()}},
		},
	}
}

func dataSourceFirewallSetRelatedVMsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := &gona.FirewallRelatedSetOptions{}
	if v, ok := d.GetOkExists("disable_interface_id_filter"); ok {
		value := v.(bool)
		opts.DisableInterfaceIDFilter = &value
	}
	mbpkgid := d.Get("mbpkgid").(int)
	vms, err := m.(*ProviderClients).V2.GetFirewallSetRelatedVMs(mbpkgid, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		out[i] = flattenFirewallSetVM(vm)
	}
	var diags diag.Diagnostics
	setValue("vms", out, d, &diags)
	d.SetId(fmt.Sprintf("firewall-related-vms-%d", mbpkgid))
	return diags
}

func dataSourceFirewallSetAvailableVMs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallSetAvailableVMsRead,
		Schema: map[string]*schema.Schema{
			"firewall_set_id":             {Type: schema.TypeInt, Required: true, Description: "Firewall set ID whose available VMs will be listed."},
			"extref_account_id":           {Type: schema.TypeInt, Optional: true, Description: "External account ID filter."},
			"vpc_id":                      {Type: schema.TypeInt, Optional: true, Description: "VPC ID filter."},
			"include_bandwidth":           {Type: schema.TypeBool, Optional: true, Description: "Include bandwidth details when supported by the API."},
			"include_ul":                  {Type: schema.TypeBool, Optional: true, Description: "Include upload details when supported by the API."},
			"check_vpc":                   {Type: schema.TypeBool, Optional: true, Description: "Check VPC eligibility when supported by the API."},
			"disable_interface_id_filter": {Type: schema.TypeBool, Optional: true, Description: "Disable interface ID filtering."},
			"vms":                         {Type: schema.TypeList, Computed: true, Description: "VMs available for firewall set attachment.", Elem: &schema.Resource{Schema: firewallSetVMSchema()}},
		},
	}
}

func dataSourceFirewallSetAvailableVMsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := &gona.FirewallAvailableVMOptions{}
	if v, ok := d.GetOk("extref_account_id"); ok {
		value := v.(int)
		opts.ExtrefAccountID = &value
	}
	if v, ok := d.GetOk("vpc_id"); ok {
		value := v.(int)
		opts.VPCID = &value
	}
	for name, dest := range map[string]**bool{
		"include_bandwidth":           &opts.IncludeBandwidth,
		"include_ul":                  &opts.IncludeUL,
		"check_vpc":                   &opts.CheckVPC,
		"disable_interface_id_filter": &opts.DisableInterfaceIDFilter,
	} {
		if v, ok := d.GetOkExists(name); ok {
			value := v.(bool)
			*dest = &value
		}
	}
	setID := d.Get("firewall_set_id").(int)
	vms, err := m.(*ProviderClients).V2.GetFirewallSetAvailableVMs(setID, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		out[i] = flattenFirewallSetVM(vm)
	}
	var diags diag.Diagnostics
	setValue("vms", out, d, &diags)
	d.SetId(fmt.Sprintf("firewall-set-%d-available-vms", setID))
	return diags
}
