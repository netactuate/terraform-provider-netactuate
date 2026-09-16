package netactuate

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func tagObjectSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
				Schema: tagResourceObjectSchema(),
			},
		},
	}
}

func tagResourceObjectSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
	}
}
