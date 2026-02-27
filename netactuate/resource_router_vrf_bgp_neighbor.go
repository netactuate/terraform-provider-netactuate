package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFBGPNeighbor() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFBGPNeighborCreate,
		ReadContext:   resourceRouterVRFBGPNeighborRead,
		UpdateContext: resourceRouterVRFBGPNeighborUpdate,
		DeleteContext: resourceRouterVRFBGPNeighborDelete,
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"vrf_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VRF.",
			},
			"neighbor_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the BGP neighbor.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name your BGP neighbor something if you want to.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the BGP neighbor.",
			},
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP address of the neighbor.",
			},
			"is_shutdown": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not the neighbor is administratively shut down.",
			},
			"do_as_override": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, this enabled as-override, overriding the ASN in outbound updates to your local ASN. This may be required if you peer with magic mesh or the cloud router from multiple peers that share the same ASN.",
			},
			"do_next_hop_self": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If true, this enables nexthop-self, which is very common for eBGP peering.",
			},
			"source_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The IP address to optionally use as the source when establishing the BGP session.",
			},
			"ipv4_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not to enable the IPv4 address family.",
			},
			"ipv6_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not to enable the IPv6 address family.",
			},
			"ebgp_multihop": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 255),
				Description:  "The ebgp-multihop configuration for the neighbor.",
			},
			"remote_asn": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, 4294967294),
				Description:  "The ASN for your neighbor.",
			},
			"md5_secret": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringLenBetween(1, 128),
				Description:  "The MD5 secret with your neighbor.",
			},
			"import_default_drop": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, drop all routes that don't match a rule on import. If false, allow them by default.",
			},
			"import_rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of routing rules on import.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"prefix_list_id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The ID of the prefix list.",
						},
						"action": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"permit", "deny", "next"}, false),
							Description:  "What to do when matching the prefix list.",
						},
						"set_local_preference": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
							Description:  "The local preference value.",
						},
					},
				},
			},
			"export_default_drop": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, drop all routes that don't match a rule on export. If false, allow them by default.",
			},
			"export_rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of routing rules on export.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"prefix_list_id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The ID of the prefix list.",
						},
						"action": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"permit", "deny", "next"}, false),
							Description:  "What to do when matching the prefix list.",
						},
						"prepend_last_asn": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 10),
							Description:  "How many times to prepend the last ASN in the as-path.",
						},
					},
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceRouterVRFBGPNeighborCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	createRequest := gona.CreateRouterVRFBGPNeighborRequest{
		Address:        d.Get("address").(string),
		IsShutdown:     d.Get("is_shutdown").(bool),
		DoAsOverride:   d.Get("do_as_override").(bool),
		DoNextHelpSelf: d.Get("do_next_hop_self").(bool),
		EnabledIPVersion: gona.BGPNeighborEnabledIPVersion{
			IPv4: d.Get("ipv4_enabled").(bool),
			IPv6: d.Get("ipv6_enabled").(bool),
		},
		ASN: gona.BGPNeighborASN{
			Remote: d.Get("remote_asn").(int),
		},
	}

	if v, ok := d.GetOk("name"); ok {
		createRequest.Name = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		createRequest.Description = v.(string)
	}

	if v, ok := d.GetOk("source_address"); ok {
		createRequest.Source = &gona.BGPNeighborSource{
			Address: v.(string),
		}
	}

	if v, ok := d.GetOk("ebgp_multihop"); ok {
		multihop := v.(int)
		createRequest.EbgpMultihop = &multihop
	}

	if v, ok := d.GetOk("md5_secret"); ok {
		createRequest.MD5Secret = v.(string)
	}

	if v, ok := d.GetOk("import_rules"); ok {
		importRules := v.([]interface{})
		rules := make([]gona.BGPNeighborRouteMapRule, len(importRules))
		for i, r := range importRules {
			ruleMap := r.(map[string]interface{})
			rules[i] = gona.BGPNeighborRouteMapRule{
				PrefixListID: ruleMap["prefix_list_id"].(int),
				Action:       ruleMap["action"].(string),
			}
			if localPref, ok := ruleMap["set_local_preference"].(int); ok && localPref > 0 {
				rules[i].SetLocalPreference = &localPref
			}
		}
		createRequest.Import = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("import_default_drop").(bool),
			Rules:         rules,
		}
	} else {
		createRequest.Import = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("import_default_drop").(bool),
			Rules:         []gona.BGPNeighborRouteMapRule{},
		}
	}

	if v, ok := d.GetOk("export_rules"); ok {
		exportRules := v.([]interface{})
		rules := make([]gona.BGPNeighborRouteMapRule, len(exportRules))
		for i, r := range exportRules {
			ruleMap := r.(map[string]interface{})
			rules[i] = gona.BGPNeighborRouteMapRule{
				PrefixListID: ruleMap["prefix_list_id"].(int),
				Action:       ruleMap["action"].(string),
			}
			if prependAsn, ok := ruleMap["prepend_last_asn"].(int); ok && prependAsn > 0 {
				rules[i].PrependLastAsn = &prependAsn
			}
		}
		createRequest.Export = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("export_default_drop").(bool),
			Rules:         rules,
		}
	} else {
		createRequest.Export = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("export_default_drop").(bool),
			Rules:         []gona.BGPNeighborRouteMapRule{},
		}
	}

	resp, err := c.CreateRouterVRFBGPNeighbor(routerID, vrfID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	d.SetId(strconv.Itoa(resp.NeighborID))
	setValue("neighbor_id", resp.NeighborID, d, &diags)

	return resourceRouterVRFBGPNeighborRead(ctx, d, m)
}

func resourceRouterVRFBGPNeighborRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	neighborID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error parsing neighbor ID: %w", err))
	}

	neighbor, err := c.GetRouterVRFBGPNeighbor(routerID, vrfID, neighborID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("neighbor_id", neighbor.NeighborID, d, &diags)
	setValue("name", neighbor.Name, d, &diags)
	setValue("description", neighbor.Description, d, &diags)
	setValue("address", neighbor.Address, d, &diags)
	setValue("is_shutdown", neighbor.IsShutdown, d, &diags)
	setValue("do_as_override", neighbor.DoAsOverride, d, &diags)
	setValue("do_next_hop_self", neighbor.DoNextHelpSelf, d, &diags)
	setValue("ipv4_enabled", neighbor.EnabledIPVersion.IPv4, d, &diags)
	setValue("ipv6_enabled", neighbor.EnabledIPVersion.IPv6, d, &diags)
	setValue("remote_asn", neighbor.ASN.Remote, d, &diags)

	if neighbor.Source != nil {
		setValue("source_address", neighbor.Source.Address, d, &diags)
	}

	if neighbor.EbgpMultihop != nil {
		setValue("ebgp_multihop", *neighbor.EbgpMultihop, d, &diags)
	}

	if neighbor.MD5Secret != "" {
		setValue("md5_secret", neighbor.MD5Secret, d, &diags)
	}

	if neighbor.Import != nil {
		setValue("import_default_drop", neighbor.Import.DoDefaultDrop, d, &diags)
		if len(neighbor.Import.Rules) > 0 {
			importRules := make([]map[string]interface{}, len(neighbor.Import.Rules))
			for i, rule := range neighbor.Import.Rules {
				importRules[i] = map[string]interface{}{
					"prefix_list_id": rule.PrefixListID,
					"action":         rule.Action,
				}
				if rule.SetLocalPreference != nil {
					importRules[i]["set_local_preference"] = *rule.SetLocalPreference
				}
			}
			setValue("import_rules", importRules, d, &diags)
		}
	}

	if neighbor.Export != nil {
		setValue("export_default_drop", neighbor.Export.DoDefaultDrop, d, &diags)
		if len(neighbor.Export.Rules) > 0 {
			exportRules := make([]map[string]interface{}, len(neighbor.Export.Rules))
			for i, rule := range neighbor.Export.Rules {
				exportRules[i] = map[string]interface{}{
					"prefix_list_id": rule.PrefixListID,
					"action":         rule.Action,
				}
				if rule.PrependLastAsn != nil {
					exportRules[i]["prepend_last_asn"] = *rule.PrependLastAsn
				}
			}
			setValue("export_rules", exportRules, d, &diags)
		}
	}

	return diags
}

func resourceRouterVRFBGPNeighborUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	neighborID := d.Get("neighbor_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	updateRequest := gona.UpdateRouterVRFBGPNeighborRequest{
		Address:        d.Get("address").(string),
		IsShutdown:     d.Get("is_shutdown").(bool),
		DoAsOverride:   d.Get("do_as_override").(bool),
		DoNextHelpSelf: d.Get("do_next_hop_self").(bool),
		EnabledIPVersion: gona.BGPNeighborEnabledIPVersion{
			IPv4: d.Get("ipv4_enabled").(bool),
			IPv6: d.Get("ipv6_enabled").(bool),
		},
		ASN: gona.BGPNeighborASN{
			Remote: d.Get("remote_asn").(int),
		},
	}

	if v, ok := d.GetOk("name"); ok {
		updateRequest.Name = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		updateRequest.Description = v.(string)
	}

	if v, ok := d.GetOk("source_address"); ok {
		updateRequest.Source = &gona.BGPNeighborSource{
			Address: v.(string),
		}
	}

	if v, ok := d.GetOk("ebgp_multihop"); ok {
		multihop := v.(int)
		updateRequest.EbgpMultihop = &multihop
	}

	if v, ok := d.GetOk("md5_secret"); ok {
		updateRequest.MD5Secret = v.(string)
	}

	if v, ok := d.GetOk("import_rules"); ok {
		importRules := v.([]interface{})
		rules := make([]gona.BGPNeighborRouteMapRule, len(importRules))
		for i, r := range importRules {
			ruleMap := r.(map[string]interface{})
			rules[i] = gona.BGPNeighborRouteMapRule{
				PrefixListID: ruleMap["prefix_list_id"].(int),
				Action:       ruleMap["action"].(string),
			}
			if localPref, ok := ruleMap["set_local_preference"].(int); ok && localPref > 0 {
				rules[i].SetLocalPreference = &localPref
			}
		}
		updateRequest.Import = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("import_default_drop").(bool),
			Rules:         rules,
		}
	} else {
		updateRequest.Import = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("import_default_drop").(bool),
			Rules:         []gona.BGPNeighborRouteMapRule{},
		}
	}

	if v, ok := d.GetOk("export_rules"); ok {
		exportRules := v.([]interface{})
		rules := make([]gona.BGPNeighborRouteMapRule, len(exportRules))
		for i, r := range exportRules {
			ruleMap := r.(map[string]interface{})
			rules[i] = gona.BGPNeighborRouteMapRule{
				PrefixListID: ruleMap["prefix_list_id"].(int),
				Action:       ruleMap["action"].(string),
			}
			if prependAsn, ok := ruleMap["prepend_last_asn"].(int); ok && prependAsn > 0 {
				rules[i].PrependLastAsn = &prependAsn
			}
		}
		updateRequest.Export = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("export_default_drop").(bool),
			Rules:         rules,
		}
	} else {
		updateRequest.Export = &gona.BGPNeighborRouteMap{
			DoDefaultDrop: d.Get("export_default_drop").(bool),
			Rules:         []gona.BGPNeighborRouteMapRule{},
		}
	}

	_, err := c.UpdateRouterVRFBGPNeighbor(routerID, vrfID, neighborID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterVRFBGPNeighborRead(ctx, d, m)
}

func resourceRouterVRFBGPNeighborDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	neighborID := d.Get("neighbor_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	err := c.DeleteRouterVRFBGPNeighbor(routerID, vrfID, neighborID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}