package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

type ProviderClients struct {
	V2 *gona.Client
	V3 *gona.V3Client
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETACTUATE_API_KEY", nil),
			},
			"api_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"api_url_v3": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Base URL for the vAPI3 endpoint. Defaults to https://vapi3.netactuate.com",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"netactuate_server":       resourceServer(),
			"netactuate_sshkey":       resourceSshKey(),
			"netactuate_bgp_sessions": resourceBGPSessions(),
			"netactuate_metal":        resourceMetal(),
			"netactuate_vpc":                        resourceVPC(),
			"netactuate_vpc_gateway_dnat_rule":       resourceVPCGatewayDNATRule(),
			"netactuate_vpc_gateway_snat_rule":       resourceVPCGatewaySNATRule(),
			"netactuate_vpc_gateway_firewall_rule":   resourceVPCGatewayFirewallRule(),
			"netactuate_vpc_floating_ip":             resourceVPCFloatingIP(),
			"netactuate_vpc_ssh_key":                 resourceVPCSSHKey(),
			"netactuate_vpc_backend_template":        resourceVPCBackendTemplate(),
			"netactuate_network_loadbalancer_group":  resourceNetworkLoadbalancerGroup(),
			"netactuate_ssl_certificate":             resourceSSLCertificate(),
			"netactuate_http_loadbalancer_group":     resourceHTTPLoadbalancerGroup(),
			"netactuate_nke_cluster":                 resourceNKECluster(),
			"netactuate_storage_bucket":              resourceStorageBucket(),
			"netactuate_storage_object_store":        resourceStorageObjectStore(),
			"netactuate_storage_block_namespace":     resourceStorageBlockNamespace(),
			"netactuate_storage_block_volume":        resourceStorageBlockVolume(),
			"netactuate_secret_list":                 resourceSecretList(),
			"netactuate_secret_list_value":           resourceSecretListValue(),
			"netactuate_router":                      resourceRouter(),
			"netactuate_router_vrf":                  resourceRouterVRF(),
			"netactuate_router_vrf_interface": 		  resourceRouterVRFInterface(),
			"netactuate_router_vrf_bgp":			  resourceRouterVRFBGP(),
			"netactuate_router_vrf_bgp_neighbor": 	  resourceRouterVRFBGPNeighbor(),

		},
		DataSourcesMap: map[string]*schema.Resource{
			"netactuate_server":           dataSourceServer(),
			"netactuate_sshkey":           dataSourceSshKey(),
			"netactuate_bgp_sessions":     dataSourceBGPSessions(),
			"netactuate_nke_versions":     dataSourceNKEVersions(),
			"netactuate_nke_worker_nodes": dataSourceNKEWorkerNodes(),
			"netactuate_nke_kubeconfig":   dataSourceNKEKubeconfig(),
			"netactuate_storage_locations": dataSourceStorageLocations(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	apiKey := d.Get("api_key").(string)
	apiUrl := d.Get("api_url").(string)
	apiUrlV3 := d.Get("api_url_v3").(string)

	if apiKey == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create NetActuate API client",
			Detail: `Unable to find NetActuate API key. It can be set with either NETACTUATE_API_KEY environment
variable or 'api_key' property`,
		})
		return nil, diags
	}

	var v2Client *gona.Client
	if apiUrl == "" {
		v2Client = gona.NewClient(apiKey)
	} else {
		v2Client = gona.NewClientCustom(apiKey, apiUrl)
	}

	v3Client := gona.NewV3Client(apiKey, apiUrlV3)

	return &ProviderClients{
		V2: v2Client,
		V3: v3Client,
	}, nil
}
