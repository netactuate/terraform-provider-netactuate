package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceDNSZone() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDNSZoneCreate,
		ReadContext:   resourceDNSZoneRead,
		DeleteContext: resourceDNSZoneDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "DNS zone name",
			},
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"NATIVE", "SLAVE"}, false)),
				Description:      "DNS zone type",
			},
			"ip": {
				Type:        schema.TypeString,
				Optional:    true,
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

func resourceDNSZoneCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.CreateDNSZoneRequest{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
	}
	if v, ok := d.GetOk("ip"); ok {
		req.IP = v.(string)
	}

	zone, err := c.CreateZone(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(zone.ID))
	return resourceDNSZoneRead(ctx, d, m)
}

func resourceDNSZoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	zone, err := c.GetZone(id)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("name", zone.Name, d, &diags)
	setValue("type", zone.Type, d, &diags)
	// ns is a list of NS records and soa is an object; flatten both to what the schema
	// declares rather than pretending the API returns strings.
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

	ip := ""
	zones, err := c.ListZones(zone.Type)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, listed := range zones {
		if listed.ID == zone.ID {
			ip = listed.Master
			break
		}
	}
	setValue("ip", ip, d, &diags)

	return diags
}

func resourceDNSZoneDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	records, err := c.ListRecords(id)
	if err != nil && !gona.IsNotFound(err) {
		return diag.FromErr(err)
	}
	for _, record := range records {
		if err := c.DeleteRecord(record.ID); err != nil && !gona.IsNotFound(err) {
			return diag.FromErr(err)
		}
	}

	if err := c.DeleteZone(id); err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
