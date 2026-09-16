package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceStorageBlockVolumes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageBlockVolumesRead,
		Schema: map[string]*schema.Schema{
			"block_volumes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of storage block volumes visible to the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"credentials": {
							Type:        schema.TypeList,
							Computed:    true,
							Sensitive:   true,
							Description: "Storage block volume credentials",
							Elem: &schema.Resource{
								Schema: storageBlockCredentialsDataSourceSchema(),
							},
						},
						"metadata": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Storage block volume metadata",
							Elem: &schema.Resource{
								Schema: storageBlockVolumeMetadataDataSourceSchema(),
							},
						},
					},
				},
			},
		},
	}
}

func storageBlockCredentialsDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"endpoints": {
			Type:        schema.TypeList,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume endpoints",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"user_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume user key",
		},
		"secret_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume secret key",
		},
		"pool": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume pool",
		},
		"namespace": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume namespace",
		},
		"cluster_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume cluster ID",
		},
		"image_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Storage block volume image name",
		},
	}
}

func storageBlockVolumeMetadataDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"block_volume_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Storage block volume ID",
		},
		"label": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Storage block volume label",
		},
		"ready": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the storage block volume is ready",
		},
		"assigned_on": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Storage block volume assignment timestamp",
		},
		"location": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Storage block volume location",
			Elem: &schema.Resource{
				Schema: v3LocationDataSourceSchema(),
			},
		},
		"capacity": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Storage block volume capacity",
			Elem: &schema.Resource{
				Schema: v3CapacityDataSourceSchema(),
			},
		},
		"hardware_class": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Storage block volume hardware class",
			Elem: &schema.Resource{
				Schema: storageHardwareClassDataSourceSchema(),
			},
		},
	}
}

func v3LocationDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Location ID",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Location name",
		},
		"flag": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Location flag value",
		},
	}
}

func v3CapacityDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"auto_scaling": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether capacity auto scaling is enabled",
		},
		"requested_gb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Requested capacity in GB, or 0 when absent",
		},
		"total_gb": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Total capacity in GB",
		},
	}
}

func storageHardwareClassDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Storage hardware class ID",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Storage hardware class name",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Storage hardware class description",
		},
	}
}

func dataSourceStorageBlockVolumesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	volumes, err := c.ListStorageBlockVolumes()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	volumeList := make([]map[string]interface{}, len(volumes))
	for i, volume := range volumes {
		volumeList[i] = flattenStorageBlockVolume(volume)
	}

	setValue("block_volumes", volumeList, d, &diags)
	d.SetId("storage-block-volumes")

	return diags
}

func flattenStorageBlockVolume(volume gona.StorageBlockVolume) map[string]interface{} {
	return map[string]interface{}{
		"credentials": flattenStorageBlockCredentials(volume.Credentials),
		"metadata":    flattenStorageBlockVolumeMetadata(volume.Metadata),
	}
}

func flattenStorageBlockCredentials(credentials gona.StorageBlockCredentials) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"endpoints":  credentials.Endpoints,
			"user_key":   credentials.UserKey,
			"secret_key": credentials.SecretKey,
			"pool":       credentials.Pool,
			"namespace":  credentials.Namespace,
			"cluster_id": credentials.ClusterID,
			"image_name": credentials.ImageName,
		},
	}
}

func flattenStorageBlockVolumeMetadata(metadata gona.StorageBlockVolumeMetadata) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"block_volume_id": metadata.BlockVolumeID,
			"label":           metadata.Label,
			"ready":           metadata.Ready,
			"assigned_on":     metadata.AssignedOn,
			"location":        flattenV3Location(metadata.Location),
			"capacity":        flattenV3Capacity(metadata.Capacity),
			"hardware_class":  flattenStorageHardwareClass(metadata.HardwareClass),
		},
	}
}

func flattenV3Location(location gona.V3Location) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":   location.ID,
			"name": location.Name,
			"flag": location.Flag,
		},
	}
}

func flattenV3Capacity(capacity gona.V3Capacity) []map[string]interface{} {
	requestedGB := 0
	if capacity.RequestedGB != nil {
		requestedGB = *capacity.RequestedGB
	}

	return []map[string]interface{}{
		{
			"auto_scaling": capacity.AutoScaling,
			"requested_gb": requestedGB,
			"total_gb":     capacity.TotalGB,
		},
	}
}

func flattenStorageHardwareClass(hardware gona.StorageHardwareClass) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":          hardware.ID,
			"name":        hardware.Name,
			"description": hardware.Description,
		},
	}
}
