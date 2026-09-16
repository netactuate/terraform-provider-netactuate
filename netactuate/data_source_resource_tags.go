package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceResourceTags() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceResourceTagsRead,
		Schema: map[string]*schema.Schema{
			"resource_name": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{resourceNameVirtualServer, resourceNameVirtualServerVPC}, false)),
				Description:      "Resource type name accepted by the tag API",
			},
			"identifier": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Resource identifier to read tags for",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Tags assigned to the resource",
				Elem: &schema.Resource{
					Schema: tagObjectSchema(),
				},
			},
		},
	}
}

func dataSourceResourceTagsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	resourceName := d.Get("resource_name").(string)
	identifier := d.Get("identifier").(int)

	tags, err := c.GetResourceTags(resourceName, identifier)
	if err != nil {
		return diag.FromErr(err)
	}

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

	var diags diag.Diagnostics
	setValue("tags", tagsList, d, &diags)
	d.SetId(resourceName + "/" + strconv.Itoa(identifier))
	return diags
}
