package netactuate

import (
	"context"
	"fmt"
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
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"location", "location_id"},
				DiffSuppressFunc: suppressStorageLocationDiff,
				Description:      "Location name where the block namespace will be created",
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
			"user_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The Ceph user key for namespace access",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The Ceph secret key for namespace access",
			},
		},
	}
}

func resourceStorageBlockNamespaceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	v2 := m.(*ProviderClients).V2

	locationID, locDiag := getStorageLocationID(d, c, v2)
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
	// The schema carries an explicit default, so false is a chosen value rather than an absent
	// one. Sending it only when true leaves the platform to apply its own default and puts an
	// unsettleable change in every later plan.
	autoScaling := d.Get("enable_auto_scaling").(bool)
	req.EnableAutoScaling = &autoScaling

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

	setValue("block_namespace_id", id, d, &diags)
	setValue("label", ns.Metadata.Label, d, &diags)
	setValue("ready", ns.Metadata.Ready, d, &diags)
	setValue("assigned_on", ns.Metadata.AssignedOn, d, &diags)
	setValue("location_id", ns.Metadata.Location.ID, d, &diags)
	if current := d.Get("location").(string); current == "" {
		setValue("location", ns.Metadata.Location.Name, d, &diags)
	}
	setValue("location_name", ns.Metadata.Location.Name, d, &diags)
	if ns.Metadata.Capacity.RequestedGB != nil {
		setValue("capacity", *ns.Metadata.Capacity.RequestedGB, d, &diags)
	}
	setValue("total_capacity_gb", ns.Metadata.Capacity.TotalGB, d, &diags)
	setValue("auto_scaling", ns.Metadata.Capacity.AutoScaling, d, &diags)
	setCredentialValue("endpoints", ns.Credentials.Endpoints, d, &diags)
	setValue("storage_pool", ns.Credentials.Pool, d, &diags)
	setValue("storage_namespace", ns.Credentials.Namespace, d, &diags)
	setValue("storage_cluster_id", ns.Credentials.ClusterID, d, &diags)
	setCredentialValue("user_key", ns.Credentials.UserKey, d, &diags)
	setCredentialValue("secret_key", ns.Credentials.SecretKey, d, &diags)

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
		// An update is not instant. A capacity change is accepted with a 200 and applied
		// over about twenty seconds, so reading straight away returns the OLD value and
		// writes it into state: apply reports success while state disagrees with the API
		// until some later refresh. PATCH capacity 2 can answer 200 with totalGB
		// still 1, then report 2 twenty seconds later.
		//
		// Same not-ready window that delays credentials and refuses an early delete.
		if err := c.WaitForStorageBlockNamespaceReady(id); err != nil {
			return diag.Errorf("storage block namespace %d updated but failed to become ready: %s", id, err)
		}
		// ready is not enough: capacity lands after it. Wait for the value asked for.
		if d.HasChange("capacity") {
			want := d.Get("capacity").(int)
			if err := waitForStorageCapacity(fmt.Sprintf("storage block namespace %d", id), want, func() (int, error) {
				obj, err := c.GetStorageBlockNamespace(id)
				if err != nil {
					return 0, err
				}
				return obj.Metadata.Capacity.TotalGB, nil
			}); err != nil {
				return diag.FromErr(err)
			}
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

	if err := deleteStorageWithRetry("block namespace "+d.Id(), func() error { return c.DeleteStorageBlockNamespace(id) }); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage block namespace %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
