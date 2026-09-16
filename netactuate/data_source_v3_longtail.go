package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceHTTPLoadbalancerGroups() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceHTTPLoadbalancerGroupsRead,
		Schema: map[string]*schema.Schema{
			"http_loadbalancer_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "HTTP load balancer ID whose groups will be listed.",
			},
			"groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "HTTP load balancer groups returned by the API.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"http_group_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Group ID assigned by the API.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Group name returned by the API.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Group description returned by the API.",
						},
						"algorithm": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Load balancing algorithm returned by the API.",
						},
						"sticky_sessions_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether sticky sessions are enabled.",
						},
						"ssl_to_backend_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the group uses SSL when connecting to backends.",
						},
						"internal_port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Backend port returned by the API.",
						},
						"is_online": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the group is online.",
						},
						"match_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Public IP address matched by this group.",
						},
						"match_ports": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Public ports matched by this group.",
						},
					},
				},
			},
		},
	}
}

func dataSourceHTTPLoadbalancerGroupsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	lbID := d.Get("http_loadbalancer_id").(int)
	groups, err := m.(*ProviderClients).V3.GetHTTPLBGroups(lbID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("groups", flattenHTTPLBGroupRows(groups), d, &diags)
	d.SetId("http-loadbalancer-groups-" + strconv.Itoa(lbID))
	return diags
}

func flattenHTTPLBGroupRows(groups []gona.HTTPLBGroup) []map[string]interface{} {
	out := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		out[i] = map[string]interface{}{
			"http_group_id":           group.HTTPGroupID,
			"name":                    group.Name,
			"description":             group.Description,
			"algorithm":               group.Algorithm,
			"sticky_sessions_enabled": group.StickySessionsEnabled,
			"ssl_to_backend_enabled":  group.SSLToBackendEnabled,
			"internal_port":           group.InternalPort,
			"is_online":               group.IsOnline,
			"match_address":           group.Match.Address,
			"match_ports":             group.Match.Ports,
		}
	}
	return out
}

func dataSourceSSLCertificates() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSSLCertificatesRead,
		Schema: map[string]*schema.Schema{
			"certificates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "SSL certificates visible to the account.",
				Elem: &schema.Resource{
					Schema: sslCertificateDataSourceSchema(),
				},
			},
		},
	}
}

func sslCertificateDataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"ssl_certificate_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "SSL certificate ID assigned by the API.",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "SSL certificate name returned by the API.",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "SSL certificate description returned by the API.",
		},
		"fingerprint": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate fingerprint returned by the API.",
		},
		"domains": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Domains covered by the certificate.",
			Elem: &schema.Schema{
				Type:        schema.TypeString,
				Description: "Domain name.",
			},
		},
		"is_active": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the certificate is active.",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate status returned by the API.",
		},
		"created": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate creation timestamp returned by the API.",
		},
		"updated": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate update timestamp returned by the API.",
		},
		"not_before": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate validity start timestamp returned by the API.",
		},
		"expiration": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Certificate expiration timestamp returned by the API.",
		},
	}
}

func dataSourceSSLCertificatesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	certs, err := m.(*ProviderClients).V3.GetSSLCertificates()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("certificates", flattenSSLCertificates(certs), d, &diags)
	d.SetId("ssl-certificates")
	return diags
}

func flattenSSLCertificates(certs []gona.SSLCertificate) []map[string]interface{} {
	out := make([]map[string]interface{}, len(certs))
	for i, cert := range certs {
		row := map[string]interface{}{
			"ssl_certificate_id": cert.SSLCertificateID,
			"name":               cert.Name,
			"description":        cert.Description,
			"fingerprint":        cert.Fingerprint,
			"domains":            cert.Domains,
			"is_active":          cert.IsActive,
			"status":             cert.Status,
			"created":            "",
			"updated":            "",
			"not_before":         "",
			"expiration":         "",
		}
		if cert.Dates != nil {
			row["created"] = cert.Dates.Created
			row["updated"] = cert.Dates.Updated
			row["not_before"] = cert.Dates.NotBefore
			row["expiration"] = cert.Dates.Expiration
		}
		out[i] = row
	}
	return out
}
