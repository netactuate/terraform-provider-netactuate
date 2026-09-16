package netactuate

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceDNSRecord() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDNSRecordCreate,
		ReadContext:   resourceDNSRecordRead,
		UpdateContext: resourceDNSRecordUpdate,
		DeleteContext: resourceDNSRecordDelete,
		CustomizeDiff: dnsMxPriorityDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"zone_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Parent DNS zone ID",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DNS record name",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fully qualified name of the record, as the API reports it",
			},
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(dnsRecordTypes, false)),
				Description:      "DNS record type",
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DNS record content",
			},
			"ttl": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(1, 604800),
				Description:  "DNS record TTL",
			},
			"priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "DNS record priority",
			},
		},
	}
}

func resourceDNSRecordCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.CreateDNSRecordRequest{
		ZoneID:        d.Get("zone_id").(int),
		Name:          d.Get("name").(string),
		Type:          d.Get("type").(string),
		RecordContent: d.Get("content").(string),
	}
	if v, ok := d.GetOk("ttl"); ok {
		req.TTL = v.(int)
	}
	if v, ok := d.GetOk("priority"); ok {
		req.Priority = v.(int)
	}

	record, err := c.CreateRecord(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(record.ID))
	return resourceDNSRecordRead(ctx, d, m)
}

func resourceDNSRecordRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	record, err := c.GetRecord(id)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("zone_id", record.ZoneID, d, &diags)
	// The API returns the FULLY QUALIFIED name ("www.example.com") while a config
	// naturally writes the relative one ("www"). Storing the API's form produces a
	// permanent diff for anyone who wrote the relative name, and storing the operator's
	// form only works when there IS one, which is not the case on import.
	//
	// So always normalise to the relative name by stripping the zone suffix. That makes
	// create and import agree, which is the property this whole programme exists to
	// protect: import then plan must be empty. The qualified form stays available as
	// fqdn, matching digitalocean_record.
	//
	// Costs one extra GET per record refresh, which is the price of import round
	// tripping correctly.
	relative := record.Name
	if zone, zerr := c.GetZone(record.ZoneID); zerr == nil && zone != nil && zone.Name != "" {
		if record.Name == zone.Name {
			relative = "@"
		} else if strings.HasSuffix(record.Name, "."+zone.Name) {
			relative = strings.TrimSuffix(record.Name, "."+zone.Name)
		}
	}
	setValue("name", relative, d, &diags)
	setValue("fqdn", record.Name, d, &diags)
	setValue("type", record.Type, d, &diags)
	setValue("content", record.Content, d, &diags)
	setValue("ttl", record.TTL.Int(), d, &diags)
	setValue("priority", record.Priority, d, &diags)
	return diags
}

func resourceDNSRecordUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateDNSRecordRequest{
		ID:            id,
		ZoneID:        d.Get("zone_id").(int),
		Name:          d.Get("name").(string),
		Type:          d.Get("type").(string),
		RecordContent: d.Get("content").(string),
	}
	if v, ok := d.GetOk("ttl"); ok {
		req.TTL = v.(int)
	}
	if v, ok := d.GetOk("priority"); ok {
		req.Priority = v.(int)
	}

	if _, err := c.UpdateRecord(req); err != nil {
		return diag.FromErr(err)
	}

	return resourceDNSRecordRead(ctx, d, m)
}

func resourceDNSRecordDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteRecord(id); err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
