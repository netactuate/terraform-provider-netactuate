package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTags() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTagsRead,
		Schema: map[string]*schema.Schema{
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of tags visible to the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Tag ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Tag name",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Tag description",
						},
						"icon": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Tag icon",
						},
						"color": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Tag color",
						},
						"is_default": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Default tag flag as returned by the API, usually 0 or 1",
						},
						"is_favorite": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Favorite tag flag as returned by the API, usually 0 or 1",
						},
						"is_locked": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Locked tag flag as returned by the API, usually 0 or 1",
						},
						"show_dashboard": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Dashboard visibility flag as returned by the API, usually 0 or 1",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Tag creation timestamp",
						},
						"mb_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Account ID",
						},
						"resources_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of resources assigned to the tag",
						},
						"resources": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Resources assigned to the tag",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Tag resource assignment ID",
									},
									"resource_tag_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Resource tag ID",
									},
									"resource_name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Resource type name",
									},
									"identifier": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Resource identifier",
									},
									"created_at": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Resource assignment creation timestamp",
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

func dataSourceTagsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	tags, err := c.GetTags()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	tagsList := make([]map[string]interface{}, len(tags))
	for i, tag := range tags {
		resourcesList := make([]map[string]interface{}, len(tag.Resources))
		for j, resource := range tag.Resources {
			resourcesList[j] = map[string]interface{}{
				"id":              resource.ID,
				"resource_tag_id": resource.ResourceTagID,
				"resource_name":   resource.ResourceName,
				"identifier":      resource.Identifier,
				"created_at":      resource.CreatedAt,
			}
		}

		tagsList[i] = map[string]interface{}{
			"id":              tag.ID,
			"name":            tag.Name,
			"description":     tag.Description,
			"icon":            tag.Icon,
			"color":           tag.Color,
			"is_default":      tag.IsDefault,
			"is_favorite":     tag.IsFavorite,
			"is_locked":       tag.IsLocked,
			"show_dashboard":  tag.ShowDashboard,
			"created_at":      tag.CreatedAt,
			"mb_id":           tag.MbID,
			"resources_count": tag.ResourcesCount,
			"resources":       resourcesList,
		}
	}

	setValue("tags", tagsList, d, &diags)
	d.SetId("tags")

	return diags
}
