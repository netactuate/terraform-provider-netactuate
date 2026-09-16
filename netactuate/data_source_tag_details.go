package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceTagResources() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTagResourcesRead,
		Schema: map[string]*schema.Schema{
			"tag_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Tag ID whose assigned resources will be listed.",
			},
			"resources": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Resources assigned to the tag.",
				Elem: &schema.Resource{
					Schema: tagResourceSchema(),
				},
			},
		},
	}
}

func dataSourceTagLogs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTagLogsRead,
		Schema: map[string]*schema.Schema{
			"tag_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Tag ID whose logs will be listed.",
			},
			"logs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Log entries for the tag.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"log_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Tag log ID returned by the API id field.",
						},
						"tag_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Tag ID returned by the API.",
						},
						"action": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Log action returned by the API.",
						},
						"message": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Log message returned by the API.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Log creation timestamp returned by the API.",
						},
						"raw_json": rawJSONSchema("Complete log row as JSON, preserving API fields not modeled as first class attributes."),
					},
				},
			},
		},
	}
}

func tagResourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Tag resource assignment ID.",
		},
		"resource_tag_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Resource tag ID.",
		},
		"resource_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Resource type name.",
		},
		"identifier": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Resource identifier.",
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Resource assignment creation timestamp.",
		},
	}
}

func dataSourceTagResourcesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tagID := d.Get("tag_id").(int)
	resources, err := m.(*ProviderClients).V2.GetTagResources(tagID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("resources", flattenTagResources(resources), d, &diags)
	d.SetId("tag-resources-" + strconv.Itoa(tagID))
	return diags
}

func dataSourceTagLogsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tagID := d.Get("tag_id").(int)
	logs, err := m.(*ProviderClients).V2.GetTagLogs(tagID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("logs", flattenTagLogs(logs), d, &diags)
	d.SetId("tag-logs-" + strconv.Itoa(tagID))
	return diags
}

func flattenTagResources(resources []gona.TagResource) []map[string]interface{} {
	out := make([]map[string]interface{}, len(resources))
	for i, resource := range resources {
		out[i] = map[string]interface{}{
			"id":              resource.ID,
			"resource_tag_id": resource.ResourceTagID,
			"resource_name":   resource.ResourceName,
			"identifier":      resource.Identifier,
			"created_at":      resource.CreatedAt,
		}
	}
	return out
}

func flattenTagLogs(logs []gona.TagLog) []map[string]interface{} {
	out := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		out[i] = map[string]interface{}{
			"log_id":     log.ID,
			"tag_id":     log.TagID,
			"action":     log.Action,
			"message":    log.Message,
			"created_at": log.CreatedAt,
			"raw_json":   compactOptionalRawJSON(log.Raw),
		}
	}
	return out
}
