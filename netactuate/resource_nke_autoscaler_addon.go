package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
	"regexp"
	"strconv"
	"time"
)

var autoscalerDurationPattern = regexp.MustCompile(`^[0-9]{1,4}[smh]$`)

const nkeAutoscalerAddonType = "netactuate-cluster-autoscaler"

func validateAutoscalerDuration(minimum, maximum time.Duration) schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(v interface{}, key string) ([]string, []error) {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil || duration < minimum || duration > maximum || !autoscalerDurationPattern.MatchString(s) {
			return nil, []error{fmt.Errorf("%s must be a duration from %s to %s", key, minimum, maximum)}
		}
		return nil, nil
	})
}
func resourceNKEAutoscalerAddon() *schema.Resource {
	return &schema.Resource{CreateContext: resourceNKEAutoscalerAddonCreate, ReadContext: resourceNKEAutoscalerAddonRead, UpdateContext: resourceNKEAutoscalerAddonUpdate, DeleteContext: resourceNKEAutoscalerAddonDelete, Importer: &schema.ResourceImporter{StateContext: resourceNKEAutoscalerAddonImport},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			return validateNKEAddonPlan(ctx, d, m, nkeAutoscalerAddonType)
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {Type: schema.TypeInt, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)), Description: "Cluster ID."},
			"version":    {Type: schema.TypeString, Optional: true, Computed: true, Description: "Autoscaler addon version."},
			"config": {Type: schema.TypeList, Optional: true, Computed: true, MaxItems: 1, Description: "Autoscaler catalog configuration.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"scale_down_enabled":               {Type: schema.TypeBool, Optional: true, Computed: true, Description: "Catalog setting scaleDownEnabled."},
				"scale_down_unneeded_time":         {Type: schema.TypeString, Optional: true, Computed: true, Description: "Catalog setting scaleDownUnneededTime.", ValidateDiagFunc: validateAutoscalerDuration(time.Minute, 24*time.Hour)},
				"scale_down_delay_after_add":       {Type: schema.TypeString, Optional: true, Computed: true, Description: "Catalog setting scaleDownDelayAfterAdd.", ValidateDiagFunc: validateAutoscalerDuration(time.Minute, 24*time.Hour)},
				"scale_down_utilization_threshold": {Type: schema.TypeFloat, Optional: true, Computed: true, Description: "Catalog setting scaleDownUtilizationThreshold.", ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(0.1, 0.9))},
				"max_node_provision_time":          {Type: schema.TypeString, Optional: true, Computed: true, Description: "Catalog setting maxNodeProvisionTime.", ValidateDiagFunc: validateAutoscalerDuration(5*time.Minute, time.Hour)},
				"expander":                         {Type: schema.TypeString, Optional: true, Computed: true, Description: "Catalog setting expander.", ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"least-waste", "priority"}, false))},
				"skip_nodes_with_local_storage":    {Type: schema.TypeBool, Optional: true, Computed: true, Description: "Catalog setting skipNodesWithLocalStorage."},
				"balance_similar_node_groups":      {Type: schema.TypeBool, Optional: true, Computed: true, Description: "Catalog setting balanceSimilarNodeGroups."},
				"max_nodes_total":                  {Type: schema.TypeInt, Optional: true, Computed: true, Description: "Catalog setting maxNodesTotal.", ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 1000))},
			}}},
		}}
}
func expandNKEAutoscalerConfig(d *schema.ResourceData) *gona.NKEAutoscalerConfig {
	config := &gona.NKEAutoscalerConfig{}
	a := d.Get("config").([]interface{})
	if len(a) == 0 || a[0] == nil {
		return config
	}
	v := a[0].(map[string]interface{})
	raw := d.GetRawConfig()
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("scale_down_enabled")
			if item.IsKnown() && !item.IsNull() {
				x := v["scale_down_enabled"].(bool)
				config.ScaleDownEnabled = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("scale_down_unneeded_time")
			if item.IsKnown() && !item.IsNull() {
				x := v["scale_down_unneeded_time"].(string)
				config.ScaleDownUnneededTime = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("scale_down_delay_after_add")
			if item.IsKnown() && !item.IsNull() {
				x := v["scale_down_delay_after_add"].(string)
				config.ScaleDownDelayAfterAdd = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("scale_down_utilization_threshold")
			if item.IsKnown() && !item.IsNull() {
				x := v["scale_down_utilization_threshold"].(float64)
				config.ScaleDownUtilizationThreshold = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("max_node_provision_time")
			if item.IsKnown() && !item.IsNull() {
				x := v["max_node_provision_time"].(string)
				config.MaxNodeProvisionTime = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("expander")
			if item.IsKnown() && !item.IsNull() {
				x := v["expander"].(string)
				config.Expander = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("skip_nodes_with_local_storage")
			if item.IsKnown() && !item.IsNull() {
				x := v["skip_nodes_with_local_storage"].(bool)
				config.SkipNodesWithLocalStorage = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("balance_similar_node_groups")
			if item.IsKnown() && !item.IsNull() {
				x := v["balance_similar_node_groups"].(bool)
				config.BalanceSimilarNodeGroups = &x
			}
		}
	}
	if !raw.IsNull() && raw.Type().HasAttribute("config") {
		block := raw.GetAttr("config")
		if block.IsKnown() && !block.IsNull() && block.LengthInt() > 0 {
			item := block.Index(cty.NumberIntVal(0)).GetAttr("max_nodes_total")
			if item.IsKnown() && !item.IsNull() {
				x := v["max_nodes_total"].(int)
				config.MaxNodesTotal = &x
			}
		}
	}
	return config
}
func flattenNKEAutoscalerConfig(c gona.NKEAutoscalerConfig) []interface{} {
	v := map[string]interface{}{}
	if c.ScaleDownEnabled != nil {
		v["scale_down_enabled"] = *c.ScaleDownEnabled
	}
	if c.ScaleDownUnneededTime != nil {
		v["scale_down_unneeded_time"] = *c.ScaleDownUnneededTime
	}
	if c.ScaleDownDelayAfterAdd != nil {
		v["scale_down_delay_after_add"] = *c.ScaleDownDelayAfterAdd
	}
	if c.ScaleDownUtilizationThreshold != nil {
		v["scale_down_utilization_threshold"] = *c.ScaleDownUtilizationThreshold
	}
	if c.MaxNodeProvisionTime != nil {
		v["max_node_provision_time"] = *c.MaxNodeProvisionTime
	}
	if c.Expander != nil {
		v["expander"] = *c.Expander
	}
	if c.SkipNodesWithLocalStorage != nil {
		v["skip_nodes_with_local_storage"] = *c.SkipNodesWithLocalStorage
	}
	if c.BalanceSimilarNodeGroups != nil {
		v["balance_similar_node_groups"] = *c.BalanceSimilarNodeGroups
	}
	if c.MaxNodesTotal != nil {
		v["max_nodes_total"] = *c.MaxNodesTotal
	}
	if len(v) == 0 {
		return []interface{}{}
	}
	return []interface{}{v}
}
func resourceNKEAutoscalerAddonCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("cluster_id").(int)
	_, err := m.(*ProviderClients).V3.CreateClusterAddon(id, &gona.CreateNKEAddonRequest{AddonType: nkeAutoscalerAddonType, Version: d.Get("version").(string), Config: expandNKEAutoscalerConfig(d)})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(id))
	return resourceNKEAutoscalerAddonRead(ctx, d, m)
}
func resourceNKEAutoscalerAddonUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("cluster_id").(int)
	_, err := m.(*ProviderClients).V3.UpdateClusterAddon(id, nkeAutoscalerAddonType, &gona.UpdateNKEAddonRequest{Version: d.Get("version").(string), Config: expandNKEAutoscalerConfig(d)})
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceNKEAutoscalerAddonRead(ctx, d, m)
}
func resourceNKEAutoscalerAddonRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	a, err := m.(*ProviderClients).V3.GetClusterAddon(id, nkeAutoscalerAddonType)
	if gona.IsV3NotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("cluster_id", id, d, &diags)
	setValue("version", a.Version, d, &diags)
	setValue("config", flattenNKEAutoscalerConfig(a.Config.NKEAutoscalerConfig), d, &diags)
	return diags
}
func resourceNKEAutoscalerAddonDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err = m.(*ProviderClients).V3.DeleteClusterAddon(id, nkeAutoscalerAddonType); err != nil && !gona.IsV3NotFound(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
func resourceNKEAutoscalerAddonImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	id, err := strconv.Atoi(d.Id())
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid import ID %q, expected clusterId", d.Id())
	}
	d.SetId(strconv.Itoa(id))
	if err = d.Set("cluster_id", id); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
