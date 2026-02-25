package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImageRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Image ID",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image description",
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
				Description: "OS category",
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

func dataSourceImageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	image, err := c.GetImage(d.Get("id").(int))
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

	if diags == nil {
		d.SetId(strconv.Itoa(image.ID))
	}

	return diags
}
