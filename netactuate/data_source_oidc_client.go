package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOIDCClient() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientRead,
		Schema:      oidcClientDataSourceSchema(),
	}
}

func dataSourceOIDCClientRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	var clientID int
	if v, ok := d.GetOk("oidc_client_id"); ok {
		clientID = v.(int)
	} else {
		label := d.Get("label").(string)
		clients, err := c.GetOIDCClients()
		if err != nil {
			return diag.FromErr(err)
		}
		for _, client := range clients {
			if client.Label == label {
				if clientID != 0 {
					return diag.Errorf("more than one OIDC client found with label %q", label)
				}
				clientID = client.ClientID
			}
		}
		if clientID == 0 {
			return diag.Errorf("no OIDC client found with label %q", label)
		}
	}

	client, err := c.GetOIDCClient(clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for key, value := range flattenOIDCClient(*client) {
		setValue(key, value, d, &diags)
	}
	d.SetId(strconv.Itoa(client.ClientID))
	return diags
}
