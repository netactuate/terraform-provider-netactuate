package netactuate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNKELBServices() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKELBServicesRead,
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The NKE cluster ID",
			},
			"services": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Managed load-balancer services (Services/Ingresses) reported by the VPC load balancer add-on (netactuate-cloud-lb)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"lb_service_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "LB service ID",
						},
						"cluster_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "NKE cluster ID",
						},
						"kind": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The Kubernetes object kind: service or ingress",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Kubernetes namespace of the Service/Ingress",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Kubernetes object name",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Protocol (L4 services only)",
						},
						"ports": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw ports JSON",
						},
						"vip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The floating IP (VIP) assigned to this service, once ready",
						},
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Service state, derived from the operator heartbeat and gateway reconcile status",
						},
						"failure_reason": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Failure reason, when state is not ready",
						},
						"source_ranges": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw sourceRanges JSON",
						},
						"tls_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "TLS mode (L7 services only)",
						},
						"hostnames": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw hostnames JSON (L7 services only)",
						},
						"health": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw health object JSON",
						},
						"timestamps": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Raw timestamps object JSON",
						},
					},
				},
			},
		},
	}
}

func dataSourceNKELBServicesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	services, err := c.ListClusterLBServices(clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceList := make([]map[string]interface{}, 0, len(services))
	for _, svc := range services {
		serviceList = append(serviceList, map[string]interface{}{
			"lb_service_id":  svc.LBServiceID,
			"cluster_id":     svc.ClusterID,
			"kind":           svc.Kind,
			"namespace":      svc.Namespace,
			"name":           svc.Name,
			"protocol":       svc.Protocol,
			"ports":          jsonListOrEmpty(svc.Ports, len(svc.Ports)),
			"vip":            svc.VIP,
			"state":          svc.State,
			"failure_reason": svc.FailureReason,
			"source_ranges":  jsonListOrEmpty(svc.SourceRanges, len(svc.SourceRanges)),
			"tls_mode":       svc.TLSMode,
			"hostnames":      jsonListOrEmpty(svc.Hostnames, len(svc.Hostnames)),
			"health":         rawJSONOrEmpty(svc.Health),
			"timestamps":     rawJSONOrEmpty(svc.Timestamps),
		})
	}

	var diags diag.Diagnostics
	setValue("services", serviceList, d, &diags)
	d.SetId(fmt.Sprintf("nke-lb-services-%d", clusterID))
	return diags
}

// jsonListOrEmpty returns a list as a JSON string, or "" when it is empty.
func jsonListOrEmpty(list interface{}, n int) string {
	if n == 0 {
		return ""
	}
	b, err := json.Marshal(list)
	if err != nil {
		return ""
	}
	return string(b)
}

// rawJSONOrEmpty returns raw JSON as a string, with a JSON null or a missing value as "".
func rawJSONOrEmpty(raw []byte) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	return string(raw)
}
