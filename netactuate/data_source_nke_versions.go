package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)
//Terraform versions, just need if we want to have output versions, we don't modifying this with requests
func dataSourceNKEVersions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKEVersionsRead,
		Schema: map[string]*schema.Schema{
			"versions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available Kubernetes versions for NKE clusters",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceNKEVersionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	versions, err := c.ListNKEVersions()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("versions", versions, d, &diags)
	d.SetId("nke-versions")

	return diags
}
