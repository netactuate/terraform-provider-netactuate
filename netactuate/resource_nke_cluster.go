package netactuate

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceNKECluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKEClusterCreate,
		ReadContext:   resourceNKEClusterRead,
		UpdateContext: resourceNKEClusterUpdate,
		DeleteContext: resourceNKEClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the NKE cluster",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the NKE cluster",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Kubernetes version.",
			},
			"replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				ForceNew:    true,
				Description: "Number of control plane replicas",
			},
			"minimum_nodes": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Minimum number of nodes (autoscaling lower bound)",
			},
			"maximum_nodes": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Maximum number of nodes (autoscaling upper bound)",
			},
			"do_autoscaling": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable autoscaling for this cluster",
			},
			"high_availability": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Enable high availability for the control plane (requires recreation)",
			},
			"kubernetes_dashboard": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Install the Kubernetes dashboard addon (requires recreation)",
			},
			// Location: accept name or ID
			"location": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "Location name where the cluster will be deployed (e.g. \"DEVRDU - Raleigh, NC\")",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "Location ID where the cluster will be deployed",
			},
			"plan": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Plan/package name for worker nodes (e.g. \"VR2x2x25\")",
			},
			"package_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The resolved plan/package ID (populated after apply)",
			},
			"pool_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Compute pool ID to assign the cluster to",
			},
			"contract_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Billing contract ID (required for some accounts, requires recreation to change)",
			},
			"tag_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Tag IDs to assign to the cluster. Note: the API does not return tag IDs so changes made outside Terraform may not be detected",
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
			// Computed outputs
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cluster status (e.g. \"Healthy\", \"Initializing\")",
			},
			"api_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Kubernetes API server URL",
			},
			"prometheus_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Prometheus metrics URL",
			},
			"kubernetes_dashboard_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Kubernetes dashboard URL (empty if dashboard addon not installed)",
			},
			"pod_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The pod network CIDR",
			},
			"service_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The service network CIDR",
			},
		},
	}
}

func resourceNKEClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v2 := m.(*ProviderClients).V2
	c := m.(*ProviderClients).V3

	locationID, locDiag := getLocationID(d, v2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	packageID, pkgDiag := getPackageID(d.Get("plan").(string), v2)
	if pkgDiag != nil {
		return diag.Diagnostics{*pkgDiag}
	}

	billing := gona.NKEBilling{
		PackageID:  packageID,
		LocationID: locationID,
	}
	if contractID := d.Get("contract_id").(int); contractID != 0 {
		billing.ContractID = &contractID
	}

	req := &gona.CreateNKEClusterRequest{
		Name:             d.Get("name").(string),
		Version:          d.Get("version").(string),
		Replicas:         d.Get("replicas").(int),
		MinimumNodes:     d.Get("minimum_nodes").(int),
		MaximumNodes:     d.Get("maximum_nodes").(int),
		DoAutoscaling:    d.Get("do_autoscaling").(bool),
		HighAvailability: d.Get("high_availability").(bool),
		Billing:          billing,
	}

	if poolID := d.Get("pool_id").(int); poolID != 0 {
		req.Pool = &gona.NKEPool{ID: poolID}
	}

	if d.Get("kubernetes_dashboard").(bool) {
		req.AddonsToInstall = &gona.NKEAddons{KubernetesDashboard: true}
	}

	if tagIDs := d.Get("tag_ids").(*schema.Set).List(); len(tagIDs) > 0 {
		for _, id := range tagIDs {
			req.Tags = append(req.Tags, gona.NKEClusterTagInput{TagID: id.(int)})
		}
	}

	clusterID, err := c.CreateNKECluster(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(clusterID))
	log.Printf("[INFO] NKE cluster created with ID: %d, waiting for Healthy status...", clusterID)

	if err := c.WaitForNKEClusterHealthy(clusterID); err != nil {
		return diag.Errorf("NKE cluster %d created but failed to become healthy: %s", clusterID, err)
	}

	return resourceNKEClusterRead(ctx, d, m)
}

func resourceNKEClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cluster, err := c.GetNKECluster(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NKE cluster %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("cluster_id", cluster.ClusterID, d, &diags)
	setValue("name", cluster.Name, d, &diags)
	setValue("version", cluster.Version.Active, d, &diags)
	setValue("replicas", cluster.Replicas, d, &diags)
	setValue("minimum_nodes", cluster.Nodes.Minimum, d, &diags)
	if cluster.Nodes.Maximum != nil {
		setValue("maximum_nodes", *cluster.Nodes.Maximum, d, &diags)
	}
	setValue("do_autoscaling", cluster.DoAutoscaling != 0, d, &diags)

	setValue("location_id", cluster.Location.ID, d, &diags)
	setValue("location", cluster.Location.Name, d, &diags)
	setValue("plan", cluster.Package.Name, d, &diags)
	setValue("package_id", cluster.Package.ID, d, &diags)

	setValue("status", cluster.Status.Cluster, d, &diags)
	setValue("api_url", cluster.URLs.API, d, &diags)
	setValue("prometheus_url", cluster.URLs.Prometheus, d, &diags)
	setValue("kubernetes_dashboard_url", cluster.URLs.KubernetesDashboard, d, &diags)
	setValue("pod_network", cluster.Networks.Pod, d, &diags)
	setValue("service_network", cluster.Networks.Service, d, &diags)

	return diags
}

func resourceNKEClusterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v2 := m.(*ProviderClients).V2
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateNKEClusterRequest{}
	changed := false

	if d.HasChange("name") {
		req.Name = d.Get("name").(string)
		changed = true
	}
	if d.HasChange("version") {
		req.Version = d.Get("version").(string)
		changed = true
	}
	if d.HasChange("plan") {
		pkgID, pkgDiag := getPackageID(d.Get("plan").(string), v2)
		if pkgDiag != nil {
			return diag.Diagnostics{*pkgDiag}
		}
		req.Billing = &gona.NKEUpdateBilling{PackageID: pkgID}
		changed = true
	}
	if d.HasChange("pool_id") {
		if v := d.Get("pool_id").(int); v != 0 {
			req.Pool = &gona.NKEPool{ID: v}
		}
		changed = true
	}
	if d.HasChange("minimum_nodes") || d.HasChange("maximum_nodes") {
		minNodes := d.Get("minimum_nodes").(int)
		nodes := &gona.NKEUpdateNodes{Minimum: minNodes}
		if v := d.Get("maximum_nodes").(int); v != 0 {
			nodes.Maximum = &v
		}
		req.Nodes = nodes
		changed = true
	}
	if d.HasChange("do_autoscaling") {
		v := d.Get("do_autoscaling").(bool)
		req.DoAutoscaling = &v
		changed = true
	}
	if d.HasChange("tag_ids") {
		for _, tagID := range d.Get("tag_ids").(*schema.Set).List() {
			req.Tags = append(req.Tags, gona.NKEClusterTagInput{TagID: tagID.(int)})
		}
		changed = true
	}

	if changed {
		if err := c.UpdateNKECluster(id, req); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceNKEClusterRead(ctx, d, m)
}

func resourceNKEClusterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting NKE cluster %d", id)

	if err := c.DeleteNKECluster(id); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NKE cluster %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

