package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStorageTypes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageTypesRead,
		Schema: map[string]*schema.Schema{
			"types": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Storage service types available to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"type":        {Type: schema.TypeString, Computed: true, Description: "Storage type code."},
					"name":        {Type: schema.TypeString, Computed: true, Description: "Storage type name."},
					"description": {Type: schema.TypeString, Computed: true, Description: "Storage type description."},
					"raw_json":    rawJSONDataSourceSchema("Full storage type response as compact JSON."),
				}},
			},
		},
	}
}

func dataSourceStorageTypesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	types, err := m.(*ProviderClients).V3.ListStorageTypes()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(types))
	for i, storageType := range types {
		raw, err := compactSurfaceJSON(storageType.Raw)
		if err != nil {
			return diag.FromErr(err)
		}
		out[i] = map[string]interface{}{"type": storageType.Type, "name": storageType.Name, "description": storageType.Description, "raw_json": raw}
	}
	var diags diag.Diagnostics
	setValue("types", out, d, &diags)
	d.SetId("storage-types")
	return diags
}

func dataSourceStorageBlockNamespaces() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageBlockNamespacesRead,
		Schema: map[string]*schema.Schema{
			"block_namespaces": {Type: schema.TypeList, Computed: true, Description: "Storage block namespaces visible to the account.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"block_namespace_id": {Type: schema.TypeInt, Computed: true, Description: "Block namespace ID."},
				"label":              {Type: schema.TypeString, Computed: true, Description: "Block namespace label."},
				"ready":              {Type: schema.TypeBool, Computed: true, Description: "Whether the block namespace is ready."},
				"assigned_on":        {Type: schema.TypeString, Computed: true, Description: "Assignment timestamp."},
			}}},
		},
	}
}

func dataSourceStorageBlockNamespacesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	namespaces, err := m.(*ProviderClients).V3.ListStorageBlockNamespaces()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(namespaces))
	for i, namespace := range namespaces {
		out[i] = map[string]interface{}{"block_namespace_id": namespace.Metadata.BlockNamespaceID, "label": namespace.Metadata.Label, "ready": namespace.Metadata.Ready, "assigned_on": namespace.Metadata.AssignedOn}
	}
	var diags diag.Diagnostics
	setValue("block_namespaces", out, d, &diags)
	d.SetId("storage-block-namespaces")
	return diags
}

func dataSourceStorageBuckets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageBucketsRead,
		Schema: map[string]*schema.Schema{
			"buckets": {Type: schema.TypeList, Computed: true, Description: "Storage buckets visible to the account.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"bucket_id":   {Type: schema.TypeInt, Computed: true, Description: "Storage bucket ID."},
				"label":       {Type: schema.TypeString, Computed: true, Description: "Storage bucket label."},
				"ready":       {Type: schema.TypeBool, Computed: true, Description: "Whether the bucket is ready."},
				"private":     {Type: schema.TypeBool, Computed: true, Description: "Whether the bucket is private."},
				"assigned_on": {Type: schema.TypeString, Computed: true, Description: "Assignment timestamp."},
			}}},
		},
	}
}

func dataSourceStorageBucketsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	buckets, err := m.(*ProviderClients).V3.ListStorageBuckets()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(buckets))
	for i, bucket := range buckets {
		out[i] = map[string]interface{}{"bucket_id": bucket.Metadata.BucketID, "label": bucket.Metadata.Label, "ready": bucket.Metadata.Ready, "private": bucket.Metadata.Private, "assigned_on": bucket.Metadata.AssignedOn}
	}
	var diags diag.Diagnostics
	setValue("buckets", out, d, &diags)
	d.SetId("storage-buckets")
	return diags
}

func dataSourceStorageObjectStores() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceStorageObjectStoresRead,
		Schema: map[string]*schema.Schema{
			"object_stores": {Type: schema.TypeList, Computed: true, Description: "Storage object stores visible to the account.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"object_store_id": {Type: schema.TypeInt, Computed: true, Description: "Object store ID."},
				"label":           {Type: schema.TypeString, Computed: true, Description: "Object store label."},
				"ready":           {Type: schema.TypeBool, Computed: true, Description: "Whether the object store is ready."},
				"assigned_on":     {Type: schema.TypeString, Computed: true, Description: "Assignment timestamp."},
			}}},
		},
	}
}

func dataSourceStorageObjectStoresRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	stores, err := m.(*ProviderClients).V3.ListStorageObjectStores()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(stores))
	for i, store := range stores {
		out[i] = map[string]interface{}{"object_store_id": store.Metadata.ObjectStoreID, "label": store.Metadata.Label, "ready": store.Metadata.Ready, "assigned_on": store.Metadata.AssignedOn}
	}
	var diags diag.Diagnostics
	setValue("object_stores", out, d, &diags)
	d.SetId("storage-object-stores")
	return diags
}

func dataSourceVPCs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCsRead,
		Schema: map[string]*schema.Schema{
			"vpcs": {Type: schema.TypeList, Computed: true, Description: "VPCs visible to the account.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"vpc_id":      {Type: schema.TypeInt, Computed: true, Description: "VPC ID."},
				"label":       {Type: schema.TypeString, Computed: true, Description: "VPC label."},
				"description": {Type: schema.TypeString, Computed: true, Description: "VPC description."},
				"status":      {Type: schema.TypeString, Computed: true, Description: "VPC status."},
				"location_id": {Type: schema.TypeInt, Computed: true, Description: "VPC location ID."},
				"location":    {Type: schema.TypeString, Computed: true, Description: "VPC location name."},
			}}},
		},
	}
}

func dataSourceVPCsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcs, err := m.(*ProviderClients).V3.ListVPCs()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(vpcs))
	for i, vpc := range vpcs {
		out[i] = map[string]interface{}{"vpc_id": vpc.VPCID, "label": vpc.Metadata.Label, "description": vpc.Metadata.Description, "status": vpc.Metadata.Status, "location_id": vpc.Location.ID, "location": vpc.Location.Name}
	}
	var diags diag.Diagnostics
	setValue("vpcs", out, d, &diags)
	d.SetId("vpcs")
	return diags
}

func dataSourceVPCLocations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCLocationsRead,
		Schema: map[string]*schema.Schema{
			"locations": {Type: schema.TypeList, Computed: true, Description: "Locations where VPCs can be created.", Elem: &schema.Resource{Schema: v3LocationDataSourceSchema()}},
		},
	}
}

func dataSourceVPCLocationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	locations, err := m.(*ProviderClients).V3.ListVPCLocations()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(locations))
	for i, location := range locations {
		out[i] = map[string]interface{}{"id": location.ID, "name": location.Name, "flag": location.Flag}
	}
	var diags diag.Diagnostics
	setValue("locations", out, d, &diags)
	d.SetId("vpc-locations")
	return diags
}

func dataSourceVPCIPReservations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCIPReservationsRead,
		Schema: map[string]*schema.Schema{
			"vpc_id":     {Type: schema.TypeInt, Required: true, Description: "VPC ID."},
			"gateways":   rawJSONDataSourceSchema("Gateway IP reservations as compact JSON."),
			"interfaces": rawJSONDataSourceSchema("Interface IP reservations as compact JSON."),
			"vms":        rawJSONDataSourceSchema("VM IP reservations as compact JSON."),
		},
	}
}

func dataSourceVPCIPReservationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID := d.Get("vpc_id").(int)
	reservations, err := m.(*ProviderClients).V3.GetVPCIPReservations(vpcID)
	if err != nil {
		return diag.FromErr(err)
	}
	gateways, err := compactSurfaceJSON(reservations.Gateways)
	if err != nil {
		return diag.FromErr(err)
	}
	interfaces, err := compactSurfaceJSON(reservations.Interfaces)
	if err != nil {
		return diag.FromErr(err)
	}
	vms, err := compactSurfaceJSON(reservations.VMs)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("gateways", gateways, d, &diags)
	setValue("interfaces", interfaces, d, &diags)
	setValue("vms", vms, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-ip-reservations", vpcID))
	return diags
}

func dataSourceVPCSSHSettings() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCSSHSettingsRead,
		Schema: map[string]*schema.Schema{
			"vpc_id":       {Type: schema.TypeInt, Required: true, Description: "VPC ID."},
			"enabled":      {Type: schema.TypeBool, Computed: true, Description: "Whether the VPC bastion service is enabled."},
			"port":         {Type: schema.TypeInt, Computed: true, Description: "Bastion SSH port, or 0 when absent."},
			"bastion_ipv4": {Type: schema.TypeString, Computed: true, Description: "Bastion IPv4 address."},
			"bastion_ipv6": {Type: schema.TypeString, Computed: true, Description: "Bastion IPv6 address."},
		},
	}
}

func dataSourceVPCSSHSettingsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID := d.Get("vpc_id").(int)
	settings, err := m.(*ProviderClients).V3.GetVPCSSHSettings(vpcID)
	if err != nil {
		return diag.FromErr(err)
	}
	port := 0
	if settings.Port != nil {
		port = *settings.Port
	}
	ipv4 := ""
	ipv6 := ""
	if settings.Bastion != nil {
		ipv4 = settings.Bastion.IPv4
		ipv6 = settings.Bastion.IPv6
	}
	var diags diag.Diagnostics
	setValue("enabled", settings.Enabled, d, &diags)
	setValue("port", port, d, &diags)
	setValue("bastion_ipv4", ipv4, d, &diags)
	setValue("bastion_ipv6", ipv6, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-ssh", vpcID))
	return diags
}
