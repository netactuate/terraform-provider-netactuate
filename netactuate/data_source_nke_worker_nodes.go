package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNKEWorkerNodes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKEWorkerNodesRead,
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the NKE cluster",
			},
			"worker_nodes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of worker nodes in the cluster",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"worker_node_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the worker node",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The hostname of the worker node",
						},
						"status_ready": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the worker node is ready",
						},
						"package_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The package/plan ID of the worker node",
						},
						"package_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The package/plan name of the worker node",
						},
						"location_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The location ID of the worker node",
						},
						"location_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The location name of the worker node",
						},
					},
				},
			},
		},
	}
}

func dataSourceNKEWorkerNodesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clusterID := d.Get("cluster_id").(int)

	nodes, err := c.ListNKEWorkerNodes(clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics


	var clusterLocationID int
	if cluster, err := c.GetNKECluster(clusterID); err == nil {
		clusterLocationID = cluster.Location.ID
	}

	workerNodes := make([]interface{}, 0, len(nodes))
	for _, node := range nodes {
		locationID := node.Location.ID
		if locationID == 0 {
			locationID = node.LocationID
		}
		if locationID == 0 {
			locationID = clusterLocationID
		}
		workerNodes = append(workerNodes, map[string]interface{}{
			"worker_node_id": node.WorkerNodeID,
			"name":           node.Name,
			"status_ready":   node.Status.Ready,
			"package_id":     node.Package.ID,
			"package_name":   node.Package.Name,
			"location_id":    locationID,
			"location_name":  node.Location.Name,
		})
	}

	setValue("worker_nodes", workerNodes, d, &diags)
	d.SetId(fmt.Sprintf("nke-worker-nodes-%d", clusterID))

	return diags
}
