package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceBGPSessions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBGPSessionsRead,
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"sessions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"mb_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"routes_received": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"config_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"last_update": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"locked": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"group_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"group_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"location_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"customer_peer_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provider_peer_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provider_ip_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provider_asn": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"customer_asn": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceBGPSessionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	MbPkgID := d.Get("mbpkgid").(int)

	sessions, err := c.GetBGPSessions(MbPkgID)
	if err != nil {
		return diag.FromErr(err)
	}

	result := make([]map[string]interface{}, len(sessions))

	for i, session := range sessions {
		s := make(map[string]interface{})

		s["id"] = session.ID
		// s["mb_id"] = session.MbID
		s["description"] = session.Description
		s["routes_received"] = session.RoutesReceived
		s["config_status"] = fmt.Sprint(session.ConfigStatus)
		s["last_update"] = session.LastUpdate
		s["locked"] = session.IsLocked()
		s["group_id"] = session.GroupID
		s["group_name"] = session.GroupName
		s["location_name"] = session.Location
		s["customer_peer_ip"] = session.CustomerIP
		s["provider_peer_ip"] = session.ProviderPeerIP
		s["provider_ip_type"] = session.ProviderIPType
		s["customer_asn"] = session.CustomerAsn
		s["provider_asn"] = session.ProviderAsn
		s["state"] = session.State

		result[i] = s
	}

	err = d.Set("sessions", result)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(MbPkgID))

	return nil
}
