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

func resourceStorageObjectStore() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStorageObjectStoreCreate,
		ReadContext:   resourceStorageObjectStoreRead,
		UpdateContext: resourceStorageObjectStoreUpdate,
		DeleteContext: resourceStorageObjectStoreDelete,
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
				Description: "The display label for the object store",
			},
			"location": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"location", "location_id"},
				DiffSuppressFunc: suppressStorageLocationDiff,
				Description:      "Location name where the object store will be created",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The location ID where the object store will be created",
			},
			"capacity": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Storage capacity in GB (1-1000)",
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
			// Computed
			"object_store_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the object store",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the object store is ready for use",
			},
			"assigned_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the object store was assigned",
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
			// S3 Credentials
			"endpoints": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "S3 endpoint URLs",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "S3 access key",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "S3 secret key",
			},
			"user_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "S3 user key",
			},
		},
	}
}

func resourceStorageObjectStoreCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	v2 := m.(*ProviderClients).V2

	locationID, locDiag := getStorageLocationID(d, c, v2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	req := &gona.CreateStorageObjectStoreRequest{
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

	storeID, err := c.CreateStorageObjectStore(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(storeID))
	log.Printf("[INFO] Storage object store created with ID: %d", storeID)

	if err := c.WaitForStorageObjectStoreReady(storeID); err != nil {
		return diag.Errorf("storage object store %d created but failed to become ready: %s", storeID, err)
	}

	return resourceStorageObjectStoreRead(ctx, d, m)
}

func resourceStorageObjectStoreRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	store, err := c.GetStorageObjectStore(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage object store %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("object_store_id", id, d, &diags)
	setValue("label", store.Metadata.Label, d, &diags)
	setValue("ready", store.Metadata.Ready, d, &diags)
	setValue("assigned_on", store.Metadata.AssignedOn, d, &diags)
	setValue("location_id", store.Metadata.Location.ID, d, &diags)
	if current := d.Get("location").(string); current == "" {
		setValue("location", store.Metadata.Location.Name, d, &diags)
	}
	setValue("location_name", store.Metadata.Location.Name, d, &diags)
	if store.Metadata.Capacity.RequestedGB != nil {
		setValue("capacity", *store.Metadata.Capacity.RequestedGB, d, &diags)
	}
	setValue("total_capacity_gb", store.Metadata.Capacity.TotalGB, d, &diags)
	setValue("auto_scaling", store.Metadata.Capacity.AutoScaling, d, &diags)
	setCredentialValue("endpoints", store.Credentials.Endpoints, d, &diags)
	setCredentialValue("access_key", store.Credentials.AccessKey, d, &diags)
	setCredentialValue("secret_key", store.Credentials.SecretKey, d, &diags)
	setCredentialValue("user_key", store.Credentials.UserKey, d, &diags)

	return diags
}

func resourceStorageObjectStoreUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateStorageObjectStoreRequest{}
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
		if err := c.UpdateStorageObjectStore(id, req); err != nil {
			return diag.FromErr(err)
		}
		// An update is not instant. A capacity change is accepted with a 200 and applied
		// over about twenty seconds, so reading straight away returns the OLD value and
		// writes it into state: apply reports success while state disagrees with the API
		// until some later refresh. PATCH capacity 2 can answer 200 with totalGB
		// still 1, then report 2 twenty seconds later.
		//
		// Same not-ready window that delays credentials and refuses an early delete.
		if err := c.WaitForStorageObjectStoreReady(id); err != nil {
			return diag.Errorf("storage object store %d updated but failed to become ready: %s", id, err)
		}
		// ready is not enough: capacity lands after it. Wait for the value asked for.
		if d.HasChange("capacity") {
			want := d.Get("capacity").(int)
			if err := waitForStorageCapacity(fmt.Sprintf("storage object store %d", id), want, func() (int, error) {
				obj, err := c.GetStorageObjectStore(id)
				if err != nil {
					return 0, err
				}
				return obj.Metadata.Capacity.TotalGB, nil
			}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceStorageObjectStoreRead(ctx, d, m)
}

func resourceStorageObjectStoreDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting storage object store %d", id)

	if err := deleteStorageWithRetry("object store "+d.Id(), func() error { return c.DeleteStorageObjectStore(id) }); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage object store %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
