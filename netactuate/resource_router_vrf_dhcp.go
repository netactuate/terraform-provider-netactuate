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

func resourceRouterVRFDHCP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFDHCPUpdate,
		ReadContext:   resourceRouterVRFDHCPRead,
		UpdateContext: resourceRouterVRFDHCPUpdate,
		DeleteContext: resourceRouterVRFDHCPDelete,
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
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether or not DHCP is enabled at all.",
			},
			"interface_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "You may restrict the DHCP service to a specific interface.",
			},
			"subnet": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The subnet for your DHCP service to assign IPs.",
			},
			"lease_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      86400,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The lease timeout in seconds. Defaults to 86400 (24 hours).",
			},
			"do_ping_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If true, prevents giving out an IP address if it already responds to ICMP pings. Defaults to true.",
			},
			"default_router_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The IP address for the default router in your DHCP subnet.",
			},
			"client_domain_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The domain for search suffixes in DHCP.",
			},
			"range": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "The range of IP addresses that can be leased.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"first_address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The first IP address allowed to be leased within the subnet.",
						},
						"last_address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The last IP address allowed to be leased within the subnet.",
						},
					},
				},
			},
			"domain_name_servers": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The DNS servers provided by the DHCP service to clients.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The IP address of the DNS server.",
						},
					},
				},
			},
			"ntp_servers": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The NTP servers provided by the DHCP service to clients.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The IP address of the NTP server.",
						},
					},
				},
			},
			"static_routes": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The static routes provided by the DHCP service to clients.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The network to route traffic towards.",
						},
						"next_hop": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "If you want to route to an IP, you provide it here.",
						},
					},
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterVRFDHCPImport,
		},
	}
}

func resourceRouterVRFDHCPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	if d.Id() != "" {
		var err error
		routerID, vrfID, err = parseRouterVRFDHCPImportID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
	}

	unlock := lockRouter(routerID)
	defer unlock()

	updateRequest := gona.UpdateRouterVRFDHCPRequest{
		Enabled:              d.Get("enabled").(bool),
		Subnet:               d.Get("subnet").(string),
		LeaseTimeout:         d.Get("lease_timeout").(int),
		DoPingCheck:          d.Get("do_ping_check").(bool),
		DefaultRouterAddress: d.Get("default_router_address").(string),
		ClientDomainName:     d.Get("client_domain_name").(string),
		InterfaceID:          d.Get("interface_id").(int),
	}

	if v, ok := d.GetOk("range"); ok {
		rangeList := v.([]interface{})
		if len(rangeList) > 0 {
			rangeMap := rangeList[0].(map[string]interface{})
			updateRequest.Range = &gona.RouterDHCPRange{
				FirstAddress: rangeMap["first_address"].(string),
				LastAddress:  rangeMap["last_address"].(string),
			}
		}
	}

	if v, ok := d.GetOk("domain_name_servers"); ok {
		serversList := v.([]interface{})
		servers := make([]gona.RouterDHCPServer, len(serversList))
		for i, server := range serversList {
			serverMap := server.(map[string]interface{})
			servers[i] = gona.RouterDHCPServer{
				Address: serverMap["address"].(string),
			}
		}
		updateRequest.DomainNameServers = servers
	}

	if v, ok := d.GetOk("ntp_servers"); ok {
		serversList := v.([]interface{})
		servers := make([]gona.RouterDHCPServer, len(serversList))
		for i, server := range serversList {
			serverMap := server.(map[string]interface{})
			servers[i] = gona.RouterDHCPServer{
				Address: serverMap["address"].(string),
			}
		}
		updateRequest.NTPServers = servers
	}

	if v, ok := d.GetOk("static_routes"); ok {
		routesList := v.([]interface{})
		routes := make([]gona.RouterDHCPStaticRoute, len(routesList))
		for i, route := range routesList {
			routeMap := route.(map[string]interface{})
			routes[i] = gona.RouterDHCPStaticRoute{
				Network: routeMap["network"].(string),
				NextHop: routeMap["next_hop"].(string),
			}
		}
		updateRequest.StaticRoutes = routes
	}

	_, err := c.UpdateRouterVRFDHCP(routerID, vrfID, &updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))

	return resourceRouterVRFDHCPRead(ctx, d, m)
}

func resourceRouterVRFDHCPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	var routerID, vrfID int
	var err error

	if d.Id() != "" {
		routerID, vrfID, err = parseRouterVRFDHCPImportID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		routerID = d.Get("router_id").(int)
		vrfID = d.Get("vrf_id").(int)
	}

	dhcpConfig, err := c.GetRouterVRFDHCP(routerID, vrfID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("enabled", dhcpConfig.Enabled, d, &diags)
	setValue("subnet", dhcpConfig.Subnet, d, &diags)
	setValue("lease_timeout", dhcpConfig.LeaseTimeout, d, &diags)
	setValue("do_ping_check", dhcpConfig.DoPingCheck, d, &diags)
	setValue("default_router_address", dhcpConfig.DefaultRouterAddress, d, &diags)
	setValue("client_domain_name", dhcpConfig.ClientDomainName, d, &diags)
	setValue("interface_id", dhcpConfig.InterfaceID, d, &diags)

	if dhcpConfig.Range != nil {
		rangeMap := map[string]interface{}{
			"first_address": dhcpConfig.Range.FirstAddress,
			"last_address":  dhcpConfig.Range.LastAddress,
		}
		setValue("range", []interface{}{rangeMap}, d, &diags)
	}

	if len(dhcpConfig.DomainNameServers) > 0 {
		servers := make([]interface{}, len(dhcpConfig.DomainNameServers))
		for i, server := range dhcpConfig.DomainNameServers {
			servers[i] = map[string]interface{}{
				"address": server.Address,
			}
		}
		setValue("domain_name_servers", servers, d, &diags)
	}

	if len(dhcpConfig.NTPServers) > 0 {
		servers := make([]interface{}, len(dhcpConfig.NTPServers))
		for i, server := range dhcpConfig.NTPServers {
			servers[i] = map[string]interface{}{
				"address": server.Address,
			}
		}
		setValue("ntp_servers", servers, d, &diags)
	}

	if len(dhcpConfig.StaticRoutes) > 0 {
		routes := make([]interface{}, len(dhcpConfig.StaticRoutes))
		for i, route := range dhcpConfig.StaticRoutes {
			routes[i] = map[string]interface{}{
				"network":  route.Network,
				"next_hop": route.NextHop,
			}
		}
		setValue("static_routes", routes, d, &diags)
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))

	return diags
}

func resourceRouterVRFDHCPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, err := parseRouterVRFDHCPImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(routerID)
	defer unlock()

	_, err = c.GetRouter(routerID)
	if err != nil {
		d.SetId("")
		return nil
	}

	_, err = c.GetRouterVRF(routerID, vrfID)
	if err != nil {
		d.SetId("")
		return nil
	}

	dhcpConfig, err := c.GetRouterVRFDHCP(routerID, vrfID)
	if err != nil {
		d.SetId("")
		return nil
	}

	disableRequest := gona.UpdateRouterVRFDHCPRequest{
		Enabled:      false,
		Subnet:       dhcpConfig.Subnet,
		LeaseTimeout: dhcpConfig.LeaseTimeout,
		DoPingCheck:  dhcpConfig.DoPingCheck,
		DefaultRouterAddress: dhcpConfig.DefaultRouterAddress,
		ClientDomainName: dhcpConfig.ClientDomainName,
		InterfaceID: dhcpConfig.InterfaceID,
		Range: dhcpConfig.Range,
		DomainNameServers: dhcpConfig.DomainNameServers,
        NTPServers: dhcpConfig.NTPServers,
        StaticRoutes: dhcpConfig.StaticRoutes,
	}

	_, err = c.UpdateRouterVRFDHCP(routerID, vrfID, &disableRequest)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceRouterVRFDHCPImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, vrfID, err := parseRouterVRFDHCPImportID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))
	d.Set("router_id", routerID)
	d.Set("vrf_id", vrfID)

	return []*schema.ResourceData{d}, nil
}

func parseRouterVRFDHCPImportID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid DHCP ID %q, expected \"routerId/vrfId\"", id)
	}

	routerID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid router_id %q: %w", parts[0], err)
	}

	vrfID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vrf_id %q: %w", parts[1], err)
	}

	return routerID, vrfID, nil
}
