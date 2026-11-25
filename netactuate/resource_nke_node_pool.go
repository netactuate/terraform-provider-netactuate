package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceNKENodePool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKENodePoolCreate, ReadContext: resourceNKENodePoolRead, UpdateContext: resourceNKENodePoolUpdate, DeleteContext: resourceNKENodePoolDelete,
		Importer: &schema.ResourceImporter{StateContext: resourceNKENodePoolImport},
		Timeouts: &schema.ResourceTimeout{Create: schema.DefaultTimeout(20 * time.Minute), Update: schema.DefaultTimeout(20 * time.Minute), Delete: schema.DefaultTimeout(20 * time.Minute)},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			if d.Get("type").(string) == "managed" && d.Get("package_id").(int) == 0 && d.NewValueKnown("package_id") {
				return fmt.Errorf("managed pools require package_id")
			}
			if d.Get("type").(string) == "manual" {
				raw := d.GetRawConfig()
				configured := false
				if !raw.IsNull() && raw.Type().HasAttribute("desired_nodes") {
					value := raw.GetAttr("desired_nodes")
					configured = value.IsKnown() && !value.IsNull()
				}
				if configured {
					return fmt.Errorf("desired_nodes is only supported for managed pools")
				}
			}
			if a := d.Get("autoscaling").([]interface{}); len(a) > 0 && a[0] != nil {
				v := a[0].(map[string]interface{})
				if v["maximum_nodes"].(int) > 0 && v["minimum_nodes"].(int) > v["maximum_nodes"].(int) {
					return fmt.Errorf("minimum_nodes must be less than or equal to maximum_nodes")
				}
				if d.Get("type") == "manual" && v["enabled"].(bool) {
					return fmt.Errorf("manual pools cannot enable autoscaling")
				}
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {Type: schema.TypeInt, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)), Description: "Cluster that owns the pool."},
			"name":       {Type: schema.TypeString, Required: true, ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringLenBetween(1, 63), validation.StringMatch(regexp.MustCompile(`^[a-z0-9]([-a-z0-9_.]*[a-z0-9])?$`), "must be a lowercase Kubernetes label value"), validation.StringNotInSlice([]string{"default"}, false))), Description: "Unique pool name. The reserved default pool cannot be managed here."},
			"type":       {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"managed", "manual"}, false)), Description: "Pool type: managed or manual. Sent as mode to the API."},
			"package_id": {Type: schema.TypeInt, Optional: true, Computed: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)), Description: "Worker package ID. Required for managed pools."},
			"labels":     {Type: schema.TypeMap, Optional: true, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "Kubernetes labels applied to pool workers."},
			"taints": {Type: schema.TypeList, Optional: true, Computed: true, Description: "Kubernetes taints applied to pool workers.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"key":    {Type: schema.TypeString, Required: true, Description: "Taint key."},
				"value":  {Type: schema.TypeString, Optional: true, Description: "Taint value."},
				"effect": {Type: schema.TypeString, Required: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"NoSchedule", "PreferNoSchedule", "NoExecute"}, false)), Description: "Taint effect."},
			}}},
			"autoscaling": {Type: schema.TypeList, Optional: true, Computed: true, MaxItems: 1, Description: "Pool autoscaling settings.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"enabled":       {Type: schema.TypeBool, Optional: true, Computed: true, Description: "Allow the cluster autoscaler to manage the pool."},
				"minimum_nodes": {Type: schema.TypeInt, Optional: true, Computed: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 64)), Description: "Minimum worker count, including zero."},
				"maximum_nodes": {Type: schema.TypeInt, Optional: true, Computed: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 64)), Description: "Maximum worker count. Omit for no upper limit."},
			}}},
			"desired_nodes":      {Type: schema.TypeInt, Optional: true, Computed: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 1000)), Description: "Desired managed worker count. Writes use the pool size endpoint."},
			"total_worker_nodes": {Type: schema.TypeInt, Computed: true, Description: "Total pool workers."},
			"ready_worker_nodes": {Type: schema.TypeInt, Computed: true, Description: "Ready pool workers."},
		},
	}
}

func expandNKENodePool(d *schema.ResourceData) *gona.CreateNKENodePoolRequest {
	r := &gona.CreateNKENodePoolRequest{Name: d.Get("name").(string), Mode: d.Get("type").(string)}
	if v, ok := d.GetOk("package_id"); ok {
		x := v.(int)
		r.PackageID = &x
	}
	r.Labels = map[string]string{}
	for k, v := range d.Get("labels").(map[string]interface{}) {
		r.Labels[k] = v.(string)
	}
	r.Taints = []gona.NKENodeTaint{}
	for _, raw := range d.Get("taints").([]interface{}) {
		v := raw.(map[string]interface{})
		r.Taints = append(r.Taints, gona.NKENodeTaint{Key: v["key"].(string), Value: v["value"].(string), Effect: v["effect"].(string)})
	}
	if a := d.Get("autoscaling").([]interface{}); len(a) > 0 {
		v := a[0].(map[string]interface{})
		minimum := v["minimum_nodes"].(int)
		if nkePoolBlockFieldConfigured(d, "minimum_nodes") {
			r.MinimumNodes = &minimum
		}
		if maximum := v["maximum_nodes"].(int); maximum > 0 && nkePoolBlockFieldConfigured(d, "maximum_nodes") {
			r.MaximumNodes = &maximum
		}
	}
	return r
}

func resourceNKENodePoolCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)
	id, err := c.CreateNKENodePool(clusterID, expandNKENodePool(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", clusterID, id))
	return applyNKENodePool(ctx, d, m, schema.TimeoutCreate, true)
}
func resourceNKENodePoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return applyNKENodePool(ctx, d, m, schema.TimeoutUpdate, false)
}
func applyNKENodePool(ctx context.Context, d *schema.ResourceData, m interface{}, phase string, created bool) diag.Diagnostics {
	clusterID, id, err := parseSurfaceTwoPartID(d.Id(), "clusterId/poolId")
	if err != nil {
		return diag.FromErr(err)
	}
	c := m.(*ProviderClients).V3
	r := expandNKENodePool(d)
	patch := &gona.UpdateNKENodePoolRequest{}
	if !created {
		patch.Name = &r.Name
		patch.PackageID = r.PackageID
		patch.Labels = &r.Labels
		patch.Taints = &r.Taints
		patch.MinimumNodes = r.MinimumNodes
		patch.MaximumNodes = r.MaximumNodes
	}
	if a := d.Get("autoscaling").([]interface{}); len(a) > 0 && nkePoolBlockFieldConfigured(d, "enabled") {
		enabled := a[0].(map[string]interface{})["enabled"].(bool)
		patch.AutoscalingEnabled = &enabled
	}
	if err = c.UpdateNKENodePool(clusterID, id, patch); err != nil {
		return diag.FromErr(err)
	}
	if (created && nkePoolDesiredConfigured(d)) || (!created && d.HasChange("desired_nodes")) {
		if err = c.SetNKENodePoolSize(clusterID, id, d.Get("desired_nodes").(int)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err = waitNKENodePool(ctx, c, clusterID, id, d.Timeout(phase), false); err != nil {
		return diag.Errorf("NKE node pool %s saved but failed to converge: %s", d.Id(), err)
	}
	return resourceNKENodePoolRead(ctx, d, m)
}
func nkePoolDesiredConfigured(d *schema.ResourceData) bool {
	raw := d.GetRawConfig()
	if !raw.IsNull() && raw.Type().HasAttribute("desired_nodes") {
		v := raw.GetAttr("desired_nodes")
		return !v.IsNull() && v.IsKnown()
	}
	return false
}
func waitNKENodePool(ctx context.Context, c *gona.V3Client, clusterID, id int, timeout time.Duration, empty bool) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		p, err := c.GetNKENodePool(clusterID, id)
		if err != nil {
			if empty && gona.IsV3NotFound(err) {
				return nil
			}
			return err
		}
		if empty {
			if p.Nodes.Total == 0 {
				return nil
			}
		} else {
			target := p.MinimumNodes
			if p.DesiredNodes != nil {
				target = *p.DesiredNodes
			}
			countReady := p.Nodes.Ready >= target
			if p.DesiredNodes != nil {
				countReady = p.Nodes.Ready == target && p.Nodes.Total == target
			}
			if countReady && p.Nodes.Building == 0 && p.Nodes.Deleting == 0 && p.Nodes.Outdated == 0 {
				return nil
			}
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return waitCtx.Err()
		case <-timer.C:
		}
	}
}
func flattenNKENodePool(p *gona.NKENodePool) map[string]interface{} {
	maximum := 0
	if p.MaximumNodes != nil {
		maximum = *p.MaximumNodes
	}
	taints := make([]interface{}, 0, len(p.Taints))
	for _, t := range p.Taints {
		taints = append(taints, map[string]interface{}{"key": t.Key, "value": t.Value, "effect": t.Effect})
	}
	out := map[string]interface{}{"name": p.Name, "type": p.Mode, "labels": p.Labels, "taints": taints, "autoscaling": []interface{}{map[string]interface{}{"enabled": p.AutoscalingEnabled == 1, "minimum_nodes": p.MinimumNodes, "maximum_nodes": maximum}}, "total_worker_nodes": p.Nodes.Total, "ready_worker_nodes": p.Nodes.Ready}
	if p.PackageID != nil {
		out["package_id"] = *p.PackageID
	}
	if p.DesiredNodes != nil {
		out["desired_nodes"] = *p.DesiredNodes
	} else {
		out["desired_nodes"] = nil
	}
	return out
}
func resourceNKENodePoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID, id, err := parseSurfaceTwoPartID(d.Id(), "clusterId/poolId")
	if err != nil {
		return diag.FromErr(err)
	}
	p, err := m.(*ProviderClients).V3.GetNKENodePool(clusterID, id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("cluster_id", clusterID, d, &diags)
	for k, v := range flattenNKENodePool(p) {
		setValue(k, v, d, &diags)
	}
	return diags
}
func resourceNKENodePoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID, id, err := parseSurfaceTwoPartID(d.Id(), "clusterId/poolId")
	if err != nil {
		return diag.FromErr(err)
	}
	c := m.(*ProviderClients).V3
	p, err := c.GetNKENodePool(clusterID, id)
	if gona.IsV3NotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if p.Name == "default" {
		return diag.Errorf("the default node pool cannot be deleted")
	}
	if p.Nodes.Total > 0 {
		zero := 0
		disabled := false
		if err = c.UpdateNKENodePool(clusterID, id, &gona.UpdateNKENodePoolRequest{MinimumNodes: &zero, AutoscalingEnabled: &disabled}); err != nil {
			return diag.FromErr(err)
		}
		if p.Mode == "managed" {
			if err = c.SetNKENodePoolSize(clusterID, id, 0); err != nil {
				return diag.FromErr(err)
			}
		} else {
			nodes, e := c.ListNKEWorkerNodes(clusterID)
			if e != nil {
				return diag.FromErr(e)
			}
			for _, node := range nodes {
				if node.PoolID == id {
					if e = c.RemoveNKEWorkerNode(clusterID, node.WorkerNodeID); e != nil && !gona.IsV3NotFound(e) {
						return diag.FromErr(e)
					}
				}
			}
		}
		if err = waitNKENodePool(ctx, c, clusterID, id, d.Timeout(schema.TimeoutDelete), true); err != nil {
			return diag.Errorf("NKE node pool %s failed to drain: %s", d.Id(), err)
		}
	}
	if err = c.DeleteNKENodePool(clusterID, id); err != nil && !gona.IsV3NotFound(err) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
func resourceNKENodePoolImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	clusterID, id, err := parseSurfaceTwoPartID(d.Id(), "clusterId/poolId")
	if err != nil || clusterID <= 0 || id <= 0 {
		return nil, fmt.Errorf("invalid import ID %q, expected clusterId/poolId", d.Id())
	}
	if err = d.Set("cluster_id", clusterID); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func nkePoolBlockFieldConfigured(d *schema.ResourceData, key string) bool {
	raw := d.GetRawConfig()
	if raw.IsNull() {
		_, ok := d.GetOkExists("autoscaling.0." + key)
		return ok
	}
	block := raw.GetAttr("autoscaling")
	if block.IsNull() || !block.IsKnown() || block.LengthInt() == 0 {
		return false
	}
	value := block.Index(cty.NumberIntVal(0)).GetAttr(key)
	return value.IsKnown() && !value.IsNull()
}
