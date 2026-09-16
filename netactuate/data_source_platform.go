package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourcePlatformStatus() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformStatusRead,
		Schema: map[string]*schema.Schema{
			"services": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Platform status services returned by the API, sorted by service name. Service names are API data and are not a fixed set.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Service name from the dynamic API response key.",
						},
						"component_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Component ID for this platform service.",
						},
						"locations": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Location status entries for this service, sorted by location name. Location names are API data and are not a fixed set.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"location": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Location name from the dynamic API response key.",
									},
									"container_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Container ID for this service at this location.",
									},
									"status": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Status returned by the API for this service at this location.",
									},
									"last_updated": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Last updated timestamp returned by the API.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourcePlatformStatusRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	services, err := c.GetPlatformStatus()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("services", flattenPlatformStatus(services), d, &diags)
	d.SetId("platform-status")
	return diags
}

func dataSourcePlatformChangeLog() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformChangeLogRead,
		Schema: map[string]*schema.Schema{
			"entries": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Platform change log entries returned by the API.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"change_log_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Change log entry ID from the API id field.",
						},
						"title": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Change log title returned by the API. API text is passed through unchanged.",
						},
						"short_description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Short description returned by the API.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status returned by the API.",
						},
						"entry_json": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Complete change log row as JSON, preserving API fields not modeled as first class attributes.",
						},
					},
				},
			},
		},
	}
}

func dataSourcePlatformChangeLogEntry() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformChangeLogEntryRead,
		Schema: map[string]*schema.Schema{
			"change_log_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Platform change log entry ID.",
			},
			"entry": platformChangeLogEntryBlockSchema("Platform change log entry returned by the API."),
		},
	}
}

func platformChangeLogEntryBlockSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: platformChangeLogEntrySchema(),
		},
	}
}

func platformChangeLogEntrySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"change_log_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Change log entry ID from the API id field.",
		},
		"title": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Change log title returned by the API. API text is passed through unchanged.",
		},
		"short_description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Short description returned by the API.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Status returned by the API.",
		},
		"entry_json": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Complete change log row as JSON, preserving API fields not modeled as first class attributes.",
		},
	}
}

func dataSourcePlatformChangeLogRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	entries, err := c.GetPlatformChangeLog()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("entries", flattenPlatformChangeLog(entries), d, &diags)
	d.SetId("platform-change-log")
	return diags
}

func dataSourcePlatformChangeLogEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	entryID := d.Get("change_log_id").(int)

	entry, err := c.GetPlatformChangeLogEntry(entryID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("entry", flattenPlatformChangeLog([]gona.PlatformChangeLogEntry{*entry}), d, &diags)
	d.SetId(fmt.Sprintf("platform-change-log-entry-%d", entryID))
	return diags
}

func dataSourcePlatformDatacenters() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformDatacentersRead,
		Schema: map[string]*schema.Schema{
			"location": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Location whose platform datacenters will be listed. This accepts the location value accepted by the API.",
			},
			"datacenters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Datacenters returned by the API.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"datacenter_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Datacenter ID returned by the API id field.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Datacenter name returned by the API.",
						},
						"location": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Requested location used to look up this datacenter.",
						},
						"iata_code": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Datacenter IATA code returned by the API.",
						},
					},
				},
			},
		},
	}
}

func dataSourcePlatformDatacentersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	location := d.Get("location").(string)

	datacenters, err := c.GetPlatformDatacenters(location)
	if err != nil {
		return diag.FromErr(err)
	}

	rows := make([]map[string]interface{}, len(datacenters))
	for i, dc := range datacenters {
		rows[i] = map[string]interface{}{
			"datacenter_id": dc.ID,
			"name":          dc.Name,
			"location":      location,
			"iata_code":     dc.IATA,
		}
	}

	var diags diag.Diagnostics
	setValue("datacenters", rows, d, &diags)
	d.SetId("platform-datacenters-" + location)
	return diags
}

func dataSourcePlatformLookingGlassInit() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformLookingGlassInitRead,
		Schema: map[string]*schema.Schema{
			"raw_json": rawJSONSchema("Complete looking glass initialization response as JSON."),
		},
	}
}

func dataSourcePlatformLookingGlassInitRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	result, err := m.(*ProviderClients).V2.GetPlatformLookingGlassInit()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("raw_json", compactOptionalRawJSON(result.Raw), d, &diags)
	d.SetId("platform-looking-glass-init")
	return diags
}

func dataSourcePlatformLookingGlass() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformLookingGlassRead,
		Schema: map[string]*schema.Schema{
			"action": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Looking glass action query parameter.",
			},
			"target": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Looking glass target query parameter.",
			},
			"location": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Looking glass location query parameter.",
			},
			"full": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Looking glass full query parameter.",
			},
			"raw_json": rawJSONSchema("Complete looking glass execution response as JSON."),
		},
	}
}

func dataSourcePlatformLookingGlassRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opts := gona.PlatformLookingGlassExecuteOptions{}
	if v, ok := d.GetOk("action"); ok {
		action := v.(string)
		opts.Action = &action
	}
	if v, ok := d.GetOk("target"); ok {
		target := v.(string)
		opts.Target = &target
	}
	if v, ok := d.GetOk("location"); ok {
		location := v.(string)
		opts.Location = &location
	}
	if v, ok := d.GetOkExists("full"); ok {
		full := v.(int)
		opts.Full = &full
	}

	result, err := m.(*ProviderClients).V2.ExecutePlatformLookingGlass(opts)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("raw_json", compactOptionalRawJSON(result.Raw), d, &diags)
	d.SetId(fmt.Sprintf("platform-looking-glass-%s-%s-%s-%s",
		optionalStringIDPart(opts.Action),
		optionalStringIDPart(opts.Target),
		optionalStringIDPart(opts.Location),
		optionalIntIDPart(opts.Full)))
	return diags
}

func dataSourcePlatformMaintenanceInfo() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePlatformMaintenanceInfoRead,
		Schema: map[string]*schema.Schema{
			"maintenance_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Platform maintenance entry ID.",
			},
			"raw_json": rawJSONSchema("Complete platform maintenance response as JSON."),
		},
	}
}

func dataSourcePlatformMaintenanceInfoRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	maintenanceID := d.Get("maintenance_id").(int)
	result, err := m.(*ProviderClients).V2.GetPlatformMaintenanceInfo(maintenanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("raw_json", compactOptionalRawJSON(result.Raw), d, &diags)
	d.SetId(fmt.Sprintf("platform-maintenance-info-%d", maintenanceID))
	return diags
}

func dataSourcePlatformIncidents() *schema.Resource {
	return platformEventsDataSource(dataSourcePlatformIncidentsRead, map[string]*schema.Schema{
		"active": platformEventListSchema("Active platform incident events for the requested location."),
	})
}

func dataSourcePlatformIncidentHistory() *schema.Resource {
	return platformEventsDataSource(dataSourcePlatformIncidentHistoryRead, map[string]*schema.Schema{
		"historic": platformEventListSchema("Historic platform incident events for the requested location."),
	})
}

func dataSourcePlatformMaintenance() *schema.Resource {
	return platformEventsDataSource(dataSourcePlatformMaintenanceRead, map[string]*schema.Schema{
		"active":   platformEventListSchema("Active platform maintenance events for the requested location."),
		"upcoming": platformEventListSchema("Upcoming platform maintenance events for the requested location."),
	})
}

func dataSourcePlatformMaintenanceHistory() *schema.Resource {
	return platformEventsDataSource(dataSourcePlatformMaintenanceHistoryRead, map[string]*schema.Schema{
		"historic": platformEventListSchema("Historic platform maintenance events for the requested location."),
	})
}

func platformEventsDataSource(read schema.ReadContextFunc, eventAttrs map[string]*schema.Schema) *schema.Resource {
	resourceSchema := map[string]*schema.Schema{
		"location": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Location to query. This is a string and accepts either a numeric location ID or an IATA code, for example 3 or sjc.",
		},
	}
	for name, attr := range eventAttrs {
		resourceSchema[name] = attr
	}

	return &schema.Resource{
		ReadContext: read,
		Schema:      resourceSchema,
	}
}

func platformEventListSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"event_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event ID returned by the API.",
				},
				"type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event type returned by the API.",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event name returned by the API.",
				},
				"status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event status returned by the API.",
				},
				"start_time": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event start time returned by the API.",
				},
				"end_time": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Event end time returned by the API.",
				},
				"components": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "Component IDs returned by the API for this event.",
					Elem: &schema.Schema{
						Type:        schema.TypeString,
						Description: "Component ID.",
					},
				},
				"containers": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "Container IDs returned by the API for this event.",
					Elem: &schema.Schema{
						Type:        schema.TypeString,
						Description: "Container ID.",
					},
				},
			},
		},
	}
}

func dataSourcePlatformIncidentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	location := d.Get("location").(string)

	events, err := c.GetPlatformIncidents(location)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	// Active and upcoming event shapes are inferred from history because both were empty when measured.
	setValue("active", flattenPlatformEvents(events.Active), d, &diags)
	d.SetId(fmt.Sprintf("platform-incidents-%s", location))
	return diags
}

func dataSourcePlatformIncidentHistoryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	location := d.Get("location").(string)

	events, err := c.GetPlatformIncidentHistory(location)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("historic", flattenPlatformEvents(events.Historic), d, &diags)
	d.SetId(fmt.Sprintf("platform-incident-history-%s", location))
	return diags
}

func dataSourcePlatformMaintenanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	location := d.Get("location").(string)

	events, err := c.GetPlatformMaintenance(location)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	// Active and upcoming event shapes are inferred from history because both were empty when measured.
	setValue("active", flattenPlatformEvents(events.Active), d, &diags)
	setValue("upcoming", flattenPlatformEvents(events.Upcoming), d, &diags)
	d.SetId(fmt.Sprintf("platform-maintenance-%s", location))
	return diags
}

func dataSourcePlatformMaintenanceHistoryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	location := d.Get("location").(string)

	events, err := c.GetPlatformMaintenanceHistory(location)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("historic", flattenPlatformEvents(events.Historic), d, &diags)
	d.SetId(fmt.Sprintf("platform-maintenance-history-%s", location))
	return diags
}

func flattenPlatformStatus(services []gona.PlatformStatusService) []map[string]interface{} {
	out := make([]map[string]interface{}, len(services))
	for i, service := range services {
		locations := make([]map[string]interface{}, len(service.Locations))
		for j, location := range service.Locations {
			locations[j] = map[string]interface{}{
				"location":     location.Location,
				"container_id": location.ContainerID,
				"status":       location.Status,
				"last_updated": location.LastUpdated,
			}
		}
		out[i] = map[string]interface{}{
			"service":      service.Service,
			"component_id": service.ComponentID,
			"locations":    locations,
		}
	}
	return out
}

func flattenPlatformChangeLog(entries []gona.PlatformChangeLogEntry) []map[string]interface{} {
	out := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		out[i] = map[string]interface{}{
			"change_log_id":     entry.ChangeLogID,
			"title":             entry.Title,
			"short_description": entry.ShortDescription,
			"status":            entry.Status,
			"entry_json":        string(entry.EntryJSON),
		}
	}
	return out
}

func optionalStringIDPart(value *string) string {
	if value == nil {
		return "unset"
	}
	return *value
}

func flattenPlatformEvents(events []gona.PlatformEvent) []map[string]interface{} {
	out := make([]map[string]interface{}, len(events))
	for i, event := range events {
		out[i] = map[string]interface{}{
			"event_id":   event.EventID,
			"type":       event.Type,
			"name":       event.Name,
			"status":     event.Status,
			"start_time": event.StartTime,
			"end_time":   event.EndTime,
			"components": event.Components,
			"containers": event.Containers,
		}
	}
	return out
}
