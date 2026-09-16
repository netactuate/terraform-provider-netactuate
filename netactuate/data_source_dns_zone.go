package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceDNSZone() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDNSZoneRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DNS zone name",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "DNS zone type",
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Master IP address for SLAVE zones",
			},
			"nameservers": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The nameservers authoritative for this zone",
			},
			"soa_primary": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The primary nameserver from the zone's SOA record",
			},
			"soa_hostmaster": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The hostmaster address from the zone's SOA record",
			},
			"soa_serial": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The serial from the zone's SOA record",
			},
			"ttl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The zone's default TTL, returned by the API as a string",
			},
		},
	}
}

func dataSourceDNSZoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	name := d.Get("name").(string)

	for _, zoneType := range []string{"NATIVE", "SLAVE"} {
		zones, err := c.ListZones(zoneType)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, zone := range zones {
			if zone.Name != name {
				continue
			}
			d.SetId(strconv.Itoa(zone.ID))
			return readDNSZoneDataSourceDetail(c, d, zone)
		}
	}

	return diag.Errorf("DNS zone %q not found", name)
}

func readDNSZoneDataSourceDetail(c *gona.Client, d *schema.ResourceData, listed gona.DNSZone) diag.Diagnostics {
	zone, err := c.GetZone(listed.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("name", zone.Name, d, &diags)
	setValue("type", zone.Type, d, &diags)
	setValue("ip", listed.Master, d, &diags)
	nameservers := make([]string, 0, len(zone.NS))
	for _, ns := range zone.NS {
		nameservers = append(nameservers, ns.Content)
	}
	setValue("nameservers", nameservers, d, &diags)
	if zone.SOA != nil {
		setValue("soa_primary", zone.SOA.Primary, d, &diags)
		setValue("soa_hostmaster", zone.SOA.Hostmaster, d, &diags)
		setValue("soa_serial", zone.SOA.Serial, d, &diags)
	}
	setValue("ttl", string(zone.TTL), d, &diags)
	return diags
}
