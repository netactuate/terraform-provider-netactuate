package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
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
		CustomizeDiff: customdiff.All(
			func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
				if !d.HasChange("version") && d.Id() != "" {
					return nil
				}
				version := d.Get("version").(string)
				c := m.(*ProviderClients).V3
				if diags := validateNKEVersion(version, c); diags.HasError() {
					return fmt.Errorf("%s", diags[0].Summary)
				}
				return nil
			},
		),
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
				Description: "The Kubernetes version. Use data.netactuate_nke_versions to list available versions.",
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
				Computed:    true,
				Description: "Whether the control plane has high availability (true when replicas > 1)",
			},
			"kubernetes_dashboard": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Install the Kubernetes dashboard addon (requires recreation)",
			},
			"do_dual_stack": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Enable dual-stack IPv4+IPv6 support (can only be set at creation, requires recreation to change)",
			},
			"vpc_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "VPC ID for worker node networking. Requires an applied outbound SNAT rule before cluster creation.",
			},
			"pod_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Pod network CIDR for the cluster",
			},
			"service_cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				// The platform accepts a /16 ONLY, and says so at apply time:
				//   400 networking.serviceCidr: "The service CIDR must be a /16 (the MVP
				//   supports /16 only)"
				// Checked at plan instead, so a customer learns before an NKE cluster
				// build is attempted rather than after their plan was approved. Same
				// reasoning as the VPC label length check.
				ValidateDiagFunc: validateServiceCIDRIsSlash16,
				Description:      "Service network CIDR for the cluster. Must be a /16: the platform supports no other prefix length.",
			},
			// Location: accept name or ID
			"location": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"location", "location_id"},
				DiffSuppressFunc: suppressLocationDiff,
				Description:      "Location name where the cluster will be deployed (e.g. \"DEVRDU - Raleigh, NC\")",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "Location ID where the cluster will be deployed",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress location_id drift when location (name) is set in config
					_, hasLocation := d.GetOk("location")
					return hasLocation && new == "0"
				},
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
			"contract_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "NKE billing contract ID. This is distinct from the server resource's package_billing_contract_id. Must be a usage-type contract.",
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

func validateNKEVersion(version string, c *gona.V3Client) diag.Diagnostics {
	versions, err := c.ListNKEVersions()
	if err != nil {
		return diag.Errorf("failed to fetch available NKE versions: %s", err)
	}
	for _, v := range versions {
		if v == version {
			return nil
		}
	}
	return diag.Errorf("invalid kubernetes version %q, available versions: %v", version, versions)
}

func resourceNKEClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v2 := m.(*ProviderClients).V2
	c := m.(*ProviderClients).V3

	if diags := validateNKEVersion(d.Get("version").(string), c); diags.HasError() {
		return diags
	}

	locationID, locDiag := getLocationID(d, v2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	packageID, pkgDiag := getPackageID(d.Get("plan").(string), v2)
	if pkgDiag != nil {
		return diag.Diagnostics{*pkgDiag}
	}

	contractID := d.Get("contract_id").(int)
	billing := gona.NKEBilling{
		PackageID:  packageID,
		LocationID: locationID,
		ContractID: &contractID,
	}

	req := &gona.CreateNKEClusterRequest{
		Name:          d.Get("name").(string),
		Version:       d.Get("version").(string),
		Replicas:      d.Get("replicas").(int),
		MinimumNodes:  d.Get("minimum_nodes").(int),
		MaximumNodes:  d.Get("maximum_nodes").(int),
		DoAutoscaling: d.Get("do_autoscaling").(bool),
		DoDualStack:   d.Get("do_dual_stack").(bool),
		Billing:       billing,
	}

	if d.Get("kubernetes_dashboard").(bool) {
		req.AddonsToInstall = &gona.NKEAddons{KubernetesDashboard: true}
	}

	networking := &gona.NKEClusterNetwork{}
	if v, ok := d.GetOk("vpc_id"); ok {
		networking.VpcID = v.(int)
	}
	if v, ok := d.GetOk("pod_cidr"); ok {
		networking.PodCIDR = v.(string)
	}
	if v, ok := d.GetOk("service_cidr"); ok {
		networking.ServiceCIDR = v.(string)
	}
	if networking.VpcID != 0 || networking.PodCIDR != "" || networking.ServiceCIDR != "" {
		req.Networking = networking
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

	// A Healthy cluster does not yet list its worker nodes, so a data source reading them in the
	// same apply would record an empty list for a cluster that is about to have several.
	if err := c.WaitForNKEWorkerNodes(clusterID, req.MinimumNodes); err != nil {
		return diag.Errorf("NKE cluster %d became healthy but its worker nodes did not register: %s", clusterID, err)
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
	setLocationPreserveFormat(cluster.Location.Name, d, &diags)
	setValue("plan", cluster.Package.Name, d, &diags)
	setValue("package_id", cluster.Package.ID, d, &diags)

	setValue("status", cluster.Status.Cluster, d, &diags)
	setValue("api_url", cluster.URLs.API, d, &diags)
	setValue("prometheus_url", cluster.URLs.Prometheus, d, &diags)
	setValue("kubernetes_dashboard_url", cluster.URLs.KubernetesDashboard, d, &diags)
	setValue("pod_network", cluster.Networks.Pod, d, &diags)
	setValue("service_network", cluster.Networks.Service, d, &diags)
	setValue("vpc_id", cluster.VpcID, d, &diags)
	setValue("pod_cidr", cluster.Networks.Pod, d, &diags)
	setValue("service_cidr", cluster.Networks.Service, d, &diags)

	setValue("high_availability", cluster.HasHighAvailability, d, &diags)
	setValue("kubernetes_dashboard", cluster.KubernetesDashboard.Requested, d, &diags)
	setValue("do_dual_stack", cluster.IsDualStack, d, &diags)
	if cluster.ContractID != 0 {
		setValue("contract_id", cluster.ContractID, d, &diags)
	}

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
		newVersion := d.Get("version").(string)
		if diags := validateNKEVersion(newVersion, c); diags.HasError() {
			return diags
		}
		req.Version = newVersion
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

// validateServiceCIDRIsSlash16 enforces the platform's only supported service CIDR shape.
//
// A /12 is rejected at apply with a clear message, but only after the plan has
// been approved and the cluster build attempted.
func validateServiceCIDRIsSlash16(v interface{}, p cty.Path) diag.Diagnostics {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	if !strings.HasSuffix(s, "/16") {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "service_cidr must be a /16",
			Detail:        fmt.Sprintf("got %q. The platform supports a /16 service CIDR only.", s),
			AttributePath: p,
		}}
	}
	return nil
}
