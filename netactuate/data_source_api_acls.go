package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAPIACLs() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the account's API key allowlist. Entries the platform added for a resource " +
			"(a VPC created with allow_api_access, an NKE DNS add-on) list those resources; " +
			"entries added by hand list none.",
		ReadContext: dataSourceAPIACLsRead,
		Schema: map[string]*schema.Schema{
			"acls": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "API key allowlist entries.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Entry ID.",
						},
						"type": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Entry type.",
						},
						"start_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "First address of the range.",
						},
						"end_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Last address of the range.",
						},
						"bits": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Prefix length.",
						},
						"sub_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Team member the entry belongs to; 0 for the account.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Entry description.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the entry was created.",
						},
						"editable": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the caller may change the entry.",
						},
						"resources": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Platform resources holding the entry; empty for entries added by hand.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "vpc or nke_cluster.",
									},
									"resource_id": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Resource ID.",
									},
									"resource_label": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Resource label.",
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

func dataSourceAPIACLsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	acls, err := c.ListAPIACLs(nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	aclList := make([]map[string]interface{}, len(acls))
	for i, acl := range acls {
		resources := make([]map[string]interface{}, len(acl.Resources))
		for j, r := range acl.Resources {
			resources[j] = map[string]interface{}{
				"resource_type":  r.ResourceType,
				"resource_id":    r.ResourceID,
				"resource_label": r.ResourceLabel,
			}
		}
		aclList[i] = map[string]interface{}{
			"id":          acl.ID,
			"type":        acl.Type,
			"start_ip":    acl.StartIP,
			"end_ip":      acl.EndIP,
			"bits":        acl.Bits,
			"sub_id":      acl.SubID,
			"description": acl.Description,
			"created_at":  acl.CreatedAt,
			"editable":    acl.Editable,
			"resources":   resources,
		}
	}

	setValue("acls", aclList, d, &diags)
	d.SetId("api-acls")

	return diags
}
