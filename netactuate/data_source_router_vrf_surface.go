package netactuate

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudRoutingRouterVRFBGPNeighbors() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterVRFBGPNeighborsRead,
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"vrf_id":    {Type: schema.TypeInt, Required: true, Description: "VRF ID."},
			"neighbors": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "BGP neighbors configured in the VRF.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"neighbor_id":    {Type: schema.TypeInt, Computed: true, Description: "BGP neighbor ID."},
					"name":           {Type: schema.TypeString, Computed: true, Description: "BGP neighbor name."},
					"description":    {Type: schema.TypeString, Computed: true, Description: "BGP neighbor description."},
					"address":        {Type: schema.TypeString, Computed: true, Description: "BGP neighbor address."},
					"remote_asn":     {Type: schema.TypeInt, Computed: true, Description: "Remote ASN."},
					"ipv4_enabled":   {Type: schema.TypeBool, Computed: true, Description: "Whether IPv4 is enabled for this neighbor."},
					"ipv6_enabled":   {Type: schema.TypeBool, Computed: true, Description: "Whether IPv6 is enabled for this neighbor."},
					"is_shutdown":    {Type: schema.TypeBool, Computed: true, Description: "Whether the neighbor is administratively shut down."},
					"source_address": {Type: schema.TypeString, Computed: true, Description: "Configured source address."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingRouterVRFBGPNeighborsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	neighbors, err := m.(*ProviderClients).V3.ListRouterVRFBGPNeighbors(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(neighbors))
	for i, neighbor := range neighbors {
		source := ""
		if neighbor.Source != nil {
			source = neighbor.Source.Address
		}
		out[i] = map[string]interface{}{
			"neighbor_id":    neighbor.NeighborID,
			"name":           neighbor.Name,
			"description":    neighbor.Description,
			"address":        neighbor.Address,
			"remote_asn":     neighbor.ASN.Remote,
			"ipv4_enabled":   neighbor.EnabledIPVersion.IPv4,
			"ipv6_enabled":   neighbor.EnabledIPVersion.IPv6,
			"is_shutdown":    neighbor.IsShutdown,
			"source_address": source,
		}
	}
	var diags diag.Diagnostics
	setValue("neighbors", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-vrf-%d-bgp-neighbors", routerID, vrfID))
	return diags
}

func dataSourceCloudRoutingRouterVRFInterfaces() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterVRFInterfacesRead,
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"vrf_id":    {Type: schema.TypeInt, Required: true, Description: "VRF ID."},
			"interfaces": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Interfaces configured in the VRF.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"interface_id":         {Type: schema.TypeInt, Computed: true, Description: "Interface ID."},
					"type":                 {Type: schema.TypeString, Computed: true, Description: "Interface type."},
					"name":                 {Type: schema.TypeString, Computed: true, Description: "Interface name."},
					"description":          {Type: schema.TypeString, Computed: true, Description: "Interface description."},
					"ipv4_cidr":            {Type: schema.TypeString, Computed: true, Description: "Interface IPv4 CIDR."},
					"ipv6_cidr":            {Type: schema.TypeString, Computed: true, Description: "Interface IPv6 CIDR."},
					"ethernet_hardware_id": {Type: schema.TypeString, Computed: true, Description: "Ethernet hardware ID."},
					"wireguard_port":       {Type: schema.TypeInt, Computed: true, Description: "WireGuard port, or 0 when absent."},
					"public_key":           {Type: schema.TypeString, Computed: true, Description: "WireGuard public key."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingRouterVRFInterfacesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	resp, err := m.(*ProviderClients).V3.ListRouterVRFInterfaces(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}
	keys := make([]string, 0, len(resp))
	for key := range resp {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		iface := resp[key]
		description := ""
		if iface.Description != nil {
			description = *iface.Description
		}
		ipv4 := ""
		if iface.IPv4CIDR != nil {
			ipv4 = *iface.IPv4CIDR
		}
		ipv6 := ""
		if iface.IPv6CIDR != nil {
			ipv6 = *iface.IPv6CIDR
		}
		ethernetHardwareID := ""
		if iface.EthernetHardwareID != nil {
			ethernetHardwareID = *iface.EthernetHardwareID
		}
		wireguardPort := 0
		if iface.WireguardPort != nil {
			wireguardPort = *iface.WireguardPort
		}
		publicKey := ""
		if iface.PublicKey != nil {
			publicKey = *iface.PublicKey
		}
		out[i] = map[string]interface{}{
			"interface_id":         iface.InterfaceID,
			"type":                 iface.Type,
			"name":                 iface.Name,
			"description":          description,
			"ipv4_cidr":            ipv4,
			"ipv6_cidr":            ipv6,
			"ethernet_hardware_id": ethernetHardwareID,
			"wireguard_port":       wireguardPort,
			"public_key":           publicKey,
		}
	}
	var diags diag.Diagnostics
	setValue("interfaces", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-vrf-%d-interfaces", routerID, vrfID))
	return diags
}

func dataSourceCloudRoutingRouterVRFTunnels() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterVRFTunnelsRead,
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"vrf_id":    {Type: schema.TypeInt, Required: true, Description: "VRF ID."},
			"tunnels": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Tunnels configured in the VRF.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"tunnel_id":               {Type: schema.TypeInt, Computed: true, Description: "Tunnel ID."},
					"name":                    {Type: schema.TypeString, Computed: true, Description: "Tunnel name."},
					"description":             {Type: schema.TypeString, Computed: true, Description: "Tunnel description."},
					"ip_key":                  {Type: schema.TypeInt, Computed: true, Description: "GRE key."},
					"mtu":                     {Type: schema.TypeString, Computed: true, Description: "Tunnel MTU."},
					"ipv4_cidr":               {Type: schema.TypeString, Computed: true, Description: "Tunnel IPv4 CIDR."},
					"ipv6_cidr":               {Type: schema.TypeString, Computed: true, Description: "Tunnel IPv6 CIDR."},
					"endpoint_address_remote": {Type: schema.TypeString, Computed: true, Description: "Remote endpoint address."},
					"endpoint_address_source": {Type: schema.TypeString, Computed: true, Description: "Source endpoint address."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingRouterVRFTunnelsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	tunnels, err := m.(*ProviderClients).V3.ListRouterVRFTunnels(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(tunnels))
	for i, tunnel := range tunnels {
		description := ""
		if tunnel.Description != nil {
			description = *tunnel.Description
		}
		ipv4 := ""
		if tunnel.IPv4CIDR != nil {
			ipv4 = *tunnel.IPv4CIDR
		}
		ipv6 := ""
		if tunnel.IPv6CIDR != nil {
			ipv6 = *tunnel.IPv6CIDR
		}
		out[i] = map[string]interface{}{
			"tunnel_id":               tunnel.TunnelID,
			"name":                    tunnel.Name,
			"description":             description,
			"ip_key":                  tunnel.IPKey,
			"mtu":                     tunnel.MTU,
			"ipv4_cidr":               ipv4,
			"ipv6_cidr":               ipv6,
			"endpoint_address_remote": tunnel.EndpointAddress.Remote,
			"endpoint_address_source": tunnel.EndpointAddress.Source,
		}
	}
	var diags diag.Diagnostics
	setValue("tunnels", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-vrf-%d-tunnels", routerID, vrfID))
	return diags
}

func dataSourceCloudRoutingRouterRoutingOverview() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterRoutingOverviewRead,
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"vrf_id":    {Type: schema.TypeInt, Required: true, Description: "VRF ID."},
			"overview":  rawJSONDataSourceSchema("Routing overview for the VRF as compact JSON."),
		},
	}
}

func dataSourceCloudRoutingRouterRoutingOverviewRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	raw, err := m.(*ProviderClients).V3.GetRouterRoutingOverview(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}
	out, err := compactSurfaceJSON(raw)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("overview", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-vrf-%d-routing-overview", routerID, vrfID))
	return diags
}
