package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceDedicatedDevices() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedDevicesRead,
		Schema: map[string]*schema.Schema{
			"per_page":    {Type: schema.TypeInt, Optional: true, Description: "Maximum number of dedicated devices to return."},
			"nic":         {Type: schema.TypeString, Optional: true, Description: "NIC filter."},
			"cpu_type":    {Type: schema.TypeString, Optional: true, Description: "CPU type filter."},
			"gpu_type":    {Type: schema.TypeString, Optional: true, Description: "GPU type filter."},
			"disk_type":   {Type: schema.TypeString, Optional: true, Description: "Disk type filter."},
			"cores":       {Type: schema.TypeString, Optional: true, Description: "CPU core count filter."},
			"ram_mb":      {Type: schema.TypeString, Optional: true, Description: "RAM size filter in MB."},
			"disk_mib":    {Type: schema.TypeString, Optional: true, Description: "Disk size filter in MiB."},
			"dc_name":     {Type: schema.TypeString, Optional: true, Description: "Datacenter name filter."},
			"region_name": {Type: schema.TypeString, Optional: true, Description: "Region name filter."},
			"devices": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Dedicated devices matching the filters. Flexible API fields are exposed in raw_json.",
				Elem:        &schema.Resource{Schema: rawCatalogRowSchema("device_id")},
			},
		},
	}
}

func dataSourceDedicatedLocations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedLocationsRead,
		Schema: map[string]*schema.Schema{
			"locations": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Dedicated server locations available to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"short_name":      {Type: schema.TypeString, Computed: true, Description: "The location's short name, such as JNB."},
					"pub_description": {Type: schema.TypeString, Computed: true, Description: "Public description of the facility."},
					"location_id":     {Type: schema.TypeInt, Computed: true, Description: "Platform location ID."},
				}},
			},
		},
	}
}

func dataSourceDedicatedDeviceOSProfiles() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedDeviceOSProfilesRead,
		Schema: map[string]*schema.Schema{
			"device_id":  {Type: schema.TypeInt, Required: true, Description: "Dedicated device ID to read compatible OS profiles for."},
			"is_buyable": {Type: schema.TypeBool, Optional: true, Description: "Limit OS profiles by buyable status."},
			"os_profiles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Dedicated OS profiles compatible with the device.",
				Elem:        &schema.Resource{Schema: dedicatedOSSchema()},
			},
		},
	}
}

func dataSourceDedicatedPlans() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedPlansRead,
		Schema: map[string]*schema.Schema{
			"location_id": {Type: schema.TypeInt, Required: true, Description: "Location ID to read dedicated server plans for."},
			"plans": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Dedicated server plans for the location. Flexible API fields are exposed in raw_json.",
				Elem:        &schema.Resource{Schema: rawCatalogRowSchema("plan_id")},
			},
		},
	}
}

func dataSourceDedicatedServers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedServersRead,
		Schema: map[string]*schema.Schema{
			"servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Dedicated servers visible to the account.",
				Elem:        &schema.Resource{Schema: dedicatedServerReadSchema()},
			},
		},
	}
}

func dataSourceDedicatedDevicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	opts := gona.DedicatedDeviceFilterOptions{
		NIC:        d.Get("nic").(string),
		CPUType:    d.Get("cpu_type").(string),
		GPUType:    d.Get("gpu_type").(string),
		DiskType:   d.Get("disk_type").(string),
		Cores:      d.Get("cores").(string),
		RAMMB:      d.Get("ram_mb").(string),
		DiskMIB:    d.Get("disk_mib").(string),
		DCName:     d.Get("dc_name").(string),
		RegionName: d.Get("region_name").(string),
	}
	if v, ok := d.GetOk("per_page"); ok {
		perPage := v.(int)
		opts.PerPage = &perPage
	}

	devices, err := c.FilterDedicatedDevices(opts)
	if err != nil {
		return diag.FromErr(err)
	}
	rows, err := flattenRawMaps(devices, "device_id")
	if err != nil {
		return diag.FromErr(fmt.Errorf("flatten dedicated devices: %w", err))
	}

	var diags diag.Diagnostics
	setValue("devices", rows, d, &diags)
	d.SetId("dedicated-devices")
	return diags
}

func dataSourceDedicatedLocationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	locations, err := c.ListDedicatedLocations()
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(locations))
	for i, location := range locations {
		rows[i] = map[string]interface{}{
			"short_name":      location.ShortName,
			"pub_description": location.PubDescription,
			"location_id":     location.LocationID,
		}
	}

	var diags diag.Diagnostics
	setValue("locations", rows, d, &diags)
	d.SetId("dedicated-locations")
	return diags
}

func dataSourceDedicatedDeviceOSProfilesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	var isBuyable *bool
	if v, ok := d.GetOkExists("is_buyable"); ok {
		value := v.(bool)
		isBuyable = &value
	}
	profiles, err := c.ListDedicatedDeviceOSProfiles(d.Get("device_id").(int), isBuyable)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("os_profiles", flattenDedicatedOSProfiles(profiles), d, &diags)
	d.SetId(fmt.Sprintf("dedicated-device-os-profiles-%d", d.Get("device_id").(int)))
	return diags
}

func dataSourceDedicatedPlansRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	locationID := d.Get("location_id").(int)

	plans, err := c.ListDedicatedPlans(locationID)
	if err != nil {
		return diag.FromErr(err)
	}
	rows, err := flattenRawMaps(plans, "plan_id")
	if err != nil {
		return diag.FromErr(fmt.Errorf("flatten dedicated plans: %w", err))
	}

	var diags diag.Diagnostics
	setValue("plans", rows, d, &diags)
	d.SetId(fmt.Sprintf("dedicated-plans-%d", locationID))
	return diags
}

func dataSourceDedicatedServersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	servers, err := c.ListDedicatedServers()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("servers", flattenDedicatedServers(servers), d, &diags)
	d.SetId("dedicated-servers")
	return diags
}

func rawCatalogRowSchema(idKey string) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		idKey:      {Type: schema.TypeInt, Computed: true, Description: "API row ID, or 0 when the row does not carry one."},
		"name":     {Type: schema.TypeString, Computed: true, Description: "API row name, when present."},
		"raw_json": {Type: schema.TypeString, Computed: true, Description: "Compact JSON row returned by the API."},
	}
}

func dedicatedServerReadSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"dedicated_server_id": {Type: schema.TypeInt, Computed: true, Description: "Dedicated server ID from the API field named id."},
		"mbpkgid":             {Type: schema.TypeInt, Computed: true, Description: "Dedicated server package ID."},
		"hostname":            {Type: schema.TypeString, Computed: true, Description: "Dedicated server hostname."},
		"datacenter_id":       {Type: schema.TypeInt, Computed: true, Description: "Datacenter ID."},
		"location":            {Type: schema.TypeString, Computed: true, Description: "Location name."},
		"primary_ipv4":        {Type: schema.TypeString, Computed: true, Description: "Primary IPv4 address."},
		"primary_ipv6":        {Type: schema.TypeString, Computed: true, Description: "Primary IPv6 address, or empty when the API returns null."},
		"package_status":      {Type: schema.TypeString, Computed: true, Description: "Package status."},
		"canceling":           {Type: schema.TypeInt, Computed: true, Description: "Canceling flag as returned by the API."},
		"nps_installed":       {Type: schema.TypeInt, Computed: true, Description: "NPS installed flag as returned by the API."},
		"nps_os":              {Type: schema.TypeString, Computed: true, Description: "Installed NPS OS name."},
		"locked":              {Type: schema.TypeInt, Computed: true, Description: "Lock flag as returned by the API."},
		"locked_msg":          {Type: schema.TypeString, Computed: true, Description: "Lock message, or empty when the API returns null."},
		"ipmi_status":         {Type: schema.TypeInt, Computed: true, Description: "IPMI status as returned by the API."},
		"ipmi_pubip":          {Type: schema.TypeString, Computed: true, Description: "IPMI public IP address."},
		"ipmi_status_time":    {Type: schema.TypeString, Computed: true, Description: "IPMI status timestamp."},
	}
}
