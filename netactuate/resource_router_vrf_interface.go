package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFInterface() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFInterfaceCreate,
		ReadContext:   resourceRouterVRFInterfaceRead,
		UpdateContext: resourceRouterVRFInterfaceUpdate,
		DeleteContext: resourceRouterVRFInterfaceDelete,
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
			"interface_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the interface.",
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"dummy",
					"loopback",
					"wireguard",
					"ethernet",
				}, false),
				Description: "The type of interface. Valid values: dummy, loopback, wireguard, ethernet.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the interface.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the interface.",
			},
			"ipv4_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An IPv4 address to apply to the interface in CIDR notation (e.g., 178.253.0.1/16).",
			},
			"ipv6_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An IPv6 address to apply to the interface in CIDR notation.",
			},
			"ethernet_hardware_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When creating a new ethernet interface, you must provide the Hardware ID NetActuate support provided you.",
			},
			"wireguard_port": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
				Description:  "The port to use if this is a wireguard interface. Must be between 1 and 65535.",
			},
		},
	}
}

func resourceRouterVRFInterfaceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	interfaceType := d.Get("type").(string)
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	ethernetHardwareID := d.Get("ethernet_hardware_id").(string)

	createRequest := gona.CreateRouterVRFInterfaceRequest{
		Type:        interfaceType,
		Name:        name,
		Description: &description,
		EthernetHardwareID: &ethernetHardwareID,
	}

	if v, ok := d.GetOk("ipv4_cidr"); ok {
		ipv4 := v.(string)
		createRequest.IPv4CIDR = &ipv4
	}

	if v, ok := d.GetOk("ipv6_cidr"); ok {
		ipv6 := v.(string)
		createRequest.IPv6CIDR = &ipv6
	}

	if v, ok := d.GetOk("wireguard_port"); ok {
		port := v.(int)
		createRequest.WireguardPort = &port
	}

	interfaceVRF, err := c.CreateRouterVRFInterface(routerID, vrfID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(interfaceVRF.InterfaceID))

	return resourceRouterVRFInterfaceRead(ctx, d, m)
}

func resourceRouterVRFInterfaceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	interfaceID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	interfaceVRF, err := c.GetRouterVRFInterface(routerID, vrfID, interfaceID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("interface_id", interfaceVRF.InterfaceID, d, &diags)
	setValue("type", interfaceVRF.Type, d, &diags)
	setValue("name", interfaceVRF.Name, d, &diags)
	setValue("description", interfaceVRF.Description, d, &diags)
	setValue("ipv4_cidr", interfaceVRF.IPv4CIDR, d, &diags)
	setValue("ipv6_cidr", interfaceVRF.IPv6CIDR, d, &diags)
	setValue("ethernet_hardware_id", interfaceVRF.EthernetHardwareID, d, &diags)

	return diags
}

func resourceRouterVRFInterfaceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	interfaceID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	interfaceType := d.Get("type").(string)
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	ethernetHardwareID := d.Get("ethernet_hardware_id").(string)

	updateRequest := gona.UpdateRouterVRFInterfaceRequest{
		Type:        interfaceType,
		Name:        name,
		Description: &description,
        EthernetHardwareID: &ethernetHardwareID,
	}

	if v, ok := d.GetOk("ipv4_cidr"); ok {
		ipv4 := v.(string)
		updateRequest.IPv4CIDR = &ipv4
	}

	if v, ok := d.GetOk("ipv6_cidr"); ok {
		ipv6 := v.(string)
		updateRequest.IPv6CIDR = &ipv6
	}

	if v, ok := d.GetOk("wireguard_port"); ok {
		port := v.(int)
		updateRequest.WireguardPort = &port
	}

	_, err = c.UpdateRouterVRFInterface(routerID, vrfID, interfaceID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterVRFInterfaceRead(ctx, d, m)
}

func resourceRouterVRFInterfaceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	interfaceID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteRouterVRFInterface(routerID, vrfID, interfaceID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}