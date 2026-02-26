package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceRouterIPSec() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterIPSecCreate,
		ReadContext:   resourceRouterIPSecRead,
		UpdateContext: resourceRouterIPSecUpdate,
		DeleteContext: resourceRouterIPSecDelete,
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"ike_do_auto_renegotiation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Auto-reconnect on peer loss.",
			},
			"ike_key_exchange_version": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          2,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{1, 2})),
				Description:      "IKE version (1 or 2).",
			},
			"ike_lifetime_seconds": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          28800,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 86400)),
				Description:      "IKE SA lifetime in seconds (0-86400).",
			},
			"ike_dh_group_number": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     14,
				Description: "Diffie-Hellman group number.",
			},
			"ike_encryption": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "aes256",
				Description: "IKE encryption algorithm.",
			},
			"ike_hash": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "sha256",
				Description: "IKE hash algorithm.",
			},
			"ike_prf": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "prfsha256",
				Description: "Pseudo-random function.",
			},
			"esp_lifetime_seconds": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          3600,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(30, 86400)),
				Description:      "ESP SA lifetime in seconds (30-86400).",
			},
			"esp_encryption": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "aes256",
				Description: "ESP encryption algorithm.",
			},
			"esp_hash": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "sha256",
				Description: "ESP hash algorithm.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func buildIPSecConfigRequest(d *schema.ResourceData) gona.UpdateRouterIPSecConfigRequest {
	return gona.UpdateRouterIPSecConfigRequest{
		IKEGroup: gona.RouterIPSecIKEGroup{
			DoAutoRenegotiation: d.Get("ike_do_auto_renegotiation").(bool),
			KeyExchangeVersion:  d.Get("ike_key_exchange_version").(int),
			LifetimeSeconds:     d.Get("ike_lifetime_seconds").(int),
			DHGroupNumber:       d.Get("ike_dh_group_number").(int),
			Encryption:          d.Get("ike_encryption").(string),
			Hash:                d.Get("ike_hash").(string),
			PRF:                 d.Get("ike_prf").(string),
		},
		ESPGroup: gona.RouterIPSecESPGroup{
			LifetimeSeconds: d.Get("esp_lifetime_seconds").(int),
			Encryption:      d.Get("esp_encryption").(string),
			Hash:            d.Get("esp_hash").(string),
		},
	}
}

func resourceRouterIPSecCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)

	req := buildIPSecConfigRequest(d)
	if err := c.UpdateRouterIPSecConfig(routerID, req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(routerID))

	return resourceRouterIPSecRead(ctx, d, m)
}

func resourceRouterIPSecRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	config, err := c.GetRouterIPSecConfig(routerID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("ike_do_auto_renegotiation", config.IKEGroup.DoAutoRenegotiation, d, &diags)
	setValue("ike_key_exchange_version", config.IKEGroup.KeyExchangeVersion, d, &diags)
	setValue("ike_lifetime_seconds", config.IKEGroup.LifetimeSeconds, d, &diags)
	setValue("ike_dh_group_number", config.IKEGroup.DHGroupNumber, d, &diags)
	setValue("ike_encryption", config.IKEGroup.Encryption, d, &diags)
	setValue("ike_hash", config.IKEGroup.Hash, d, &diags)
	setValue("ike_prf", config.IKEGroup.PRF, d, &diags)
	setValue("esp_lifetime_seconds", config.ESPGroup.LifetimeSeconds, d, &diags)
	setValue("esp_encryption", config.ESPGroup.Encryption, d, &diags)
	setValue("esp_hash", config.ESPGroup.Hash, d, &diags)

	return diags
}

func resourceRouterIPSecUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildIPSecConfigRequest(d)
	if err := c.UpdateRouterIPSecConfig(routerID, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterIPSecRead(ctx, d, m)
}

func resourceRouterIPSecDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	defaults := gona.UpdateRouterIPSecConfigRequest{
		IKEGroup: gona.RouterIPSecIKEGroup{
			DoAutoRenegotiation: true,
			KeyExchangeVersion:  2,
			LifetimeSeconds:     28800,
			DHGroupNumber:       14,
			Encryption:          "aes256",
			Hash:                "sha256",
			PRF:                 "prfsha256",
		},
		ESPGroup: gona.RouterIPSecESPGroup{
			LifetimeSeconds: 3600,
			Encryption:      "aes256",
			Hash:            "sha256",
		},
	}

	if err := c.UpdateRouterIPSecConfig(routerID, defaults); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
