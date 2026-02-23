package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFBGP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFBGPUpdate,
		ReadContext:   resourceRouterVRFBGPRead,
		UpdateContext: resourceRouterVRFBGPUpdate,
		DeleteContext: resourceRouterVRFBGPDelete,
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
			"local_asn": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 4294967294),
				Description:  "Your local ASN for BGP. Must be between 1 and 4294967294.",
			},
			"networks": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of networks to announce over your BGP sessions.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The network to announce via BGP.",
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

func resourceRouterVRFBGPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	bgpConfig, err := c.GetRouterVRFBGP(routerID, vrfID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)

	if bgpConfig.LocalAsn != nil {
		setValue("local_asn", *bgpConfig.LocalAsn, d, &diags)
	}

	if len(bgpConfig.Networks) > 0 {
		networks := make([]map[string]interface{}, len(bgpConfig.Networks))
		for i, network := range bgpConfig.Networks {
			networks[i] = map[string]interface{}{
				"subnet": network.Subnet,
			}
		}
		setValue("networks", networks, d, &diags)
	}

	return diags
}

func resourceRouterVRFBGPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	updateRequest := gona.UpdateRouterVRFBGPRequest{}

	if v, ok := d.GetOk("local_asn"); ok {
		localAsn := v.(int)
		updateRequest.ASN = &gona.RouterVRFBGPASN{
			Local: &localAsn,
		}
	}

	if v, ok := d.GetOk("networks"); ok {
		networksList := v.([]interface{})
		networks := make([]gona.RouterVRFBGPNetwork, len(networksList))
		for i, n := range networksList {
			networkMap := n.(map[string]interface{})
			networks[i] = gona.RouterVRFBGPNetwork{
				Subnet: networkMap["subnet"].(string),
			}
		}
		updateRequest.Networks = networks
	}

	_, err := c.UpdateRouterVRFBGP(routerID, vrfID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(vrfID))

	return resourceRouterVRFBGPRead(ctx, d, m)
}

func resourceRouterVRFBGPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	updateRequest := gona.UpdateRouterVRFBGPRequest{
		Networks: []gona.RouterVRFBGPNetwork{},
	}

	_, err := c.UpdateRouterVRFBGP(routerID, vrfID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}