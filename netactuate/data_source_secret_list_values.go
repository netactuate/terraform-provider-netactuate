package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceSecretListValues() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecretListValuesRead,
		Schema: map[string]*schema.Schema{
			"secret_list_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Secret list ID whose values will be listed",
			},
			"values": {
				Type:     schema.TypeList,
				Computed: true,
				// The whole list is marked sensitive. SDKv2 drops the nested
				// per-field Sensitive flag on a Computed-only TypeList (it becomes
				// a cty object attribute with no per-field sensitivity), so the
				// secret values would otherwise render in cleartext in plan, show
				// and state. See provider_sensitive_shape_test.go.
				Sensitive:   true,
				Description: "List of values in the secret list",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Secret list value ID",
						},
						"secret_list_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Parent secret list ID",
						},
						"secret_key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Secret key name",
						},
						"secret_value": {
							Type:        schema.TypeString,
							Computed:    true,
							Sensitive:   true,
							Description: "Secret value",
						},
					},
				},
			},
		},
	}
}

func dataSourceSecretValues() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecretValuesRead,
		Schema: map[string]*schema.Schema{
			"values": secretValuesSchema("All secret values visible to the account."),
		},
	}
}

func secretValuesSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		// Marked sensitive at the list level on purpose: a Computed-only TypeList
		// silently discards the nested secret_value Sensitive flag in SDKv2, so the
		// only place the flag survives is here. See provider_sensitive_shape_test.go.
		Sensitive:   true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Secret list value ID",
				},
				"secret_list_id": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Parent secret list ID",
				},
				"secret_key": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Secret key name",
				},
				"secret_value": {
					Type:        schema.TypeString,
					Computed:    true,
					Sensitive:   true,
					Description: "Secret value",
				},
			},
		},
	}
}

func dataSourceSecretListValuesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	listID := d.Get("secret_list_id").(int)

	values, err := c.GetSecretListValues(listID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("values", flattenSecretValues(values), d, &diags)
	d.SetId(fmt.Sprintf("secret-list-values-%d", listID))

	return diags
}

func dataSourceSecretValuesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	values, err := c.GetAllSecretValues()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("values", flattenSecretValues(values), d, &diags)
	d.SetId("secret-values")
	return diags
}

func flattenSecretValues(values []gona.SecretListValue) []map[string]interface{} {
	valuesList := make([]map[string]interface{}, len(values))
	for i, value := range values {
		valuesList[i] = map[string]interface{}{
			"id":             value.ID,
			"secret_list_id": value.SecretListID,
			"secret_key":     value.SecretKey,
			"secret_value":   value.SecretValue,
		}
	}
	return valuesList
}
