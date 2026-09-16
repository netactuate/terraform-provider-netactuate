package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPCBackendTemplates() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVPCBackendTemplatesRead,
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VPC ID whose backend templates will be listed",
			},
			"templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backend templates attached to the VPC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"backend_template_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Backend template ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Backend template name",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Backend template description",
						},
						"backend_hosts": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Backend hosts in the backend template",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"backend_host_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Backend host ID",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Backend host name",
									},
									"address": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Backend host address",
									},
									"internal_address": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Backend host internal address",
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

func dataSourceVPCBackendTemplatesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	vpcID := d.Get("vpc_id").(int)

	templates, err := c.ListVPCBackendTemplates(vpcID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	templatesList := make([]map[string]interface{}, len(templates))
	for i, template := range templates {
		backendHostsList := make([]map[string]interface{}, len(template.BackendHosts))
		for j, backendHost := range template.BackendHosts {
			backendHostsList[j] = map[string]interface{}{
				"backend_host_id":  backendHost.BackendHostID,
				"name":             backendHost.Name,
				"address":          backendHost.Address,
				"internal_address": backendHost.InternalAddress,
			}
		}

		templatesList[i] = map[string]interface{}{
			"backend_template_id": template.BackendTemplateID,
			"name":                template.Name,
			"description":         template.Description,
			"backend_hosts":       backendHostsList,
		}
	}

	setValue("templates", templatesList, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-backend-templates", vpcID))

	return diags
}
