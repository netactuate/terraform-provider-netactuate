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

func resourceStorageBlockVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStorageBlockVolumeCreate,
		ReadContext:   resourceStorageBlockVolumeRead,
		UpdateContext: resourceStorageBlockVolumeUpdate,
		DeleteContext: resourceStorageBlockVolumeDelete,
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
				Description: "The display label for the block volume",
			},
			"location": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "Location name where the block volume will be created",
			},
			"location_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The location ID where the block volume will be created",
			},
			"capacity": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Storage capacity in GB (1-1000)",
			},
			// Computed
			"block_volume_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the block volume",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the block volume is ready for use",
			},
			"assigned_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the block volume was assigned",
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
			// Connection Creds
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
			"image_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The block storage image name",
			},
		},
	}
}

func resourceStorageBlockVolumeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	locationID, locDiag := getStorageLocationID(d, c)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	req := &gona.CreateStorageBlockVolumeRequest{
		LocationID: locationID,
		Label:      d.Get("label").(string),
	}

	if v, ok := d.GetOk("capacity"); ok {
		req.Capacity = v.(int)
	}

	volID, err := c.CreateStorageBlockVolume(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(volID))
	log.Printf("[INFO] Storage block volume created with ID: %d", volID)

	if err := c.WaitForStorageBlockVolumeReady(volID); err != nil {
		return diag.Errorf("storage block volume %d created but failed to become ready: %s", volID, err)
	}

	return resourceStorageBlockVolumeRead(ctx, d, m)
}

func resourceStorageBlockVolumeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	vol, err := c.GetStorageBlockVolume(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage block volume %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("block_volume_id", id, d, &diags)
	setValue("label", vol.Metadata.Label, d, &diags)
	setValue("ready", vol.Metadata.Ready, d, &diags)
	setValue("assigned_on", vol.Metadata.AssignedOn, d, &diags)
	setValue("location_id", vol.Metadata.Location.ID, d, &diags)
	setValue("location", vol.Metadata.Location.Name, d, &diags)
	setValue("location_name", vol.Metadata.Location.Name, d, &diags)
	if vol.Metadata.Capacity.RequestedGB != nil {
		setValue("capacity", *vol.Metadata.Capacity.RequestedGB, d, &diags)
	}
	setValue("total_capacity_gb", vol.Metadata.Capacity.TotalGB, d, &diags)
	setValue("endpoints", vol.Credentials.Endpoints, d, &diags)
	setValue("storage_pool", vol.Credentials.Pool, d, &diags)
	setValue("storage_namespace", vol.Credentials.Namespace, d, &diags)
	setValue("storage_cluster_id", vol.Credentials.ClusterID, d, &diags)
	setValue("image_name", vol.Credentials.ImageName, d, &diags)

	return diags
}

func resourceStorageBlockVolumeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateStorageBlockVolumeRequest{}
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

	if changed {
		if err := c.UpdateStorageBlockVolume(id, req); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceStorageBlockVolumeRead(ctx, d, m)
}

func resourceStorageBlockVolumeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting storage block volume %d", id)

	if err := c.DeleteStorageBlockVolume(id); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Storage block volume %d already deleted", id)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
