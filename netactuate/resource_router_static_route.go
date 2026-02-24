package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterStaticRoute() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterStaticRouteCreate,
		ReadContext:   resourceRouterStaticRouteRead,
		UpdateContext: resourceRouterStaticRouteUpdate,
		DeleteContext: resourceRouterStaticRouteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterStaticRouteImport,
		},
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"vrf_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VRF.",
			},
			"route_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the static route.",
			},
			"network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network to route traffic towards (CIDR notation, e.g. 10.0.0.0/24).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the static route.",
			},
			"distance": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 255),
				Description:  "An optional administrative distance for the route (1-255).",
			},
			"next_hop": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The next-hop IP address to route traffic to (IPv4 or IPv6).",
			},
			"interface_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The ID of the interface to route traffic to.",
			},
			"tunnel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The ID of the tunnel to route traffic to.",
			},
			"ipsec_peer_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The ID of the IPSec peer to route traffic to.",
			},
		},
	}
}

func buildStaticRouteVia(d *schema.ResourceData) gona.StaticRouteVia {
	via := gona.StaticRouteVia{}

	if v, ok := d.GetOk("next_hop"); ok {
		via.NextHop = v.(string)
	}
	if v, ok := d.GetOk("interface_id"); ok {
		intID := v.(int)
		via.InterfaceID = &intID
	}
	if v, ok := d.GetOk("tunnel_id"); ok {
		tunID := v.(int)
		via.TunnelID = &tunID
	}
	if v, ok := d.GetOk("ipsec_peer_id"); ok {
		peerID := v.(int)
		via.IPSecPeerID = &peerID
	}

	return via
}

func resourceRouterStaticRouteCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	createRequest := gona.CreateRouterStaticRouteRequest{
		Network: d.Get("network").(string),
		Via:     buildStaticRouteVia(d),
	}

	if v, ok := d.GetOk("description"); ok {
		createRequest.Description = v.(string)
	}

	if v, ok := d.GetOk("distance"); ok {
		dist := v.(int)
		createRequest.Distance = &dist
	}

	resp, err := c.CreateRouterStaticRoute(routerID, vrfID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d/%d", routerID, vrfID, resp.RouteID))

	var diags diag.Diagnostics
	setValue("route_id", resp.RouteID, d, &diags)

	readDiags := resourceRouterStaticRouteRead(ctx, d, m)
	diags = append(diags, readDiags...)
	return diags
}

func resourceRouterStaticRouteRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, routeID, err := parseStaticRouteID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	route, err := c.GetRouterStaticRoute(routerID, vrfID, routeID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("route_id", route.RouteID, d, &diags)
	setValue("network", route.Network, d, &diags)
	setValue("description", route.Description, d, &diags)

	if route.Distance != nil {
		setValue("distance", *route.Distance, d, &diags)
	}

	if route.Via.NextHop != "" {
		setValue("next_hop", route.Via.NextHop, d, &diags)
	}
	if route.Via.InterfaceID != nil {
		setValue("interface_id", *route.Via.InterfaceID, d, &diags)
	}
	if route.Via.TunnelID != nil {
		setValue("tunnel_id", *route.Via.TunnelID, d, &diags)
	}
	if route.Via.IPSecPeerID != nil {
		setValue("ipsec_peer_id", *route.Via.IPSecPeerID, d, &diags)
	}

	return diags
}

func resourceRouterStaticRouteUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, routeID, err := parseStaticRouteID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := gona.UpdateRouterStaticRouteRequest{
		Network: d.Get("network").(string),
		Via:     buildStaticRouteVia(d),
	}

	if v, ok := d.GetOk("description"); ok {
		updateRequest.Description = v.(string)
	}

	if v, ok := d.GetOk("distance"); ok {
		dist := v.(int)
		updateRequest.Distance = &dist
	}

	_, err = c.UpdateRouterStaticRoute(routerID, vrfID, routeID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterStaticRouteRead(ctx, d, m)
}

func resourceRouterStaticRouteDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, routeID, err := parseStaticRouteID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteRouterStaticRoute(routerID, vrfID, routeID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceRouterStaticRouteImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, vrfID, routeID, err := parseStaticRouteID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"routerId/vrfId/routeId\" (e.g. \"42/1/10\")", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d/%d", routerID, vrfID, routeID))
	d.Set("router_id", routerID)
	d.Set("vrf_id", vrfID)

	return []*schema.ResourceData{d}, nil
}

func parseStaticRouteID(id string) (int, int, int, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid static route ID %q, expected \"routerId/vrfId/routeId\"", id)
	}
	routerID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid router_id %q: %w", parts[0], err)
	}
	vrfID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid vrf_id %q: %w", parts[1], err)
	}
	routeID, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid route_id %q: %w", parts[2], err)
	}
	return routerID, vrfID, routeID, nil
}
