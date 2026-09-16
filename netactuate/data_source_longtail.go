package netactuate

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceLocationByCurrentIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLocationByCurrentIPRead,
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Public IP address returned by the API.",
			},
			"location": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Location detected for the caller's current IP address.",
			},
			"raw_json": rawJSONSchema("Complete location response as JSON, preserving API fields not modeled as first class attributes."),
		},
	}
}

func dataSourceLocationByCurrentIPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	location, err := m.(*ProviderClients).V2.GetLocationByCurrentIP()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("ip", location.IP, d, &diags)
	setValue("location", location.Location, d, &diags)
	setValue("raw_json", compactOptionalRawJSON(location.Raw), d, &diags)
	d.SetId("location")
	return diags
}

func dataSourceServices() *schema.Resource {
	return servicesListDataSource(dataSourceServicesRead, "Account services returned by the API.")
}

func dataSourceColocationServices() *schema.Resource {
	return servicesListDataSource(dataSourceColocationServicesRead, "Colocation services returned by the API.")
}

func dataSourceColocationService() *schema.Resource {
	return serviceItemDataSource(dataSourceColocationServiceRead, "Colocation service ID.")
}

func dataSourceIPTransitServices() *schema.Resource {
	return servicesListDataSource(dataSourceIPTransitServicesRead, "IP transit services returned by the API.")
}

func dataSourceIPTransitService() *schema.Resource {
	return serviceItemDataSource(dataSourceIPTransitServiceRead, "IP transit service ID.")
}

func dataSourceTransportServices() *schema.Resource {
	return servicesListDataSource(dataSourceTransportServicesRead, "Transport services returned by the API.")
}

func dataSourceTransportService() *schema.Resource {
	return serviceItemDataSource(dataSourceTransportServiceRead, "Transport service ID.")
}

func servicesListDataSource(read schema.ReadContextFunc, description string) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			"service_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Optional service ID filter sent to endpoints that accept a service_id query parameter.",
			},
			"services": serviceRowsSchema(description),
		},
	}
}

func serviceItemDataSource(read schema.ReadContextFunc, idDescription string) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			"service_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: idDescription,
			},
			"service": serviceRowBlockSchema("Service returned by the API."),
		},
	}
}

func serviceRowsSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: serviceRowSchema(),
		},
	}
}

func serviceRowBlockSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: serviceRowSchema(),
		},
	}
}

func serviceRowSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"service_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Service ID returned by the API id field.",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Service description returned by the API. Services carry a description rather than a name.",
		},
		"raw_json": rawJSONSchema("Complete service row as JSON, preserving API fields not modeled as first class attributes."),
	}
}

func dataSourceServicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	services, err := m.(*ProviderClients).V2.GetServices()
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(services))
	for i, service := range services {
		rows[i] = flattenServiceRow(service.ID, service.Description, service.Raw)
	}
	return setServiceRows(d, "services", "services", rows)
}

func dataSourceColocationServicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := serviceListOptions(d)
	services, err := m.(*ProviderClients).V2.GetColocationServices(opts)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(services))
	for i, service := range services {
		rows[i] = flattenServiceRow(service.ID, service.Description, service.Raw)
	}
	return setServiceRows(d, "services", serviceListID("colocation-services", opts.ServiceID), rows)
}

func dataSourceColocationServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceID := d.Get("service_id").(int)
	service, err := m.(*ProviderClients).V2.GetColocationService(serviceID)
	if err != nil {
		return diag.FromErr(err)
	}
	return setServiceRows(d, "service", fmt.Sprintf("colocation-service-%d", serviceID), []map[string]interface{}{
		flattenServiceRow(service.ID, service.Description, service.Raw),
	})
}

func dataSourceIPTransitServicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := serviceListOptions(d)
	services, err := m.(*ProviderClients).V2.GetIPTransitServices(opts)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(services))
	for i, service := range services {
		rows[i] = flattenServiceRow(service.ID, service.Description, service.Raw)
	}
	return setServiceRows(d, "services", serviceListID("iptransit-services", opts.ServiceID), rows)
}

func dataSourceIPTransitServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceID := d.Get("service_id").(int)
	service, err := m.(*ProviderClients).V2.GetIPTransitService(serviceID)
	if err != nil {
		return diag.FromErr(err)
	}
	return setServiceRows(d, "service", fmt.Sprintf("iptransit-service-%d", serviceID), []map[string]interface{}{
		flattenServiceRow(service.ID, service.Description, service.Raw),
	})
}

func dataSourceTransportServicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := serviceListOptions(d)
	services, err := m.(*ProviderClients).V2.GetTransportServices(opts)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(services))
	for i, service := range services {
		rows[i] = flattenServiceRow(service.ID, service.Description, service.Raw)
	}
	return setServiceRows(d, "services", serviceListID("transport-services", opts.ServiceID), rows)
}

func dataSourceTransportServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceID := d.Get("service_id").(int)
	service, err := m.(*ProviderClients).V2.GetTransportService(serviceID)
	if err != nil {
		return diag.FromErr(err)
	}
	return setServiceRows(d, "service", fmt.Sprintf("transport-service-%d", serviceID), []map[string]interface{}{
		flattenServiceRow(service.ID, service.Description, service.Raw),
	})
}

func dataSourceIPTransitIPs() *schema.Resource {
	return servicePortDataSource(dataSourceIPTransitIPsRead, "service_iptransit_id", "Optional IP transit service ID filter.", "IP addresses assigned to IP transit services.")
}

func dataSourceIPTransitPorts() *schema.Resource {
	return servicePortDataSource(dataSourceIPTransitPortsRead, "service_iptransit_id", "Optional IP transit service ID filter.", "Ports assigned to IP transit services.")
}

func dataSourceTransportPorts() *schema.Resource {
	return servicePortDataSource(dataSourceTransportPortsRead, "service_transport_id", "Optional transport service ID filter.", "Ports assigned to transport services.")
}

func servicePortDataSource(read schema.ReadContextFunc, filterName, filterDescription, rowsDescription string) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			filterName: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: filterDescription,
			},
			"rows": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: rowsDescription,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"row_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Row ID returned by the API id field.",
						},
						"service_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Parent service ID returned by the API.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name returned by the API when present.",
						},
						"ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP address returned by the API when present.",
						},
						"raw_json": rawJSONSchema("Complete row as JSON, preserving API fields not modeled as first class attributes."),
					},
				},
			},
		},
	}
}

func dataSourceIPTransitIPsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	filter := optionalIntFromData(d, "service_iptransit_id")
	addresses, err := m.(*ProviderClients).V2.GetIPTransitIPAddresses(filter)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(addresses))
	for i, address := range addresses {
		rows[i] = flattenServicePortRow(address.ID, address.ServiceIPTransitID, "", address.IP, address.Raw)
	}
	return setServiceRows(d, "rows", serviceListID("iptransit-ips", filter), rows)
}

func dataSourceIPTransitPortsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	filter := optionalIntFromData(d, "service_iptransit_id")
	ports, err := m.(*ProviderClients).V2.GetIPTransitPorts(filter)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(ports))
	for i, port := range ports {
		rows[i] = flattenServicePortRow(port.ID, port.ServiceIPTransitID, port.Name, "", port.Raw)
	}
	return setServiceRows(d, "rows", serviceListID("iptransit-ports", filter), rows)
}

func dataSourceTransportPortsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	filter := optionalIntFromData(d, "service_transport_id")
	ports, err := m.(*ProviderClients).V2.GetTransportPorts(filter)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]map[string]interface{}, len(ports))
	for i, port := range ports {
		rows[i] = flattenServicePortRow(port.ID, port.ServiceTransportID, port.Name, "", port.Raw)
	}
	return setServiceRows(d, "rows", serviceListID("transport-ports", filter), rows)
}

func serviceListOptions(d *schema.ResourceData) gona.ServiceListOptions {
	return gona.ServiceListOptions{ServiceID: optionalIntFromData(d, "service_id")}
}

func optionalIntFromData(d *schema.ResourceData, key string) *int {
	if v, ok := d.GetOkExists(key); ok {
		value := v.(int)
		return &value
	}
	return nil
}

func serviceListID(prefix string, value *int) string {
	if value == nil {
		return prefix
	}
	return prefix + "-" + strconv.Itoa(*value)
}

func setServiceRows(d *schema.ResourceData, key, id string, rows []map[string]interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	setValue(key, rows, d, &diags)
	d.SetId(id)
	return diags
}

func flattenServiceRow(id int, description string, raw json.RawMessage) map[string]interface{} {
	return map[string]interface{}{
		"service_id":  id,
		"description": description,
		"raw_json":    compactOptionalRawJSON(raw),
	}
}

func flattenServicePortRow(id, serviceID int, name, ip string, raw json.RawMessage) map[string]interface{} {
	return map[string]interface{}{
		"row_id":     id,
		"service_id": serviceID,
		"name":       name,
		"ip":         ip,
		"raw_json":   compactOptionalRawJSON(raw),
	}
}

func rawJSONSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: description,
	}
}

func compactOptionalRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var compact json.RawMessage
	if err := json.Unmarshal(raw, &compact); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(compact)
	if err != nil {
		return string(raw)
	}
	return string(out)
}
