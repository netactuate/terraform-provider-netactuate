package netactuate

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceImage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceImageCreate,
		ReadContext:   resourceImageRead,
		UpdateContext: resourceImageUpdate,
		DeleteContext: resourceImageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name for the custom image",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the custom image",
			},
			"server_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Source server ID (mbpkgid) to create the image from",
			},
			"keep_ssh_userdirs": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Whether to preserve SSH user directories in the image",
			},
			"os_enabled": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Image status: 1=enabled, 0=disabled",
			},
			"bits": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Architecture (32 or 64)",
			},
			"category": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OS category (e.g. ubuntu, centos)",
			},
			"image_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image type",
			},
			"subtype": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image subtype",
			},
			"size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image size",
			},
			"tech": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Virtualization technology",
			},
			"script_bash": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the image supports bash scripts",
			},
			"script_cloudinit": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the image supports cloud-init",
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
		},
	}
}

func resourceImageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.CreateImageRequest{
		MbPkgID:          d.Get("server_id").(int),
		ImageName:        d.Get("name").(string),
		ImageDescription: d.Get("description").(string),
		KeepSSHUserdirs:  d.Get("keep_ssh_userdirs").(bool),
	}

	resp, err := c.CreateImage(req)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Image creation job started with queue_id: %d", resp.QueueID)

	status, err := c.WaitForImageQueue(resp.QueueID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(status.ImageID))

	return resourceImageRead(ctx, d, m)
}

func resourceImageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	image, err := c.GetImage(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("name", image.Name, d, &diags)
	if image.Description != nil {
		setValue("description", *image.Description, d, &diags)
	} else {
		setValue("description", "", d, &diags)
	}
	if image.Enabled != nil {
		setValue("os_enabled", *image.Enabled, d, &diags)
	}
	setValue("bits", image.Bits, d, &diags)
	setValue("category", image.Category, d, &diags)
	setValue("image_type", image.Type, d, &diags)
	setValue("subtype", image.Subtype, d, &diags)
	setValue("size", image.Size, d, &diags)
	setValue("tech", image.Tech, d, &diags)
	setValue("script_bash", image.ScriptBash == 1, d, &diags)
	setValue("script_cloudinit", image.ScriptCloudinit == 1, d, &diags)
	setValue("created", image.Created, d, &diags)

	return diags
}

func resourceImageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	if d.HasChange("name") || d.HasChange("description") {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		name := d.Get("name").(string)
		description := d.Get("description").(string)

		if err := c.EditImage(id, name, description); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceImageRead(ctx, d, m)
}

func resourceImageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.DeleteImage(id)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Image deletion job started with queue_id: %d", resp.QueueID)

	if _, err := c.WaitForImageQueue(resp.QueueID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
