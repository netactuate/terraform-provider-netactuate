package netactuate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceCloudExtras() *schema.Resource {
	return rawJSONDataSource("mbpkgid", "Server package ID whose optional extras should be read.", "cloud-extras", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetCloudExtras(mbpkgID)
		return raw, fmt.Sprintf("cloud-extras-%d", mbpkgID), err
	})
}

func dataSourceImagesProvisioningJobsCount() *schema.Resource {
	return rawJSONDataSource("", "", "cloud-images-provisioning-jobs-count", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		raw, err := c.GetImagesProvisioningJobsCount()
		return raw, "cloud-images-provisioning-jobs-count", err
	})
}

func dataSourceCloudBaseImages() *schema.Resource {
	return imageListDataSource("base_images", "Base images available for cloud server deployment.", func(c *gona.Client) ([]gona.Image, error) {
		return c.GetBaseImages()
	})
}

func dataSourceCloudPrivateImages() *schema.Resource {
	return imageListDataSource("private_images", "Private images available to the account.", func(c *gona.Client) ([]gona.Image, error) {
		return c.GetPrivateImages()
	})
}

func dataSourceCloudIPLimits() *schema.Resource {
	return rawJSONDataSource("mbpkgid", "Server package ID whose IP limits should be read.", "cloud-iplimits", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetIPLimits(mbpkgID)
		return raw, fmt.Sprintf("cloud-iplimits-%d", mbpkgID), err
	})
}

func dataSourceCloudKernels() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudKernelsRead,
		Schema: map[string]*schema.Schema{
			"kernels": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Boot kernels available for cloud servers.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"kernel_id": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Kernel ID from the API field named id.",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Kernel name.",
					},
					"description": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Kernel description, or an empty string when the API returns null.",
					},
				}},
			},
		},
	}
}

func dataSourceCloudLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudLocationRead,
		Schema: map[string]*schema.Schema{
			"location_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Cloud location ID to read.",
			},
			"name":      computedString("Cloud location name."),
			"location":  computedString("Cloud location code."),
			"city":      computedString("Cloud location city."),
			"country":   computedString("Cloud location country."),
			"iata_code": computedString("Cloud location IATA code."),
			"flag":      computedString("Cloud location flag value."),
			"latitude":  computedString("Cloud location latitude, or an empty string when the API returns null."),
			"longitude": computedString("Cloud location longitude, or an empty string when the API returns null."),
		},
	}
}

func dataSourceCloudFloatingIPv4VMs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudFloatingIPv4VMsRead,
		Schema: map[string]*schema.Schema{
			"floating_ipv4_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Floating IPv4 ID whose allowed virtual machines should be read.",
			},
			"vms": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Virtual machines allowed to access the floating IPv4 address.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"mbpkgid": computedInt("Server package ID."),
					"fqdn":    computedString("Server hostname."),
					"ip":      computedString("Server IP address."),
				}},
			},
		},
	}
}

func dataSourceCloudNetworkingLocations() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudNetworkingLocationsRead,
		Schema: map[string]*schema.Schema{
			"locations": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Mappings between cloud location IDs and datacenter IDs.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"location_id":   computedInt("Cloud location ID."),
					"datacenter_id": computedInt("Datacenter ID."),
				}},
			},
		},
	}
}

func dataSourceCloudVLAN() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudVLANRead,
		Schema: map[string]*schema.Schema{
			"customer_vlan_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Customer VLAN ID to read.",
			},
			"vlan": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Customer VLAN details.",
				Elem:        &schema.Resource{Schema: vlanSchema()},
			},
		},
	}
}

func dataSourceCloudLocationVLANs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudLocationVLANsRead,
		Schema: map[string]*schema.Schema{
			"location_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Cloud location ID whose customer VLANs should be listed.",
			},
			"vlans": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Customer VLANs at the requested cloud location.",
				Elem:        &schema.Resource{Schema: vlanSchema()},
			},
		},
	}
}

func dataSourceCloudPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudPoolRead,
		Schema: map[string]*schema.Schema{
			"cloud_pool_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Cloud pool ID to read.",
			},
			"name":         computedString("Cloud pool name."),
			"description":  computedString("Cloud pool description."),
			"private":      computedInt("Private flag as returned by the API."),
			"created":      computedString("Cloud pool creation timestamp."),
			"last_updated": computedString("Cloud pool last update timestamp."),
		},
	}
}

func dataSourceCloudScalingOptions() *schema.Resource {
	return rawJSONDataSourceWithSchema(map[string]*schema.Schema{
		"mbpkgid": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Server package ID whose scaling options should be read.",
		},
		"include_current_plan": optionalBool("Whether to include the current plan in the returned scaling options."),
		"min_ram":              optionalInt("Minimum RAM filter for scaling options."),
		"max_ram":              optionalInt("Maximum RAM filter for scaling options."),
		"min_cpus":             optionalInt("Minimum CPU filter for scaling options."),
		"max_cpus":             optionalInt("Maximum CPU filter for scaling options."),
	}, func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		req := &gona.ScalingOptionsRequest{}
		if v, ok := d.GetOkExists("include_current_plan"); ok {
			value := v.(bool)
			req.IncludeCurrentPlan = &value
		}
		if v, ok := d.GetOk("min_ram"); ok {
			value := v.(int)
			req.MinRAM = &value
		}
		if v, ok := d.GetOk("max_ram"); ok {
			value := v.(int)
			req.MaxRAM = &value
		}
		if v, ok := d.GetOk("min_cpus"); ok {
			value := v.(int)
			req.MinCPUs = &value
		}
		if v, ok := d.GetOk("max_cpus"); ok {
			value := v.(int)
			req.MaxCPUs = &value
		}
		raw, err := c.GetScalingOptions(mbpkgID, req)
		return raw, fmt.Sprintf("cloud-scaling-%d", mbpkgID), err
	})
}

func dataSourceServerDeploymentInfo() *schema.Resource {
	return rawJSONDataSourceWithSchema(map[string]*schema.Schema{
		"contract_type": optionalString("Optional contract type filter for deployment metadata."),
	}, func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		contractType := d.Get("contract_type").(string)
		raw, err := c.GetServerDeploymentInfo(contractType)
		return raw, "cloud-server-deploy-info", err
	})
}

func dataSourceServerVNCStatus() *schema.Resource {
	return rawJSONDataSource("mbpkgid", "Server package ID whose VNC status should be read.", "cloud-server-vnc-status", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetServerVNCStatus(mbpkgID)
		return raw, fmt.Sprintf("cloud-server-vnc-status-%d", mbpkgID), err
	})
}

func dataSourceServerIPv4() *schema.Resource {
	return serverIPAddressDataSource("ipv4", "IPv4 addresses attached to the server.", func(c *gona.Client, mbpkgID int) ([]gona.ServerIPAddress, error) {
		return c.GetServerIPv4(mbpkgID)
	})
}

func dataSourceServerIPv6() *schema.Resource {
	return serverIPAddressDataSource("ipv6", "IPv6 addresses attached to the server.", func(c *gona.Client, mbpkgID int) ([]gona.ServerIPAddress, error) {
		return c.GetServerIPv6(mbpkgID)
	})
}

func dataSourceServerNetworkIPs() *schema.Resource {
	return rawJSONDataSource("mbpkgid", "Server package ID whose network IP data should be read.", "cloud-server-networkips", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetServerNetworkIPs(mbpkgID)
		return raw, fmt.Sprintf("cloud-server-networkips-%d", mbpkgID), err
	})
}

func dataSourceServerBGPSessionsRaw() *schema.Resource {
	return rawJSONDataSourceWithSchema(map[string]*schema.Schema{
		"mbpkgid": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Server package ID whose BGP sessions should be read.",
		},
		"group_type": optionalString("Optional BGP group type filter."),
	}, func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetServerBGPSessions(mbpkgID, d.Get("group_type").(string))
		return raw, fmt.Sprintf("cloud-server-sessions-%d", mbpkgID), err
	})
}

func dataSourceCurrentServer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCurrentServerRead,
		Schema:      computedServerSchema(),
	}
}

func dataSourceUnprovisionedPackages() *schema.Resource {
	return rawJSONDataSource("", "", "cloud-servers-unprovisioned", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		raw, err := c.GetUnprovisionedPackages()
		return raw, "cloud-servers-unprovisioned", err
	})
}

func dataSourceVirtualServerContract() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVirtualServerContractRead,
		Schema: contractUsageSchema(map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Server package ID whose virtual server contract should be read.",
			},
		}),
	}
}

func dataSourceServerSummary() *schema.Resource {
	return rawJSONDataSource("mbpkgid", "Server package ID whose summary should be read.", "cloud-server-summary", func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		mbpkgID := d.Get("mbpkgid").(int)
		raw, err := c.GetServerSummary(mbpkgID)
		return raw, fmt.Sprintf("cloud-server-summary-%d", mbpkgID), err
	})
}

func dataSourceCloudPlanID() *schema.Resource {
	return rawJSONDataSourceWithSchema(map[string]*schema.Schema{
		"plan_name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Plan name whose platform plan ID should be read.",
		},
	}, func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		planName := d.Get("plan_name").(string)
		raw, err := c.GetPlanID(planName)
		return raw, "cloud-plan-id-" + planName, err
	})
}

func dataSourceCloudDeploySizes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudDeploySizesRead,
		Schema: map[string]*schema.Schema{
			"location": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Location name or code whose deploy sizes should be read.",
			},
			"min_cpu": optionalInt("Minimum CPU filter for deploy sizes."),
			"min_ram": optionalInt("Minimum RAM filter for deploy sizes."),
			"sizes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Deploy sizes available at the requested location.",
				Elem:        &schema.Resource{Schema: sizeSchema()},
			},
		},
	}
}

func dataSourceCloudStorageLocations() *schema.Resource {
	return rawJSONDataSourceWithSchema(map[string]*schema.Schema{
		"cloud_pool_id": optionalInt("Optional cloud pool ID filter for storage locations."),
	}, func(c *gona.Client, d *schema.ResourceData) (json.RawMessage, string, error) {
		var poolID *int
		if v, ok := d.GetOk("cloud_pool_id"); ok {
			value := v.(int)
			poolID = &value
		}
		raw, err := c.GetStorageLocations(poolID)
		return raw, "cloud-storage-locations", err
	})
}

func rawJSONDataSource(argName, argDescription, idPrefix string, read func(*gona.Client, *schema.ResourceData) (json.RawMessage, string, error)) *schema.Resource {
	s := map[string]*schema.Schema{}
	if argName != "" {
		s[argName] = &schema.Schema{Type: schema.TypeInt, Required: true, Description: argDescription}
	}
	return rawJSONDataSourceWithSchema(s, read)
}

func rawJSONDataSourceWithSchema(s map[string]*schema.Schema, read func(*gona.Client, *schema.ResourceData) (json.RawMessage, string, error)) *schema.Resource {
	s["raw_json"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Raw JSON payload returned by the API.",
	}
	return &schema.Resource{
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			c := m.(*ProviderClients).V2
			raw, id, err := read(c, d)
			if err != nil {
				return diag.FromErr(err)
			}
			var diags diag.Diagnostics
			setValue("raw_json", string(raw), d, &diags)
			d.SetId(id)
			return diags
		},
		Schema: s,
	}
}

func imageListDataSource(field, description string, read func(*gona.Client) ([]gona.Image, error)) *schema.Resource {
	return &schema.Resource{
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			images, err := read(m.(*ProviderClients).V2)
			if err != nil {
				return diag.FromErr(err)
			}
			var diags diag.Diagnostics
			setValue(field, flattenImagesForCompute(images), d, &diags)
			d.SetId(field)
			return diags
		},
		Schema: map[string]*schema.Schema{
			field: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: description,
				Elem:        &schema.Resource{Schema: imageSchemaForCompute()},
			},
		},
	}
}

func dataSourceCloudKernelsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	kernels, err := m.(*ProviderClients).V2.GetKernels()
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(kernels))
	for i, kernel := range kernels {
		description := ""
		if kernel.Description != nil {
			description = *kernel.Description
		}
		rows[i] = map[string]interface{}{"kernel_id": kernel.ID, "name": kernel.Name, "description": description}
	}
	var diags diag.Diagnostics
	setValue("kernels", rows, d, &diags)
	d.SetId("cloud-kernels")
	return diags
}

func dataSourceCloudLocationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	locationID := d.Get("location_id").(int)
	location, err := m.(*ProviderClients).V2.GetCloudLocation(locationID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("name", location.Name, d, &diags)
	setValue("location", location.Location, d, &diags)
	setValue("city", location.City, d, &diags)
	setValue("country", location.Country, d, &diags)
	setValue("iata_code", location.IATACode, d, &diags)
	setValue("flag", location.Flag, d, &diags)
	setValue("latitude", stringPtrOrEmpty(location.Latitude), d, &diags)
	setValue("longitude", stringPtrOrEmpty(location.Longitude), d, &diags)
	d.SetId(fmt.Sprintf("cloud-location-%d", locationID))
	return diags
}

func dataSourceCloudFloatingIPv4VMsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	floatingIPv4ID := d.Get("floating_ipv4_id").(int)
	vms, err := m.(*ProviderClients).V3.ListCloudFloatingIPv4VMs(floatingIPv4ID)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		rows[i] = map[string]interface{}{"mbpkgid": vm.MBPkgID, "fqdn": vm.FQDN, "ip": vm.IP}
	}
	var diags diag.Diagnostics
	setValue("vms", rows, d, &diags)
	d.SetId(fmt.Sprintf("cloud-floating-ipv4-%d-vms", floatingIPv4ID))
	return diags
}

func dataSourceCloudNetworkingLocationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	locations, err := m.(*ProviderClients).V3.ListCloudNetworkingLocations()
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(locations))
	for i, location := range locations {
		rows[i] = map[string]interface{}{"location_id": location.LocationID, "datacenter_id": location.DatacenterID}
	}
	var diags diag.Diagnostics
	setValue("locations", rows, d, &diags)
	d.SetId("cloud-networking-locations")
	return diags
}

func dataSourceCloudVLANRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	customerVLANID := d.Get("customer_vlan_id").(int)
	vlan, err := m.(*ProviderClients).V2.GetCustomerVLAN(customerVLANID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("vlan", flattenCloudVLANs([]gona.VLAN{vlan}), d, &diags)
	d.SetId(fmt.Sprintf("cloud-vlan-%d", customerVLANID))
	return diags
}

func dataSourceCloudLocationVLANsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	locationID := d.Get("location_id").(int)
	vlans, err := m.(*ProviderClients).V2.ListCustomerVLANsAtLocation(locationID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("vlans", flattenCloudVLANs(vlans), d, &diags)
	d.SetId(fmt.Sprintf("cloud-location-%d-vlans", locationID))
	return diags
}

func dataSourceCloudPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	poolID := d.Get("cloud_pool_id").(int)
	pool, err := m.(*ProviderClients).V2.GetCloudPool(poolID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("name", pool.Name, d, &diags)
	setValue("description", pool.Description, d, &diags)
	setValue("private", pool.Private, d, &diags)
	setValue("created", pool.Created, d, &diags)
	setValue("last_updated", pool.LastUpdated, d, &diags)
	d.SetId(fmt.Sprintf("cloud-pool-%d", poolID))
	return diags
}

func serverIPAddressDataSource(field, description string, read func(*gona.Client, int) ([]gona.ServerIPAddress, error)) *schema.Resource {
	return &schema.Resource{
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			mbpkgID := d.Get("mbpkgid").(int)
			addresses, err := read(m.(*ProviderClients).V2, mbpkgID)
			if err != nil {
				return diag.FromErr(err)
			}
			var diags diag.Diagnostics
			setValue(field, flattenServerIPAddresses(addresses), d, &diags)
			d.SetId(fmt.Sprintf("server-%d-%s", mbpkgID, field))
			return diags
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Server package ID whose addresses should be read.",
			},
			field: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: description,
				Elem:        &schema.Resource{Schema: serverIPAddressSchema()},
			},
		},
	}
}

func dataSourceCurrentServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	server, err := m.(*ProviderClients).V2.GetCurrentServer()
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setComputedServerState(server, d, &diags)
	d.SetId(fmt.Sprintf("current-server-%d", server.ID))
	return diags
}

func dataSourceVirtualServerContractRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mbpkgID := d.Get("mbpkgid").(int)
	contract, err := m.(*ProviderClients).V2.GetVirtualServerContract(mbpkgID)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setContractUsageState(contract, d, &diags)
	d.SetId(fmt.Sprintf("virtual-server-contract-%d", mbpkgID))
	return diags
}

func dataSourceCloudDeploySizesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	req := &gona.DeploySizesRequest{}
	if v, ok := d.GetOk("min_cpu"); ok {
		value := v.(int)
		req.MinCPU = &value
	}
	if v, ok := d.GetOk("min_ram"); ok {
		value := v.(int)
		req.MinRAM = &value
	}
	sizes, err := m.(*ProviderClients).V2.GetDeploySizes(d.Get("location").(string), req)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("sizes", flattenSizesForCompute(sizes), d, &diags)
	d.SetId("cloud-deploy-sizes-" + d.Get("location").(string))
	return diags
}

func computedString(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Computed: true, Description: description}
}

func computedInt(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeInt, Computed: true, Description: description}
}

func computedBool(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeBool, Computed: true, Description: description}
}

func optionalString(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Optional: true, Description: description}
}

func optionalInt(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeInt, Optional: true, Description: description}
}

func optionalBool(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeBool, Optional: true, Description: description}
}

func imageSchemaForCompute() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"image_id":         computedInt("Image ID from the API field named id."),
		"name":             computedString("Image name."),
		"description":      computedString("Image description, or an empty string when the API returns null."),
		"type":             computedString("Image type."),
		"subtype":          computedString("Image subtype."),
		"bits":             computedString("Image architecture width."),
		"tech":             computedString("Image technology value."),
		"size":             computedString("Image size value."),
		"category":         computedString("Image category."),
		"enabled":          computedInt("Image enabled flag, or zero when the API returns null."),
		"script_bash":      computedInt("Whether the image supports bash scripts, using the API integer flag."),
		"script_cloudinit": computedInt("Whether the image supports cloud-init scripts, using the API integer flag."),
		"created":          computedString("Image creation timestamp."),
		"updated":          computedString("Image update timestamp."),
	}
}

func flattenImagesForCompute(images []gona.Image) []map[string]interface{} {
	rows := make([]map[string]interface{}, len(images))
	for i, image := range images {
		rows[i] = map[string]interface{}{
			"image_id":         image.ID,
			"name":             image.Name,
			"description":      stringPtrOrEmpty(image.Description),
			"type":             image.Type,
			"subtype":          image.Subtype,
			"bits":             image.Bits,
			"tech":             image.Tech,
			"size":             image.Size,
			"category":         image.Category,
			"enabled":          intPtrOrZero(image.Enabled),
			"script_bash":      image.ScriptBash,
			"script_cloudinit": image.ScriptCloudinit,
			"created":          image.Created,
			"updated":          image.Updated,
		}
	}
	return rows
}

func vlanSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"customer_vlan_id": computedInt("Customer VLAN ID from the API field named id."),
		"mbid":             computedInt("Account ID for the VLAN."),
		"private":          computedInt("Private flag as returned by the API."),
		"allow_sriov":      computedInt("SR-IOV flag as returned by the API."),
		"display_name":     computedString("VLAN display name."),
		"description":      computedString("VLAN description."),
		"last_updated":     computedString("VLAN last update timestamp."),
		"created":          computedString("VLAN creation timestamp."),
		"provisioned_locations": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Locations where the VLAN has provisioning status.",
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"provisioned": computedBool("Whether the VLAN is provisioned in this location."),
				"name":        computedString("Location name."),
				"location_id": computedInt("Location ID."),
				"flag":        computedString("Location flag value."),
				"iata_code":   computedString("Location IATA code."),
			}},
		},
	}
}

func flattenCloudVLANs(vlans []gona.VLAN) []map[string]interface{} {
	rows := make([]map[string]interface{}, len(vlans))
	for i, vlan := range vlans {
		locations := make([]map[string]interface{}, len(vlan.ProvisionedLocations))
		for j, location := range vlan.ProvisionedLocations {
			locations[j] = map[string]interface{}{
				"provisioned": location.Provisioned,
				"name":        location.Name,
				"location_id": location.LocationID,
				"flag":        location.Flag,
				"iata_code":   location.IATACode,
			}
		}
		rows[i] = map[string]interface{}{
			"customer_vlan_id":      vlan.ID,
			"mbid":                  vlan.MBID,
			"private":               vlan.Private,
			"allow_sriov":           vlan.AllowSRIOV,
			"display_name":          vlan.DisplayName,
			"description":           vlan.Description,
			"last_updated":          vlan.LastUpdated,
			"created":               vlan.Created,
			"provisioned_locations": locations,
		}
	}
	return rows
}

func serverIPAddressSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"address_id": computedInt("Address ID from the API field named id."),
		"ip":         computedString("IP address."),
		"reverse":    computedString("Reverse DNS value, or an empty string when the API returns null."),
		"netmask":    computedString("Netmask, or an empty string when the API returns null."),
		"gateway":    computedString("Gateway, or an empty string when the API returns null."),
		"type":       computedString("Address type, or an empty string when the API returns null."),
		"primary":    computedInt("Primary flag, or zero when the API returns null."),
		"raw_json":   computedString("Raw JSON for this address."),
	}
}

func flattenServerIPAddresses(addresses []gona.ServerIPAddress) []map[string]interface{} {
	rows := make([]map[string]interface{}, len(addresses))
	for i, address := range addresses {
		rows[i] = map[string]interface{}{
			"address_id": address.ID,
			"ip":         address.IP,
			"reverse":    stringPtrOrEmpty(address.Reverse),
			"netmask":    stringPtrOrEmpty(address.Netmask),
			"gateway":    stringPtrOrEmpty(address.Gateway),
			"type":       stringPtrOrEmpty(address.Type),
			"primary":    intPtrOrZero(address.Primary),
			"raw_json":   string(address.Raw),
		}
	}
	return rows
}

func computedServerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mbpkgid":                     computedInt("Server package ID."),
		"hostname":                    computedString("Server hostname."),
		"os":                          computedString("Server operating system name."),
		"os_id":                       computedInt("Server operating system ID."),
		"primary_ipv4":                computedString("Server primary IPv4 address."),
		"primary_ipv6":                computedString("Server primary IPv6 address."),
		"plan_id":                     computedInt("Server plan ID."),
		"package":                     computedString("Server package name."),
		"package_billing_contract_id": computedInt("Server contract ID, from the API field named contract_id."),
		"location":                    computedString("Server location name."),
		"location_id":                 computedInt("Server location ID."),
		"status":                      computedString("Server status."),
		"state":                       computedString("Server power state."),
		"installed":                   computedInt("Installed flag as returned by the API."),
		"cloud_pool_id":               computedInt("Cloud pool ID, or zero when the API returns null."),
		"vpc_id":                      computedInt("VPC ID, or zero when the API returns null."),
		"vpc_reserved_network":        computedString("Reserved VPC network address."),
	}
}

func setComputedServerState(server gona.Server, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("mbpkgid", server.ID, d, diags)
	setValue("hostname", server.Name, d, diags)
	setValue("os", server.OS, d, diags)
	setValue("os_id", server.OSID, d, diags)
	setValue("primary_ipv4", server.PrimaryIPv4, d, diags)
	setValue("primary_ipv6", server.PrimaryIPv6, d, diags)
	setValue("plan_id", server.PlanID, d, diags)
	setValue("package", server.Package, d, diags)
	setValue("package_billing_contract_id", server.PackageBillingContractId, d, diags)
	setValue("location", server.Location, d, diags)
	setValue("location_id", server.LocationID, d, diags)
	setValue("status", server.ServerStatus, d, diags)
	setValue("state", server.PowerStatus, d, diags)
	setValue("installed", server.Installed, d, diags)
	setValue("cloud_pool_id", intPtrOrZero(server.CloudPoolID), d, diags)
	setValue("vpc_id", intPtrOrZero(server.VpcID), d, diags)
	setValue("vpc_reserved_network", server.VpcReservedNetwork, d, diags)
}

func sizeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"plan_id":   computedInt("Plan ID."),
		"plan":      computedString("Plan name."),
		"ram":       computedString("RAM amount as returned by the API."),
		"disk":      computedString("Disk amount as returned by the API."),
		"transfer":  computedString("Transfer amount as returned by the API."),
		"price":     computedString("Price as returned by the API."),
		"cpu":       computedInt("CPU count."),
		"port":      computedString("Port speed as returned by the API."),
		"available": {Type: schema.TypeFloat, Computed: true, Description: "Available capacity as returned by the API."},
	}
}

func flattenSizesForCompute(sizes []gona.Size) []map[string]interface{} {
	rows := make([]map[string]interface{}, len(sizes))
	for i, size := range sizes {
		rows[i] = map[string]interface{}{
			"plan_id":   size.PlanID,
			"plan":      size.Plan,
			"ram":       size.RAM,
			"disk":      size.Disk,
			"transfer":  size.Transfer,
			"price":     size.Price,
			"cpu":       size.CPU,
			"port":      size.Port,
			"available": size.Available,
		}
	}
	return rows
}

func contractUsageSchema(extra map[string]*schema.Schema) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"contract_id":          computedInt("Contract usage record ID from the API field named id."),
		"contract_mbpkgid":     computedInt("Contract package ID."),
		"parent_contract_id":   computedInt("Parent contract ID."),
		"brand":                computedString("Contract brand."),
		"mb_id":                computedInt("Account ID."),
		"contract_type":        computedString("Contract type."),
		"is_free":              computedInt("Free contract flag."),
		"include_bandwidth":    computedInt("Included bandwidth flag."),
		"customer_po":          computedString("Customer purchase order."),
		"customer_description": computedString("Customer description."),
		"po_monthly_limit":     computedInt("Purchase order monthly limit."),
		"monthly_discount":     computedInt("Monthly discount percentage."),
		"hourly_discount":      computedInt("Hourly discount percentage."),
		"max_cpus":             computedInt("Maximum CPUs."),
		"max_ram":              computedInt("Maximum RAM."),
		"max_disk":             computedInt("Maximum disk."),
		"allow_overage":        computedInt("Allow overage flag."),
	}
	for k, v := range extra {
		s[k] = v
	}
	return s
}

func setContractUsageState(contract gona.ContractUsage, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("contract_id", intPtrOrZero(contract.ID), d, diags)
	setValue("contract_mbpkgid", intPtrOrZero(contract.ContractMBPkgID), d, diags)
	setValue("parent_contract_id", intPtrOrZero(contract.ParentContractID), d, diags)
	setValue("brand", stringPtrOrEmpty(contract.Brand), d, diags)
	setValue("mb_id", intPtrOrZero(contract.MBID), d, diags)
	setValue("contract_type", stringPtrOrEmpty(contract.ContractType), d, diags)
	setValue("is_free", intPtrOrZero(contract.IsFree), d, diags)
	setValue("include_bandwidth", intPtrOrZero(contract.IncludeBandwidth), d, diags)
	setValue("customer_po", stringPtrOrEmpty(contract.CustomerPO), d, diags)
	setValue("customer_description", stringPtrOrEmpty(contract.CustomerDescription), d, diags)
	setValue("po_monthly_limit", intPtrOrZero(contract.POMonthlyLimit), d, diags)
	setValue("monthly_discount", intPtrOrZero(contract.MonthlyDiscount), d, diags)
	setValue("hourly_discount", intPtrOrZero(contract.HourlyDiscount), d, diags)
	setValue("max_cpus", intPtrOrZero(contract.MaxCPUs), d, diags)
	setValue("max_ram", intPtrOrZero(contract.MaxRAM), d, diags)
	setValue("max_disk", intPtrOrZero(contract.MaxDisk), d, diags)
	setValue("allow_overage", intPtrOrZero(contract.AllowOverage), d, diags)
}
