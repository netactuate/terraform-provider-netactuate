package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceMyImages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMyImagesRead,
		Schema: map[string]*schema.Schema{
			"images": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of account-owned images",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Image ID",
						},
						"os": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image name",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image description, or empty string when the API returns null",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image type",
						},
						"subtype": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image subtype",
						},
						"bits": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image architecture bits",
						},
						"tech": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image virtualization technology",
						},
						"size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image size",
						},
						"category": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image category",
						},
						"os_enabled": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Image enabled flag, or 0 when the API returns null",
						},
						"script_bash": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Bash script support flag",
						},
						"script_cloudinit": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Cloud-init support flag",
						},
						"created": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image creation timestamp",
						},
						"updated": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image update timestamp",
						},
						"active_build": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Active image build job, when one exists",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Build job ID",
									},
									"status": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Build job status",
									},
									"command": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job command",
									},
									"ts_insert": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job insert timestamp",
									},
									"mb_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Account ID for the build job",
									},
									"mb_pkgid": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Server package ID for the build job",
									},
									"params": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job parameters",
									},
									"build_packet": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job packet",
									},
									"response": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job response",
									},
									"created": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job creation timestamp",
									},
									"last_updated": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Build job last update timestamp",
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

func dataSourceMyImagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	images, err := c.GetMyImages()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	imagesList := make([]map[string]interface{}, len(images))
	for i, image := range images {
		description := ""
		if image.Description != nil {
			description = *image.Description
		}
		enabled := 0
		if image.Enabled != nil {
			enabled = *image.Enabled
		}

		activeBuild := []map[string]interface{}{}
		if image.ActiveBuild != nil {
			activeBuild = append(activeBuild, map[string]interface{}{
				"id":           image.ActiveBuild.ID,
				"status":       image.ActiveBuild.Status,
				"command":      image.ActiveBuild.Command,
				"ts_insert":    image.ActiveBuild.TSInsert,
				"mb_id":        image.ActiveBuild.MbID,
				"mb_pkgid":     image.ActiveBuild.MbPkgID,
				"params":       image.ActiveBuild.Params,
				"build_packet": image.ActiveBuild.BuildPacket,
				"response":     image.ActiveBuild.Response,
				"created":      image.ActiveBuild.Created,
				"last_updated": image.ActiveBuild.LastUpdated,
			})
		}

		imagesList[i] = map[string]interface{}{
			"id":               image.ID,
			"os":               image.Name,
			"description":      description,
			"type":             image.Type,
			"subtype":          image.Subtype,
			"bits":             image.Bits,
			"tech":             image.Tech,
			"size":             image.Size,
			"category":         image.Category,
			"os_enabled":       enabled,
			"script_bash":      image.ScriptBash,
			"script_cloudinit": image.ScriptCloudinit,
			"created":          image.Created,
			"updated":          image.Updated,
			"active_build":     activeBuild,
		}
	}

	setValue("images", imagesList, d, &diags)
	d.SetId("my-images")

	return diags
}
