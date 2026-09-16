package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceBGPPrefixPurchase() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBGPPrefixPurchaseCreate,
		ReadContext:   resourceBGPPrefixPurchaseRead,
		DeleteContext: resourceBGPPrefixPurchaseDelete,
		Description:   "Purchases an anycast BGP prefix. The API has no release endpoint for purchased prefixes, so deleting this resource removes it from Terraform state only and leaves the prefix on the account.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Prefix purchase name.",
			},
			"group_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{"group_id", "asn_id"},
				Description:  "BGP group ID to attach the prefix to.",
			},
			"asn_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{"group_id", "asn_id"},
				Description:  "BGP ASN ID used when group_id is not supplied.",
			},
			"anycast_profile": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Anycast profile ID.",
			},
			"agreement_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Accepted legal agreement ID for the purchase.",
			},
			"bgp_prefix_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "API assigned BGP prefix ID.",
			},
			"prefix": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Purchased prefix.",
			},
		},
	}
}

func resourceBGPPrefixPurchaseCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.BuyBGPPrefixesRequest{
		Name:        d.Get("name").(string),
		AgreementID: d.Get("agreement_id").(int),
	}
	if v, ok := d.GetOk("group_id"); ok {
		req.GroupID = v.(int)
	}
	if v, ok := d.GetOk("asn_id"); ok {
		req.ASNID = v.(int)
	}
	if v, ok := d.GetOk("anycast_profile"); ok {
		req.AnycastProfile = v.(int)
	}

	prefix, err := c.BuyBGPPrefixes(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(prefix.ID))
	return resourceBGPPrefixPurchaseRead(ctx, d, m)
}

func resourceBGPPrefixPurchaseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	prefix, err := c.GetBGPPrefix(id)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setBGPPrefixState(prefix, d, &diags)
	return diags
}

func resourceBGPPrefixPurchaseDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "BGP prefix remains on the account",
		Detail:   "The NetActuate API has no release endpoint for purchased BGP prefixes. Terraform removed BGP prefix " + id + " from state, but the prefix remains on the account and must be released through support.",
	}}
}

func setBGPPrefixState(prefix *gona.BGPPrefix, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("bgp_prefix_id", prefix.ID, d, diags)
	setValue("name", prefix.Name, d, diags)
	setValue("prefix", prefix.Prefix, d, diags)
	if prefix.GroupID != 0 {
		setValue("group_id", prefix.GroupID, d, diags)
	}
	if prefix.ASNID != 0 {
		setValue("asn_id", prefix.ASNID, d, diags)
	}
	if prefix.AnycastProfile != 0 {
		setValue("anycast_profile", prefix.AnycastProfile, d, diags)
	}
	if prefix.AgreementID != 0 {
		setValue("agreement_id", prefix.AgreementID, d, diags)
	}
}
