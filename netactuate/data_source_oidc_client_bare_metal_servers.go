package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceOIDCClientBareMetalServers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientBareMetalServersRead,
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "OIDC client ID.",
			},
			"servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Bare metal servers allowed to access the OIDC client.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mbpkgid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Bare metal package ID.",
						},
						"raw_json": rawJSONSchema("Complete server row as JSON, preserving API fields not modeled as first class attributes."),
					},
				},
			},
		},
	}
}

func dataSourceOIDCClientBareMetalServersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clientID := d.Get("oidc_client_id").(int)

	servers, err := c.GetOIDCClientBareMetalServers(clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("servers", flattenOIDCBareMetalServers(servers), d, &diags)
	d.SetId("oidc-client-bare-metal-servers-" + strconv.Itoa(clientID))
	return diags
}

func flattenOIDCBareMetalServers(servers []gona.OIDCClientBareMetalServer) []map[string]interface{} {
	out := make([]map[string]interface{}, len(servers))
	for i, server := range servers {
		out[i] = map[string]interface{}{
			"mbpkgid":  server.MBPkgID,
			"raw_json": compactOptionalRawJSON(server.Raw),
		}
	}
	return out
}
