package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceTagAssignment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTagAssignmentCreate,
		ReadContext:   resourceTagAssignmentRead,
		DeleteContext: resourceTagAssignmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceTagAssignmentImport,
		},
		Schema: map[string]*schema.Schema{
			"tag_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Tag ID to assign",
			},
			"resource_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{resourceNameVirtualServer, resourceNameVirtualServerVPC}, false)),
				Description:      "Resource type name accepted by the tag API",
			},
			"identifier": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Resource identifier to tag",
			},
		},
	}
}

func resourceTagAssignmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	tagID := d.Get("tag_id").(int)
	resourceName := d.Get("resource_name").(string)
	identifier := d.Get("identifier").(int)

	if err := c.AssignTagResource(tagID, resourceName, identifier); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tagAssignmentID(tagID, resourceName, identifier))
	return resourceTagAssignmentRead(ctx, d, m)
}

func resourceTagAssignmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	tagID := d.Get("tag_id").(int)
	resourceName := d.Get("resource_name").(string)
	identifier := d.Get("identifier").(int)

	tags, err := c.GetResourceTags(resourceName, identifier)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, tag := range tags {
		if tag.ID == tagID {
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceTagAssignmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	tagID := d.Get("tag_id").(int)
	resourceName := d.Get("resource_name").(string)
	identifier := d.Get("identifier").(int)

	if err := c.RemoveTagResource(tagID, resourceName, identifier); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceTagAssignmentImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("expected import ID tag_id/resource_name/identifier")
	}

	tagID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid tag_id %q: %w", parts[0], err)
	}
	identifier, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid identifier %q: %w", parts[2], err)
	}

	if err := d.Set("tag_id", tagID); err != nil {
		return nil, err
	}
	if err := d.Set("resource_name", parts[1]); err != nil {
		return nil, err
	}
	if err := d.Set("identifier", identifier); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func tagAssignmentID(tagID int, resourceName string, identifier int) string {
	return fmt.Sprintf("%d/%s/%d", tagID, resourceName, identifier)
}
