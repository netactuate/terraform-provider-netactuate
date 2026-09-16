package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOIDCClientKeys() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientKeysRead,
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "OIDC client ID",
			},
			"keys": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Keys attached to the OIDC client",
				Elem: &schema.Resource{
					Schema: oidcKeyElementSchema(),
				},
			},
		},
	}
}

func dataSourceOIDCClientKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID := d.Get("oidc_client_id").(int)
	keys, err := c.GetOIDCClientKeys(clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		flattened[i] = flattenOIDCKey(key)
	}

	var diags diag.Diagnostics
	setValue("keys", flattened, d, &diags)
	d.SetId("oidc-client-keys-" + strconv.Itoa(clientID))
	return diags
}
