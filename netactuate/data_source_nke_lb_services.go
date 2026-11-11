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
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Ports on the floating IP and the node ports they forward to",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"port": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Port on the floating IP",
									},
									"node_port": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Node port it forwards to",
									},
								},
							},
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
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Source CIDRs allowed to reach the service",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"tls_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "TLS mode (L7 services only)",
						},
						"hostnames": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Hostnames (L7 services only)",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"last_heartbeat_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the add-on last reported",
						},
						"requested_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the service was requested",
						},
						"installed_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the service was installed",
						},
						"ready_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the service became ready",
						},
						"created_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "When the service was created",
						},
					},
				},
			},
		},
	}
}

type nkeLBServiceHealth struct {
	LastHeartbeatOn string `json:"lastHeartbeatOn"`
}

type nkeLBServiceTimestamps struct {
	RequestedOn string `json:"requestedOn"`
	InstalledOn string `json:"installedOn"`
	ReadyOn     string `json:"readyOn"`
	CreatedOn   string `json:"createdOn"`
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
		ports := make([]map[string]interface{}, 0, len(svc.Ports))
		for _, p := range svc.Ports {
			ports = append(ports, map[string]interface{}{"port": p.Port, "node_port": p.NodePort})
		}

		var health nkeLBServiceHealth
		var ts nkeLBServiceTimestamps
		if len(svc.Health) > 0 {
			_ = json.Unmarshal(svc.Health, &health)
		}
		if len(svc.Timestamps) > 0 {
			_ = json.Unmarshal(svc.Timestamps, &ts)
		}

		serviceList = append(serviceList, map[string]interface{}{
			"lb_service_id":     svc.LBServiceID,
			"cluster_id":        svc.ClusterID,
			"kind":              svc.Kind,
			"namespace":         svc.Namespace,
			"name":              svc.Name,
			"protocol":          svc.Protocol,
			"ports":             ports,
			"vip":               svc.VIP,
			"state":             svc.State,
			"failure_reason":    svc.FailureReason,
			"source_ranges":     svc.SourceRanges,
			"tls_mode":          svc.TLSMode,
			"hostnames":         svc.Hostnames,
			"last_heartbeat_on": health.LastHeartbeatOn,
			"requested_on":      ts.RequestedOn,
			"installed_on":      ts.InstalledOn,
			"ready_on":          ts.ReadyOn,
			"created_on":        ts.CreatedOn,
		})
	}

	var diags diag.Diagnostics
	setValue("services", serviceList, d, &diags)
	d.SetId(fmt.Sprintf("nke-lb-services-%d", clusterID))
	return diags
}
