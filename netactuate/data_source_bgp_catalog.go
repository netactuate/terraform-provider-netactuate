package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceBGPASN() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPASNRead,
		Schema: map[string]*schema.Schema{
			"bgp_asn_id": {Type: schema.TypeInt, Required: true, Description: "BGP ASN ID to read."},
			"asn":        {Type: schema.TypeInt, Computed: true, Description: "Autonomous system number."},
			"name":       {Type: schema.TypeString, Computed: true, Description: "ASN name."},
			"group_type": {Type: schema.TypeString, Computed: true, Description: "BGP group type."},
		},
	}
}

func dataSourceBGPGroups() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPGroupsRead,
		Schema: map[string]*schema.Schema{
			"group_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BGP group type filter. The API defaults this to anycast when omitted.",
			},
			"groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "BGP groups visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"bgp_group_id": {Type: schema.TypeInt, Computed: true, Description: "BGP group ID."},
					"name":         {Type: schema.TypeString, Computed: true, Description: "BGP group name."},
					"description":  {Type: schema.TypeString, Computed: true, Description: "BGP group description."},
					"group_type":   {Type: schema.TypeString, Computed: true, Description: "BGP group type."},
				}},
			},
		},
	}
}

func dataSourceBGPGroupsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	groups, err := c.ListBGPGroups(d.Get("group_type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	result := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		result[i] = map[string]interface{}{
			"bgp_group_id": group.ID,
			"name":         group.Name,
			"description":  group.Description,
			"group_type":   group.GroupType,
		}
	}

	var diags diag.Diagnostics
	setValue("groups", result, d, &diags)
	d.SetId("bgp-groups")
	return diags
}

func dataSourceBGPASNRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	asn, err := c.GetBGPASN(d.Get("bgp_asn_id").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setBGPASNState(asn, d, &diags)
	d.SetId(strconv.Itoa(asn.ID))
	return diags
}

func dataSourceBGPPrefix() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPPrefixRead,
		Schema: map[string]*schema.Schema{
			"bgp_prefix_id":   {Type: schema.TypeInt, Required: true, Description: "BGP prefix ID to read."},
			"name":            {Type: schema.TypeString, Computed: true, Description: "Prefix name."},
			"prefix":          {Type: schema.TypeString, Computed: true, Description: "BGP prefix."},
			"group_id":        {Type: schema.TypeInt, Computed: true, Description: "BGP group ID."},
			"asn_id":          {Type: schema.TypeInt, Computed: true, Description: "BGP ASN ID."},
			"anycast_profile": {Type: schema.TypeInt, Computed: true, Description: "Anycast profile ID."},
			"agreement_id":    {Type: schema.TypeInt, Computed: true, Description: "Agreement ID used for the purchase."},
		},
	}
}

func dataSourceBGPPrefixes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPPrefixesRead,
		Schema: map[string]*schema.Schema{
			"group_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BGP group type filter. The API defaults this to anycast when omitted.",
			},
			"prefixes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "BGP prefixes visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"bgp_prefix_id":   {Type: schema.TypeInt, Computed: true, Description: "BGP prefix ID."},
					"name":            {Type: schema.TypeString, Computed: true, Description: "Prefix name."},
					"prefix":          {Type: schema.TypeString, Computed: true, Description: "BGP prefix."},
					"group_id":        {Type: schema.TypeInt, Computed: true, Description: "BGP group ID."},
					"asn_id":          {Type: schema.TypeInt, Computed: true, Description: "BGP ASN ID."},
					"anycast_profile": {Type: schema.TypeInt, Computed: true, Description: "Anycast profile ID."},
					"agreement_id":    {Type: schema.TypeInt, Computed: true, Description: "Agreement ID used for the purchase."},
				}},
			},
		},
	}
}

func dataSourceBGPPrefixesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	prefixes, err := c.ListBGPPrefixes(d.Get("group_type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	result := make([]map[string]interface{}, len(prefixes))
	for i, prefix := range prefixes {
		result[i] = map[string]interface{}{
			"bgp_prefix_id":   prefix.ID,
			"name":            prefix.Name,
			"prefix":          prefix.Prefix,
			"group_id":        prefix.GroupID,
			"asn_id":          prefix.ASNID,
			"anycast_profile": prefix.AnycastProfile,
			"agreement_id":    prefix.AgreementID,
		}
	}

	var diags diag.Diagnostics
	setValue("prefixes", result, d, &diags)
	d.SetId("bgp-prefixes")
	return diags
}

func dataSourceBGPPrefixRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	prefix, err := c.GetBGPPrefix(d.Get("bgp_prefix_id").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setBGPPrefixState(prefix, d, &diags)
	d.SetId(strconv.Itoa(prefix.ID))
	return diags
}

func dataSourceBGPSummary() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPSummaryRead,
		Schema: map[string]*schema.Schema{
			"summary_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compact JSON BGP summary payload returned by the API.",
			},
		},
	}
}

func dataSourceBGPDashboard() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPDashboardRead,
		Schema: map[string]*schema.Schema{
			"group_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BGP group type filter.",
			},
			"flap_window": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Dashboard flap window filter.",
			},
			"dashboard_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compact JSON BGP dashboard payload returned by the API.",
			},
		},
	}
}

func dataSourceBGPASNs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPASNsRead,
		Schema: map[string]*schema.Schema{
			"group_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "BGP group type filter. The API defaults this to anycast when omitted.",
			},
			"asns": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "BGP ASNs visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"bgp_asn_id": {Type: schema.TypeInt, Computed: true, Description: "BGP ASN ID."},
					"asn":        {Type: schema.TypeInt, Computed: true, Description: "Autonomous system number."},
					"name":       {Type: schema.TypeString, Computed: true, Description: "ASN name."},
					"group_type": {Type: schema.TypeString, Computed: true, Description: "BGP group type."},
				}},
			},
		},
	}
}

func dataSourceBGPASNsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	asns, err := c.ListBGPASNs(d.Get("group_type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	result := make([]map[string]interface{}, len(asns))
	for i, asn := range asns {
		result[i] = map[string]interface{}{
			"bgp_asn_id": asn.ID,
			"asn":        asn.ASN,
			"name":       asn.Name,
			"group_type": asn.GroupType,
		}
	}

	var diags diag.Diagnostics
	setValue("asns", result, d, &diags)
	d.SetId("bgp-asns")
	return diags
}

func dataSourceBGPSummaryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	summary, err := c.GetBGPSummary()
	if err != nil {
		return diag.FromErr(err)
	}

	summaryJSON, err := rawMapString(summary)
	if err != nil {
		return diag.FromErr(fmt.Errorf("encode BGP summary: %w", err))
	}

	var diags diag.Diagnostics
	setValue("summary_json", summaryJSON, d, &diags)
	d.SetId("bgp-summary")
	return diags
}

func dataSourceBGPDashboardRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	opts := gona.BGPDashboardOptions{GroupType: d.Get("group_type").(string)}
	if v, ok := d.GetOk("flap_window"); ok {
		flapWindow := v.(int)
		opts.FlapWindow = &flapWindow
	}

	dashboard, err := c.GetBGPDashboard(opts)
	if err != nil {
		return diag.FromErr(err)
	}

	dashboardJSON, err := rawMapString(dashboard)
	if err != nil {
		return diag.FromErr(fmt.Errorf("encode BGP dashboard: %w", err))
	}

	var diags diag.Diagnostics
	setValue("dashboard_json", dashboardJSON, d, &diags)
	d.SetId("bgp-dashboard")
	return diags
}

func setBGPASNState(asn *gona.BGPASN, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("bgp_asn_id", asn.ID, d, diags)
	setValue("asn", asn.ASN, d, diags)
	setValue("name", asn.Name, d, diags)
	setValue("group_type", asn.GroupType, d, diags)
}

func dataSourceAccountAgreements() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAccountAgreementsRead,
		Schema: map[string]*schema.Schema{
			"agreements": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Legal agreements visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"agreement_id": {Type: schema.TypeInt, Computed: true, Description: "Agreement ID."},
					"name":         {Type: schema.TypeString, Computed: true, Description: "Agreement name."},
					"title":        {Type: schema.TypeString, Computed: true, Description: "Agreement title."},
					"description":  {Type: schema.TypeString, Computed: true, Description: "Agreement description."},
					"version":      {Type: schema.TypeString, Computed: true, Description: "Agreement version."},
				}},
			},
		},
	}
}

func dataSourceAccountAgreementsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	agreements, err := c.ListAccountAgreements()
	if err != nil {
		return diag.FromErr(err)
	}

	result := make([]map[string]interface{}, len(agreements))
	for i, agreement := range agreements {
		result[i] = map[string]interface{}{
			"agreement_id": agreement.ID,
			"name":         agreement.Name,
			"title":        agreement.Title,
			"description":  agreement.Description,
			"version":      agreement.Version,
		}
	}

	var diags diag.Diagnostics
	setValue("agreements", result, d, &diags)
	d.SetId("account-agreements")
	return diags
}
