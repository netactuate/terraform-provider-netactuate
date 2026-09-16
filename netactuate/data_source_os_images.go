package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOSImages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOSImagesRead,
		Schema: map[string]*schema.Schema{
			"os_images": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available operating system images",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Operating system image ID",
						},
						"os": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image name",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image type",
						},
						"subtype": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image subtype",
						},
						"size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image size",
						},
						"bits": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image architecture bits",
						},
						"tech": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image virtualization technology",
						},
					},
				},
			},
		},
	}
}

func dataSourceOSImagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	osImages, err := c.GetOSs()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	osImagesList := make([]map[string]interface{}, len(osImages))
	for i, osImage := range osImages {
		osImagesList[i] = map[string]interface{}{
			"id":      osImage.ID,
			"os":      osImage.Os,
			"type":    osImage.Type,
			"subtype": osImage.Subtype,
			"size":    osImage.Size,
			"bits":    osImage.Bits,
			"tech":    osImage.Tech,
		}
	}

	setValue("os_images", osImagesList, d, &diags)
	d.SetId("os-images")

	return diags
}
