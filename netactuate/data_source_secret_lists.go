package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSecretLists() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecretListsRead,
		Schema: map[string]*schema.Schema{
			"secret_lists": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of secret lists for the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Secret list ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Secret list name",
						},
					},
				},
			},
		},
	}
}

func dataSourceSecretListsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	secretLists, err := c.GetSecretLists()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	secretListsList := make([]map[string]interface{}, len(secretLists))
	for i, secretList := range secretLists {
		secretListsList[i] = map[string]interface{}{
			"id":   secretList.ID,
			"name": secretList.Name,
		}
	}

	setValue("secret_lists", secretListsList, d, &diags)
	d.SetId("secret-lists")

	return diags
}
