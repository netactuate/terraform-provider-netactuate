package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceServer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"plan_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"package": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"location_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"image": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"ip_v4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_v6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ipv4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ipv6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"bgp_peers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"group_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"localasn": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"peerasn": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"localpeerv4": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"localpeerv6": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     schema.TypeString,
						},
						"ipv6": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     schema.TypeString,
						},
					},
				},
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	server, err := c.GetServer(d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	ips, err := c.GetIPs(server.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	bgpSessions, err := c.GetBGPSessions(server.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("hostname", server.Name, d, &diags)
	setValue("package", server.Package, d, &diags)
	setValue("plan_id", server.PlanID, d, &diags)
	setValue("location_id", server.LocationID, d, &diags)
	setValue("image", server.OS, d, &diags)
	setValue("image_id", server.OSID, d, &diags)
	setValue("ip_v4", server.PrimaryIPv4, d, &diags)
	setValue("ip_v6", server.PrimaryIPv6, d, &diags)
	setValue("status", server.ServerStatus, d, &diags)
	setValue("state", server.PowerStatus, d, &diags)

	if len(ips.IPv4) > 0 {
		setValue("public_ipv4", ips.IPv4[0].IP, d, &diags)
	}
	if len(ips.IPv6) > 0 {
		setValue("public_ipv6", ips.IPv6[0].IP, d, &diags)
	}

	if len(bgpSessions) > 0 {
		var peerV4 []string
		var peerV6 []string

		bgpPeers := make(map[string]interface{})

		session := (bgpSessions)[0]

		bgpPeers["group_id"] = session.GroupID
		bgpPeers["localasn"] = session.CustomerAsn
		bgpPeers["peerasn"] = session.ProviderAsn

		for _, session := range bgpSessions {
			if session.IsProviderIPTypeV4() {
				bgpPeers["localpeerv4"] = session.CustomerIP
				peerV4 = append(peerV4, session.ProviderPeerIP)
			} else {
				bgpPeers["localpeerv6"] = session.CustomerIP
				peerV6 = append(peerV6, session.ProviderPeerIP)
			}
		}

		if len(peerV4) > 0 {
			bgpPeers["ipv4"] = peerV4
		}
		if len(peerV6) > 0 {
			bgpPeers["ipv6"] = peerV6
		}

		setValue("bgp_peers", []map[string]interface{}{bgpPeers}, d, &diags)
	}

	if diags == nil {
		d.SetId(strconv.Itoa(server.ID))
	}

	return diags
}
