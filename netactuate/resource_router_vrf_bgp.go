package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFBGP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFBGPCreate,
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
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Your local ASN for BGP. Must be between 1 and 4294967294.",
			},
			"networks": {
				Type:        schema.TypeList,
				Required:     true,
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
			StateContext: resourceRouterVRFBGPImport,
		},
	}
}

func resourceRouterVRFBGPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))

	return resourceRouterVRFBGPUpdate(ctx, d, m)
}

func resourceRouterVRFBGPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, err := parseRouterVRFBGPImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	bgpConfig, err := c.GetRouterVRFBGP(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))

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

	routerID, vrfID, err := parseRouterVRFBGPImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(routerID)
	defer unlock()

	updateRequest := gona.UpdateRouterVRFBGPRequest{}

	if v, ok := d.GetOk("local_asn"); ok {
		localAsn := v.(string)
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

	_, err = c.UpdateRouterVRFBGP(routerID, vrfID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))

	return resourceRouterVRFBGPRead(ctx, d, m)
}

func resourceRouterVRFBGPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceRouterVRFBGPImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, vrfID, err := parseRouterVRFBGPImportID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))
	d.Set("router_id", routerID)
	d.Set("vrf_id", vrfID)

	return []*schema.ResourceData{d}, nil
}

func parseRouterVRFBGPImportID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid BGP ID %q, expected \"routerId/vrfId\"", id)
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
