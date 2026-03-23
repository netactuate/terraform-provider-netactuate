package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceRouterVRFIPSecPeer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFIPSecPeerCreate,
		ReadContext:   resourceRouterVRFIPSecPeerRead,
		UpdateContext: resourceRouterVRFIPSecPeerUpdate,
		DeleteContext: resourceRouterVRFIPSecPeerDelete,
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			doInitiate := diff.Get("do_initiate_connection").(bool)
			_, hasPeerAddr := diff.GetOk("peer_address")

			if !doInitiate && hasPeerAddr {
				return fmt.Errorf("peer_address cannot be set when do_initiate_connection is false (passive mode accepts connections from any address)")
			}
			if doInitiate && !hasPeerAddr {
				return fmt.Errorf("peer_address is required when do_initiate_connection is true (initiating mode)")
			}
			return nil
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
			"ipsec_peer_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The IPSec peer ID.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IPSec peer.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the IPSec peer.",
			},
			"remote_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Remote peer identifier (1-64 chars).",
			},
			"psk_secret": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Pre-shared key (1-256 chars).",
			},
			"do_initiate_connection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to initiate or respond to connection.",
			},
			"peer_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Remote router IPv4 address. Required when do_initiate_connection is true. Cannot be set in passive mode (do_initiate_connection = false).",
			},
			"overlay_ipv4": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Overlay tunnel IPv4 CIDR.",
			},
			"overlay_ipv6": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Overlay tunnel IPv6 CIDR.",
			},
			"local_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Local router ID.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func buildIPSecPeerRequest(d *schema.ResourceData) gona.CreateRouterVRFIPSecPeerRequest {
	req := gona.CreateRouterVRFIPSecPeerRequest{
		Name:                 d.Get("name").(string),
		RemoteID:             d.Get("remote_id").(string),
		PSKSecret:            d.Get("psk_secret").(string),
		DoInitiateConnection: d.Get("do_initiate_connection").(bool),
	}

	if v, ok := d.GetOk("peer_address"); ok {
		req.PeerAddress = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		desc := v.(string)
		req.Description = &desc
	}

	if v, ok := d.GetOk("overlay_ipv4"); ok {
		ipv4 := v.(string)
		req.OverlayNetwork.IPv4 = &ipv4
	}

	if v, ok := d.GetOk("overlay_ipv6"); ok {
		ipv6 := v.(string)
		req.OverlayNetwork.IPv6 = &ipv6
	}

	return req
}

func resourceRouterVRFIPSecPeerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	req := buildIPSecPeerRequest(d)

	peer, err := c.CreateRouterVRFIPSecPeer(routerID, vrfID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(peer.IPSecPeerID))

	return resourceRouterVRFIPSecPeerRead(ctx, d, m)
}

func resourceRouterVRFIPSecPeerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)
	peerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	peer, err := c.GetRouterVRFIPSecPeer(routerID, vrfID, peerID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrfID, d, &diags)
	setValue("ipsec_peer_id", peer.IPSecPeerID, d, &diags)
	setValue("name", peer.Name, d, &diags)
	setValue("description", peer.Description, d, &diags)
	setValue("remote_id", peer.RemoteID, d, &diags)
	setValue("psk_secret", peer.PSKSecret, d, &diags)
	setValue("do_initiate_connection", peer.DoInitiateConnection, d, &diags)
	setValue("peer_address", peer.PeerAddress, d, &diags)
	setValue("local_id", peer.LocalID, d, &diags)
	setValue("overlay_ipv4", peer.OverlayNetwork.IPv4, d, &diags)
	setValue("overlay_ipv6", peer.OverlayNetwork.IPv6, d, &diags)

	return diags
}

func resourceRouterVRFIPSecPeerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	peerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildIPSecPeerRequest(d)

	_, err = c.UpdateRouterVRFIPSecPeer(routerID, vrfID, peerID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterVRFIPSecPeerRead(ctx, d, m)
}

func resourceRouterVRFIPSecPeerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID := d.Get("vrf_id").(int)

	unlock := lockRouter(routerID)
	defer unlock()

	peerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteRouterVRFIPSecPeer(routerID, vrfID, peerID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
