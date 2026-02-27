package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFTunnel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFTunnelCreate,
		ReadContext:   resourceRouterVRFTunnelRead,
		UpdateContext: resourceRouterVRFTunnelUpdate,
		DeleteContext: resourceRouterVRFTunnelDelete,
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
			"tunnel_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the tunnel.",
			},
			"ip_key": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The GRE key to associate with the tunnel.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name your tunnel something you'll use to refer to it in other configurations.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the tunnel.",
			},
			"mtu": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "The MTU for the tunnel.",
				ValidateFunc: validation.IntBetween(68, 16000),
			},
			"ipv4_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An IPv4 address to apply to the tunnel interface.",
			},
			"ipv6_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An IPv6 address to apply to the tunnel interface.",
			},
			"endpoint_address_remote": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The remote (destination) IP address for the tunnel endpoint.",
			},
			"endpoint_address_source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The source IP address for the tunnel endpoint.",
			},
			"ip_version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The IP version of the tunnel.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			ipv4CIDR := d.Get("ipv4_cidr").(string)
			ipv6CIDR := d.Get("ipv6_cidr").(string)

			if ipv4CIDR == "" && ipv6CIDR == "" {
				return fmt.Errorf("at least one of ipv4_cidr or ipv6_cidr must be set")
			}

			return nil
		},
	}
}

func resourceRouterVRFTunnelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	ipKey := d.Get("ip_key").(int)
	name := d.Get("name").(string)
	mtu := d.Get("mtu").(int)
	endpointRemote := d.Get("endpoint_address_remote").(string)

	unlock := lockRouter(routerID)
	defer unlock()

	createRequest := gona.CreateRouterVRFTunnelRequest{
		IPKey: ipKey,
		Name:  name,
		MTU:   mtu,
		EndpointAddress: gona.CreateRouterVRFTunnelEndpoint{
			Remote: endpointRemote,
		},
	}

	if v, ok := d.GetOk("description"); ok {
		description := v.(string)
		createRequest.Description = &description
	}

	if v, ok := d.GetOk("ipv4_cidr"); ok {
		ipv4 := v.(string)
		createRequest.IPv4CIDR = &ipv4
	}

	if v, ok := d.GetOk("ipv6_cidr"); ok {
		ipv6 := v.(string)
		createRequest.IPv6CIDR = &ipv6
	}

	tunnel, err := c.CreateRouterVRFTunnel(routerID, vrfID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(tunnel.TunnelID))

	return resourceRouterVRFTunnelRead(ctx, d, m)
}

func resourceRouterVRFTunnelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	tunnelID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tunnel, err := c.GetRouterVRFTunnel(routerID, vrfID, tunnelID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	mtu, err := strconv.Atoi(tunnel.MTU)
	if err != nil {
		return diag.FromErr(err)
	}
	setValue("mtu", mtu, d, &diags)
	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("tunnel_id", tunnel.TunnelID, d, &diags)
	setValue("ip_key", tunnel.IPKey, d, &diags)
	setValue("name", tunnel.Name, d, &diags)
	setValue("description", tunnel.Description, d, &diags)
	setValue("ipv4_cidr", tunnel.IPv4CIDR, d, &diags)
	setValue("ipv6_cidr", tunnel.IPv6CIDR, d, &diags)
	setValue("ip_version", tunnel.IPVersion, d, &diags)
	setValue("endpoint_address_remote", tunnel.EndpointAddress.Remote, d, &diags)
	setValue("endpoint_address_source", tunnel.EndpointAddress.Source, d, &diags)


	return diags
}

func resourceRouterVRFTunnelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	tunnelID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ipKey := d.Get("ip_key").(int)
	name := d.Get("name").(string)
	mtu := d.Get("mtu").(int)
	endpointRemote := d.Get("endpoint_address_remote").(string)

	updateRequest := gona.UpdateRouterVRFTunnelRequest{
		IPKey: ipKey,
		Name:  name,
		MTU:   mtu,
		EndpointAddress: gona.UpdateRouterVRFTunnelEndpoint{
			Remote: endpointRemote,
		},
	}

	if v, ok := d.GetOk("description"); ok {
		description := v.(string)
		updateRequest.Description = &description
	}

	if v, ok := d.GetOk("ipv4_cidr"); ok {
		ipv4 := v.(string)
		updateRequest.IPv4CIDR = &ipv4
	}

	if v, ok := d.GetOk("ipv6_cidr"); ok {
		ipv6 := v.(string)
		updateRequest.IPv6CIDR = &ipv6
	}

	_, err = c.UpdateRouterVRFTunnel(routerID, vrfID, tunnelID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterVRFTunnelRead(ctx, d, m)
}

func resourceRouterVRFTunnelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	tunnelID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteRouterVRFTunnel(routerID, vrfID, tunnelID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}