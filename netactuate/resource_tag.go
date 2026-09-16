package netactuate

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceTag() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTagCreate,
		ReadContext:   resourceTagRead,
		UpdateContext: resourceTagUpdate,
		DeleteContext: resourceTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				// The platform normalises the name on BOTH create and update, in TagController:
				//   strtolower(str_replace(' ', '-', $request->name))
				// so "My Tag" is stored as "my-tag". Config and state otherwise disagree forever:
				// plan shows a name change, apply succeeds, and the next plan shows the same
				// change again without converging.
				//
				// Case-only suppression leaves a name containing a space permanently dirty.
				// Uniqueness is checked
				// against the normalised form too (TagNameNotTakenRule), so two names differing
				// only by case or spaces are the same tag to the platform.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return normaliseTagName(old) == normaliseTagName(new)
				},
				Description: "Tag name. The platform lowercases it and replaces spaces with hyphens, so \"My Tag\" is stored as \"my-tag\".",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Tag description",
			},
			"color": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Tag color",
			},
			"icon": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Tag icon",
			},
			"is_default": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{0, 1})),
				Description:      "Default tag flag as a 0 or 1 integer, not a boolean",
			},
			"is_favorite": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{0, 1})),
				Description:      "Favorite tag flag as a 0 or 1 integer, not a boolean",
			},
			"is_locked": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{0, 1})),
				Description:      "Locked tag flag as a 0 or 1 integer, not a boolean",
			},
			"show_dashboard": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{0, 1})),
				Description:      "Dashboard visibility flag as a 0 or 1 integer, not a boolean",
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
		},
	}
}

func resourceTagCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.CreateTagRequest{
		Name: d.Get("name").(string),
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}
	if v, ok := d.GetOk("icon"); ok {
		req.Icon = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		req.Color = v.(string)
	}

	tag, err := c.CreateTag(req)
	if err != nil {
		return diag.FromErr(err)
	}
	if tag == nil || tag.ID == 0 {
		return diag.Errorf("tag %q was created without an ID", req.Name)
	}

	d.SetId(strconv.Itoa(tag.ID))
	if tagNeedsUpdateAfterCreate(d) {
		if _, err := c.UpdateTag(tag.ID, tagUpdateRequest(d, tag)); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceTagRead(ctx, d, m)
}

func resourceTagRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tag, err := c.GetTag(id)
	if err != nil {
		if tagNotFound(err) {
			log.Printf("[WARN] Tag %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setTagState(tag, d, &diags)
	return diags
}

func resourceTagUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	current, err := c.GetTag(id)
	if err != nil {
		if tagNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if _, err := c.UpdateTag(id, tagUpdateRequest(d, current)); err != nil {
		return diag.FromErr(err)
	}

	return resourceTagRead(ctx, d, m)
}

func resourceTagDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteTag(id); err != nil {
		if gona.IsNotFound(err) || tagNotFound(err) {
			d.SetId("")
			return nil
		}
		// The API refuses with 412 when a tag is locked or still assigned to something. The
		// underlying message says which, and is passed through, but on its own it does not tell
		// the reader what to do about it. Behaviour documented in gona's DeleteTag; the two
		// causes are recorded there rather than measured here, because provoking a 412 needs a
		// locked tag or a live assignment.
		if strings.Contains(err.Error(), "412") {
			return diag.Errorf(
				"tag %d could not be deleted: %s. A tag is refused while it is locked "+
					"(is_locked = 1) or still assigned to a resource. Remove the "+
					"netactuate_tag_assignment resources that reference it, or clear is_locked, "+
					"then retry.", id, err)
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func tagNeedsUpdateAfterCreate(d *schema.ResourceData) bool {
	return d.Get("is_default").(int) != 0 ||
		d.Get("is_favorite").(int) != 0 ||
		d.Get("is_locked").(int) != 0 ||
		d.Get("show_dashboard").(int) != 0
}

func tagUpdateRequest(d *schema.ResourceData, current *gona.Tag) *gona.UpdateTagRequest {
	req := &gona.UpdateTagRequest{
		Name:          d.Get("name").(string),
		Description:   current.Description,
		Icon:          current.Icon,
		Color:         current.Color,
		IsDefault:     d.Get("is_default").(int),
		IsFavorite:    d.Get("is_favorite").(int),
		IsLocked:      d.Get("is_locked").(int),
		ShowDashboard: d.Get("show_dashboard").(int),
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}
	if v, ok := d.GetOk("icon"); ok {
		req.Icon = v.(string)
	}
	if v, ok := d.GetOk("color"); ok {
		req.Color = v.(string)
	}
	return req
}

func setTagState(tag *gona.Tag, d *schema.ResourceData, diags *diag.Diagnostics) {
	resources := make([]map[string]interface{}, len(tag.Resources))
	for i, resource := range tag.Resources {
		resources[i] = map[string]interface{}{
			"id":              resource.ID,
			"resource_tag_id": resource.ResourceTagID,
			"resource_name":   resource.ResourceName,
			"identifier":      resource.Identifier,
			"created_at":      resource.CreatedAt,
		}
	}

	setValue("name", tag.Name, d, diags)
	setValue("description", tag.Description, d, diags)
	setValue("icon", tag.Icon, d, diags)
	setValue("color", tag.Color, d, diags)
	setValue("is_default", tag.IsDefault, d, diags)
	setValue("is_favorite", tag.IsFavorite, d, diags)
	setValue("is_locked", tag.IsLocked, d, diags)
	setValue("show_dashboard", tag.ShowDashboard, d, diags)
	setValue("created_at", tag.CreatedAt, d, diags)
	setValue("mb_id", tag.MbID, d, diags)
	setValue("resources_count", tag.ResourcesCount, d, diags)
	setValue("resources", resources, d, diags)
}

func tagNotFound(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "tag ") &&
		strings.Contains(strings.ToLower(err.Error()), "not found")
}

// normaliseTagName mirrors the platform's own normalisation so a plan can settle.
// TagController applies strtolower(str_replace(' ', '-', name)) on create and on update.
func normaliseTagName(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}
