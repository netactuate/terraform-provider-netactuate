package netactuate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// nkeWorkerPrivateIPsInterval is a var so tests can shorten it.
var nkeWorkerPrivateIPsInterval = 10 * time.Second

const nkeWorkerPrivateIPsDescription = "Maps each worker node of an NKE cluster in a VPC to its private IP, for load balancer backends. " +
	"The cluster must be in a VPC that the API key's account owns. " +
	"The read waits, within timeouts { read }, until at least the cluster's minimum number of workers have an address, " +
	"and while workers above the maximum are being removed, so it returns the new set of workers in the same apply as a node count change."

func dataSourceNKEWorkerPrivateIPs() *schema.Resource {
	return &schema.Resource{
		Description: nkeWorkerPrivateIPsDescription,
		ReadContext: dataSourceNKEWorkerPrivateIPsRead,
		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(20 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the NKE cluster. The cluster must be in a VPC that the API key's account owns",
			},
			"private_ips": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Worker node name to private IP in the cluster's VPC, for each worker that has one",
			},
		},
	}
}

func dataSourceNKEWorkerPrivateIPsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	for {
		cluster, err := c.GetNKECluster(clusterID)
		if err != nil {
			return diag.FromErr(err)
		}
		if cluster.VpcID == 0 {
			return diag.Errorf("NKE cluster %d is not in a VPC: its workers have no VPC private IPs", clusterID)
		}
		ips, err := nkeWorkerPrivateIPs(c, clusterID, cluster.VpcID)
		if err != nil {
			return diag.FromErr(err)
		}

		pending := nkeWorkerPrivateIPsPending(cluster, len(ips))
		if pending == "" {
			var diags diag.Diagnostics
			setValue("private_ips", ips, d, &diags)
			d.SetId(fmt.Sprintf("nke-worker-private-ips-%d", clusterID))
			return diags
		}
		// The SDK bounds ctx by timeouts { read }.
		select {
		case <-ctx.Done():
			return diag.Errorf("NKE cluster %d: %s at the read timeout. Raise timeouts { read } on the data source to wait longer", clusterID, pending)
		case <-time.After(nkeWorkerPrivateIPsInterval):
		}
	}
}

// A worker's private IP is its primary-interface reservation in the VPC, assigned while the worker
// builds, before it joins the cluster.
func nkeWorkerPrivateIPs(c *gona.V3Client, clusterID, vpcID int) (map[string]interface{}, error) {
	nodes, err := c.ListNKEWorkerNodes(clusterID)
	if err != nil {
		return nil, err
	}
	reservations, err := c.ListVPCIPReservations(vpcID)
	if err != nil {
		return nil, err
	}
	byMBPkgID := map[int]string{}
	for _, r := range reservations {
		if r.Type == "Primary Interface" && r.VM.MBPkgID > 0 {
			byMBPkgID[r.VM.MBPkgID] = strings.TrimSuffix(r.Network, "/32")
		}
	}
	ips := map[string]interface{}{}
	for _, n := range nodes {
		if ip := byMBPkgID[n.MBPkgID]; n.MBPkgID > 0 && ip != "" {
			ips[n.Name] = ip
		}
	}
	return ips, nil
}

// nkeWorkerPrivateIPsPending says what the read is still waiting for, or "" when it is done.
func nkeWorkerPrivateIPsPending(cluster *gona.NKECluster, withIP int) string {
	nodes := cluster.Nodes
	if withIP < nodes.Minimum {
		return fmt.Sprintf("%d of %d worker nodes have a private IP", withIP, nodes.Minimum)
	}
	// A rolling version upgrade runs one worker above the maximum; that is not a scale-down.
	if nodes.Outdated > 0 {
		return ""
	}
	if nodes.Maximum != nil && nodes.Total > *nodes.Maximum {
		return fmt.Sprintf("%d worker nodes, above the maximum of %d, are being reduced", nodes.Total, *nodes.Maximum)
	}
	if nodes.Deleting > 0 {
		return fmt.Sprintf("%d worker nodes are being removed", nodes.Deleting)
	}
	return ""
}
