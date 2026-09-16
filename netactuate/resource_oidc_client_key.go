package netactuate

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceOIDCClientKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOIDCClientKeyCreate,
		ReadContext:   resourceOIDCClientKeyRead,
		UpdateContext: resourceOIDCClientKeyUpdate,
		DeleteContext: resourceOIDCClientKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "OIDC client ID",
			},
			"key_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "OIDC client key ID assigned by the API",
			},
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display label for the key",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description for the key",
			},
			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Sensitive:   true,
				Description: "Public JWK value. The parent netactuate_oidc_client must NOT set jwks_uri: the two ways of supplying keys are mutually exclusive and the API rejects the combination with HTTP 400.",
			},
			"provided_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the key was provided",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Key type",
			},
		},
	}
}

func resourceOIDCClientKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID := d.Get("oidc_client_id").(int)
	keys, err := c.CreateOIDCClientKeys(clientID, []gona.CreateOIDCClientKeyRequest{{
		Label:       d.Get("label").(string),
		Description: d.Get("description").(string),
		PublicKey:   d.Get("public_key").(string),
	}})
	if err != nil {
		return diag.FromErr(err)
	}
	if len(keys) == 0 {
		return diag.Errorf("OIDC client %d key create returned no keys", clientID)
	}

	d.SetId(fmt.Sprintf("%d/%d", clientID, keys[0].KeyID))
	return resourceOIDCClientKeyRead(ctx, d, m)
}

func resourceOIDCClientKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, keyID, err := parseTwoPartID(d.Id(), "clientId", "keyId")
	if err != nil {
		return diag.FromErr(err)
	}

	keys, err := c.GetOIDCClientKeys(clientID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] OIDC client %d not found for key %d, removing from state", clientID, keyID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	for _, key := range keys {
		if key.KeyID == keyID {
			var diags diag.Diagnostics
			setValue("oidc_client_id", clientID, d, &diags)
			for attr, value := range flattenOIDCKey(key) {
				if attr == "public_key" && value == "" {
					continue
				}
				setValue(attr, value, d, &diags)
			}
			return diags
		}
	}

	d.SetId("")
	return nil
}

func resourceOIDCClientKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, keyID, err := parseTwoPartID(d.Id(), "clientId", "keyId")
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("label") || d.HasChange("description") {
		err := c.UpdateOIDCClientKey(clientID, keyID, &gona.UpdateOIDCClientKeyRequest{
			Label:       d.Get("label").(string),
			Description: d.Get("description").(string),
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceOIDCClientKeyRead(ctx, d, m)
}

func resourceOIDCClientKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, keyID, err := parseTwoPartID(d.Id(), "clientId", "keyId")
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteOIDCClientKey(clientID, keyID); err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
