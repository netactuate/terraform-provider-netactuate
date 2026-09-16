package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOIDCClientAuthLogs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientAuthLogsRead,
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "OIDC client ID",
			},
			"logs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Authentication logs for the OIDC client",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"log_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Log ID assigned by the API",
						},
						"issued_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the JWT was issued",
						},
						"expires_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the JWT expires",
						},
						"jti": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "JWT ID",
						},
					},
				},
			},
		},
	}
}

func dataSourceOIDCClientAuthLogsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID := d.Get("oidc_client_id").(int)
	logs, err := c.GetOIDCClientAuthLogs(clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		flattened[i] = flattenOIDCAuthLog(log)
	}

	var diags diag.Diagnostics
	setValue("logs", flattened, d, &diags)
	d.SetId("oidc-client-auth-logs-" + strconv.Itoa(clientID))
	return diags
}

func dataSourceOIDCClientChangeLogs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOIDCClientChangeLogsRead,
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "OIDC client ID",
			},
			"logs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Change logs for the OIDC client",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Key ID for key events",
						},
						"recorded_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the event was recorded",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Event type",
						},
					},
				},
			},
		},
	}
}

func dataSourceOIDCClientChangeLogsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID := d.Get("oidc_client_id").(int)
	logs, err := c.GetOIDCClientChangeLogs(clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		flattened[i] = flattenOIDCChangeLog(log)
	}

	var diags diag.Diagnostics
	setValue("logs", flattened, d, &diags)
	d.SetId("oidc-client-change-logs-" + strconv.Itoa(clientID))
	return diags
}
