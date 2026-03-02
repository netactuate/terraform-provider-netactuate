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

func resourceStorageBlockNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStorageBlockNamespaceCreate,
		ReadContext:   resourceStorageBlockNamespaceRead,
		UpdateContext: resourceStorageBlockNamespaceUpdate,
		DeleteContext: resourceStorageBlockNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The display label for the block namespace",
			},
			"location": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "Location name where the block namespace will be created",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The location ID where the block namespace will be created",
			},
			"capacity": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Storage capacity in GB",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Get("enable_auto_scaling").(bool)
				},
			},
			"enable_auto_scaling": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Auto-scale storage to match peak usage",
			},
			"block_namespace_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the block namespace",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the block namespace is ready for use",
			},
			"assigned_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the block namespace was assigned",
			},
			"location_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the location",
			},
			"total_capacity_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total provisioned capacity in GB",
			},
			"auto_scaling": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether auto-scaling is currently enabled",
			},
			"endpoints": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Block storage endpoint URLs",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"storage_pool": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The storage pool name",
			},
			"storage_namespace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The storage namespace",
			},
			"storage_cluster_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The storage cluster ID",
			},
		},
	}
}

func resourceStorageBlockNamespaceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	locationID, locDiag := getStorageLocationID(d, c)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	req := &gona.CreateStorageBlockNamespaceRequest{
		LocationID: locationID,
		Label:      d.Get("label").(string),
	}

	if v, ok := d.GetOk("capacity"); ok {
		req.Capacity = v.(int)
	}
	if v := d.Get("enable_auto_scaling").(bool); v {
		req.EnableAutoScaling = &v
	}

	nsID, err := c.CreateStorageBlockNamespace(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(nsID))
	log.Printf("[INFO] Storage block namespace created with ID: %d", nsID)

	if err := c.WaitForStorageBlockNamespaceReady(nsID); err != nil {
		return diag.Errorf("storage block namespace %d created but failed to become ready: %s", nsID, err)
	}

	return resourceStorageBlockNamespaceRead(ctx, d, m)
}

func resourceStorageBlockNamespaceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := c.GetStorageBlockNamespace(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage block namespace %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("block_namespace_id", ns.Metadata.BlockNamespaceID, d, &diags)
	setValue("label", ns.Metadata.Label, d, &diags)
	setValue("ready", ns.Metadata.Ready, d, &diags)
	setValue("assigned_on", ns.Metadata.AssignedOn, d, &diags)
	setValue("location_id", ns.Metadata.Location.ID, d, &diags)
	setValue("location", ns.Metadata.Location.Name, d, &diags)
	setValue("location_name", ns.Metadata.Location.Name, d, &diags)
	if ns.Metadata.Capacity.RequestedGB != nil {
		setValue("capacity", *ns.Metadata.Capacity.RequestedGB, d, &diags)
	}
	setValue("total_capacity_gb", ns.Metadata.Capacity.TotalGB, d, &diags)
	setValue("auto_scaling", ns.Metadata.Capacity.AutoScaling, d, &diags)
	setValue("endpoints", ns.Credentials.Endpoints, d, &diags)
	setValue("storage_pool", ns.Credentials.Pool, d, &diags)
	setValue("storage_namespace", ns.Credentials.Namespace, d, &diags)
	setValue("storage_cluster_id", ns.Credentials.ClusterID, d, &diags)

	return diags
}

func resourceStorageBlockNamespaceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateStorageBlockNamespaceRequest{}
	changed := false

	if d.HasChange("label") {
		req.Label = d.Get("label").(string)
		changed = true
	}
	if d.HasChange("capacity") {
		newCap := d.Get("capacity").(int)
		currentTotal := d.Get("total_capacity_gb").(int)
		if newCap > currentTotal {
			req.Capacity = newCap
			changed = true
		}
	}
	if d.HasChange("enable_auto_scaling") {
		v := d.Get("enable_auto_scaling").(bool)
		req.EnableAutoScaling = &v
		changed = true
	}

	if changed {
		if err := c.UpdateStorageBlockNamespace(id, req); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceStorageBlockNamespaceRead(ctx, d, m)
}

func resourceStorageBlockNamespaceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting storage block namespace %d", id)

	if err := c.DeleteStorageBlockNamespace(id); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage block namespace %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
