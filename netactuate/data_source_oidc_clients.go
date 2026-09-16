package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOIDCClients() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientsRead,
		Schema: map[string]*schema.Schema{
			"clients": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "OIDC clients visible to the account",
				Elem: &schema.Resource{
					Schema: oidcClientListElementSchema(),
				},
			},
		},
	}
}

func dataSourceOIDCClientsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clients, err := c.GetOIDCClients()
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := make([]map[string]interface{}, len(clients))
	for i, client := range clients {
		flattened[i] = flattenOIDCClient(client)
	}

	var diags diag.Diagnostics
	setValue("clients", flattened, d, &diags)
	d.SetId("oidc-clients")
	return diags
}
