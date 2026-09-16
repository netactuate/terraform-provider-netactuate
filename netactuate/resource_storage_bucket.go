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

func resourceStorageBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStorageBucketCreate,
		ReadContext:   resourceStorageBucketRead,
		UpdateContext: resourceStorageBucketUpdate,
		DeleteContext: resourceStorageBucketDelete,
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
				Description: "The display label for the storage bucket",
			},
			"location": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ExactlyOneOf:     []string{"location", "location_id"},
				DiffSuppressFunc: suppressStorageLocationDiff,
				Description:      "Location name where the bucket will be created",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The location ID where the bucket will be created",
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
			"private": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the bucket has no public read access",
			},
			// Computed
			"bucket_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the bucket",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the bucket is ready for use",
			},
			"assigned_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the bucket was assigned",
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
			// Connection Cred
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

func resourceStorageBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	v2 := m.(*ProviderClients).V2

	locationID, locDiag := getStorageLocationID(d, c, v2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	req := &gona.CreateStorageBucketRequest{
		LocationID: locationID,
		Label:      d.Get("label").(string),
	}

	if v, ok := d.GetOk("capacity"); ok {
		req.Capacity = v.(int)
	}
	// Both carry an explicit schema default, so false is a chosen value and must be sent.
	// Omitting it lets the platform apply its own default and the resource never converges.
	autoScaling := d.Get("enable_auto_scaling").(bool)
	req.EnableAutoScaling = &autoScaling
	private := d.Get("private").(bool)
	req.Private = &private

	bucketID, err := c.CreateStorageBucket(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(bucketID))
	log.Printf("[INFO] Storage bucket created with ID: %d", bucketID)

	if err := c.WaitForStorageBucketReady(bucketID); err != nil {
		return diag.Errorf("storage bucket %d created but failed to become ready: %s", bucketID, err)
	}

	return resourceStorageBucketRead(ctx, d, m)
}

func resourceStorageBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	bucket, err := c.GetStorageBucket(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage bucket %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("bucket_id", id, d, &diags)
	setValue("label", bucket.Metadata.Label, d, &diags)
	setValue("ready", bucket.Metadata.Ready, d, &diags)
	setValue("private", bucket.Metadata.Private, d, &diags)
	setValue("assigned_on", bucket.Metadata.AssignedOn, d, &diags)
	setValue("location_id", bucket.Metadata.Location.ID, d, &diags)
	if current := d.Get("location").(string); current == "" {
		setValue("location", bucket.Metadata.Location.Name, d, &diags)
	}
	setValue("location_name", bucket.Metadata.Location.Name, d, &diags)
	if bucket.Metadata.Capacity.RequestedGB != nil {
		setValue("capacity", *bucket.Metadata.Capacity.RequestedGB, d, &diags)
	}
	setValue("total_capacity_gb", bucket.Metadata.Capacity.TotalGB, d, &diags)
	setValue("auto_scaling", bucket.Metadata.Capacity.AutoScaling, d, &diags)
	setCredentialValue("endpoints", bucket.Credentials.Endpoints, d, &diags)
	setCredentialValue("access_key", bucket.Credentials.AccessKey, d, &diags)
	setCredentialValue("secret_key", bucket.Credentials.SecretKey, d, &diags)
	setCredentialValue("user_key", bucket.Credentials.UserKey, d, &diags)

	return diags
}

func resourceStorageBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateStorageBucketRequest{}
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
	if d.HasChange("private") {
		v := d.Get("private").(bool)
		req.Private = &v
		changed = true
	}

	if changed {
		if err := c.UpdateStorageBucket(id, req); err != nil {
			return diag.FromErr(err)
		}
		// An update is not instant. A capacity change is accepted with a 200 and applied
		// over about twenty seconds, so reading straight away returns the OLD value and
		// writes it into state: apply reports success while state disagrees with the API
		// until some later refresh. PATCH capacity 2 can answer 200 with totalGB
		// still 1, then report 2 twenty seconds later.
		//
		// Same not-ready window that delays credentials and refuses an early delete.
		if err := c.WaitForStorageBucketReady(id); err != nil {
			return diag.Errorf("storage bucket %d updated but failed to become ready: %s", id, err)
		}
		// ready is not enough: capacity lands after it. Wait for the value asked for.
		if d.HasChange("capacity") {
			want := d.Get("capacity").(int)
			if err := waitForStorageCapacity(fmt.Sprintf("storage bucket %d", id), want, func() (int, error) {
				obj, err := c.GetStorageBucket(id)
				if err != nil {
					return 0, err
				}
				return obj.Metadata.Capacity.TotalGB, nil
			}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceStorageBucketRead(ctx, d, m)
}

func resourceStorageBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting storage bucket %d", id)

	if err := deleteStorageWithRetry("bucket "+d.Id(), func() error { return c.DeleteStorageBucket(id) }); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage bucket %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
