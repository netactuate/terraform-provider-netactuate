package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceRouterNTP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterNTPCreate,
		ReadContext:   resourceRouterNTPRead,
		UpdateContext: resourceRouterNTPUpdate,
		DeleteContext: resourceRouterNTPDelete,
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether or not NTP is enabled.",
			},
			"interface_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "You may restrict the NTP service to a specific interface.",
			},
			"upstreams": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The upstream NTP servers to use by the cloud router.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"domain": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The hostname of the NTP server, like time.cloudflare.com.",
						},
					},
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceRouterNTPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	routerID := d.Get("router_id").(int)

	enabled := d.Get("enabled").(bool)

	updateRequest := gona.UpdateRouterNTPConfigRequest{
		Enabled: &enabled,
	}

	if v, ok := d.GetOk("interface_id"); ok {
		interfaceID := v.(int)
		updateRequest.InterfaceID = &interfaceID
	}

	if v, ok := d.GetOk("upstreams"); ok {
		upstreamsList := v.([]interface{})
		upstreams := make([]gona.RouterNTPUpstream, len(upstreamsList))
		for i, upstream := range upstreamsList {
			upstreamMap := upstream.(map[string]interface{})
			upstreams[i] = gona.RouterNTPUpstream{
				Domain: upstreamMap["domain"].(string),
			}
		}
		updateRequest.Upstreams = upstreams
	}

	_, err := c.UpdateRouterNTPConfig(routerID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(routerID))

	return resourceRouterNTPRead(ctx, d, m)
}

func resourceRouterNTPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ntp, err := c.GetRouterNTPConfig(routerID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("enabled", ntp.Enabled, d, &diags)

	if ntp.InterfaceID != nil {
		setValue("interface_id", *ntp.InterfaceID, d, &diags)
	}

	if ntp.Upstreams != nil && len(ntp.Upstreams) > 0 {
		upstreams := make([]map[string]interface{}, len(ntp.Upstreams))
		for i, upstream := range ntp.Upstreams {
			upstreams[i] = map[string]interface{}{
				"domain": upstream.Domain,
			}
		}
		setValue("upstreams", upstreams, d, &diags)
	}

	return diags
}

func resourceRouterNTPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("enabled", "interface_id", "upstreams") {
		enabled := d.Get("enabled").(bool)

		updateRequest := gona.UpdateRouterNTPConfigRequest{
			Enabled: &enabled,
		}

		if v, ok := d.GetOk("interface_id"); ok {
			interfaceID := v.(int)
			updateRequest.InterfaceID = &interfaceID
		}

		if v, ok := d.GetOk("upstreams"); ok {
			upstreamsList := v.([]interface{})
			upstreams := make([]gona.RouterNTPUpstream, len(upstreamsList))
			for i, upstream := range upstreamsList {
				upstreamMap := upstream.(map[string]interface{})
				upstreams[i] = gona.RouterNTPUpstream{
					Domain: upstreamMap["domain"].(string),
				}
			}
			updateRequest.Upstreams = upstreams
		}

		_, err := c.UpdateRouterNTPConfig(routerID, updateRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRouterNTPRead(ctx, d, m)
}

func resourceRouterNTPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.GetRouter(routerID)
	if err != nil {
		d.SetId("")
		return nil
	}

	ntpConfig, err := c.GetRouterNTPConfig(routerID)
	if err != nil {
		d.SetId("")
		return nil
	}

	enabled := false
	updateRequest := gona.UpdateRouterNTPConfigRequest{
		Enabled: &enabled,
		Upstreams: ntpConfig.Upstreams,
	}

	_, err = c.UpdateRouterNTPConfig(routerID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}