package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceNKEAccessURLs() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKEAccessURLsCreate,
		ReadContext:   resourceNKEAccessURLsRead,
		DeleteContext: resourceNKEAccessURLsDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"cluster_id":           {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "NKE cluster ID for which access URLs are created."},
			"api":                  {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Secure URL for Kubernetes API access."},
			"prometheus":           {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Secure URL for Prometheus access."},
			"kubernetes_dashboard": {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Secure URL for Kubernetes dashboard access."},
		},
	}
}

func resourceNKEAccessURLsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID := d.Get("cluster_id").(int)
	urls, err := m.(*ProviderClients).V3.CreateNKEAccessURLs(clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("api", urls.API, d, &diags)
	setValue("prometheus", urls.Prometheus, d, &diags)
	setValue("kubernetes_dashboard", urls.KubernetesDashboard, d, &diags)
	d.SetId(strconv.Itoa(clusterID))
	return diags
}

func resourceNKEAccessURLsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	cluster, err := m.(*ProviderClients).V3.GetNKECluster(clusterID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("cluster_id", clusterID, d, &diags)
	setValue("api", cluster.URLs.API, d, &diags)
	setValue("prometheus", cluster.URLs.Prometheus, d, &diags)
	setValue("kubernetes_dashboard", cluster.URLs.KubernetesDashboard, d, &diags)
	return diags
}

func resourceNKEAccessURLsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "NKE access URLs remain platform managed", Detail: "The access URL endpoint has no delete operation. Terraform removed access URL marker " + id + " from state."}}
}

func resourceNKEWorkerNode() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKEWorkerNodeUpdate,
		ReadContext:   resourceNKEWorkerNodeRead,
		UpdateContext: resourceNKEWorkerNodeUpdate,
		DeleteContext: resourceNKEWorkerNodeDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceNKEWorkerNodeImport},
		Schema: map[string]*schema.Schema{
			"cluster_id":     {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "NKE cluster ID that owns the worker node."},
			"worker_node_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "NKE worker node ID."},
			"label":          {Type: schema.TypeString, Optional: true, Description: "Worker node label."},
			"tag_ids":        {Type: schema.TypeSet, Optional: true, Elem: &schema.Schema{Type: schema.TypeInt}, Description: "Tag IDs to apply to the worker node."},
			"name":           {Type: schema.TypeString, Computed: true, Description: "Worker node name."},
			"status_ready":   {Type: schema.TypeBool, Computed: true, Description: "Whether the worker node is ready."},
			"package_id":     {Type: schema.TypeInt, Computed: true, Description: "Worker node package ID."},
			"package_name":   {Type: schema.TypeString, Computed: true, Description: "Worker node package name."},
			"location_id":    {Type: schema.TypeInt, Computed: true, Description: "Worker node location ID."},
			"location_name":  {Type: schema.TypeString, Computed: true, Description: "Worker node location name."},
		},
	}
}

func resourceNKEWorkerNodeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID := d.Get("cluster_id").(int)
	workerNodeID := d.Get("worker_node_id").(int)
	req := &gona.UpdateNKEWorkerNodeRequest{}
	if v, ok := d.GetOk("label"); ok {
		req.Label = v.(string)
	}
	if v, ok := d.GetOk("tag_ids"); ok {
		for _, raw := range v.(*schema.Set).List() {
			req.Tags = append(req.Tags, gona.NKEClusterTagInput{TagID: raw.(int)})
		}
	}
	node, err := m.(*ProviderClients).V3.UpdateNKEWorkerNode(clusterID, workerNodeID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", clusterID, workerNodeID))
	// The PATCH response is minimal, so read the node back rather than trusting
	// it. Otherwise a value changed in the same apply is not available to
	// outputs or downstream resources until the next refresh.
	_ = node
	return resourceNKEWorkerNodeRead(ctx, d, m)
}

func resourceNKEWorkerNodeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID, workerNodeID, err := parseSurfaceTwoPartID(d.Id(), "clusterId/workerNodeId")
	if err != nil {
		return diag.FromErr(err)
	}
	node, err := m.(*ProviderClients).V3.GetNKEWorkerNode(clusterID, workerNodeID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setNKEWorkerNodeState(d, node)
}

func setNKEWorkerNodeState(d *schema.ResourceData, node *gona.NKEWorkerNode) diag.Diagnostics {
	var diags diag.Diagnostics
	setValue("cluster_id", node.ClusterID, d, &diags)
	setValue("worker_node_id", node.WorkerNodeID, d, &diags)
	setValue("name", node.Name, d, &diags)
	setValue("status_ready", node.Status.Ready, d, &diags)
	setValue("package_id", node.Package.ID, d, &diags)
	setValue("package_name", node.Package.Name, d, &diags)
	locationID := node.Location.ID
	if locationID == 0 {
		locationID = node.LocationID
	}
	setValue("location_id", locationID, d, &diags)
	setValue("location_name", node.Location.Name, d, &diags)
	return diags
}

func resourceNKEWorkerNodeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clusterID, workerNodeID, err := parseSurfaceTwoPartID(d.Id(), "clusterId/workerNodeId")
	if err != nil {
		return diag.FromErr(err)
	}
	if err := m.(*ProviderClients).V3.DeleteNKEWorkerNode(clusterID, workerNodeID); err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceNKEWorkerNodeImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	clusterID, workerNodeID, err := parseSurfaceTwoPartID(d.Id(), "clusterId/workerNodeId")
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected clusterId/workerNodeId: %w", d.Id(), err)
	}
	d.Set("cluster_id", clusterID)
	d.Set("worker_node_id", workerNodeID)
	return []*schema.ResourceData{d}, nil
}
