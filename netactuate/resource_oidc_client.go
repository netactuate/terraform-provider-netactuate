package netactuate

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceOIDCClient() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOIDCClientCreate,
		ReadContext:   resourceOIDCClientRead,
		UpdateContext: resourceOIDCClientUpdate,
		DeleteContext: resourceOIDCClientDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: oidcClientSchema(),
	}
}

func resourceOIDCClientCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	req := &gona.CreateOIDCClientRequest{
		Label:            d.Get("label").(string),
		Description:      d.Get("description").(string),
		AccountDefault:   d.Get("account_default").(bool),
		EnforceAllowList: d.Get("enforce_allow_list").(bool),
		TTL:              d.Get("ttl").(int),
		DefaultAudience:  d.Get("default_audience").(string),
	}
	if v, ok := d.GetOk("jwks_uri"); ok {
		jwksURI := v.(string)
		req.JWKSURI = &jwksURI
	}

	clientID, err := c.CreateOIDCClient(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(clientID))
	return resourceOIDCClientRead(ctx, d, m)
}

func resourceOIDCClientRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	client, err := c.GetOIDCClient(clientID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] OIDC client %d not found, removing from state", clientID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for key, value := range flattenOIDCClient(*client) {
		setValue(key, value, d, &diags)
	}
	return diags
}

func resourceOIDCClientUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateOIDCClientRequest{}
	changed := false

	if d.HasChange("label") {
		req.Label = d.Get("label").(string)
		changed = true
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
		changed = true
	}
	if d.HasChange("jwks_uri") {
		jwksURI := d.Get("jwks_uri").(string)
		req.JWKSURI = &jwksURI
		changed = true
	}
	if d.HasChange("account_default") {
		v := d.Get("account_default").(bool)
		req.AccountDefault = &v
		changed = true
	}
	if d.HasChange("enforce_allow_list") {
		v := d.Get("enforce_allow_list").(bool)
		req.EnforceAllowList = &v
		changed = true
	}
	if d.HasChange("ttl") {
		req.TTL = d.Get("ttl").(int)
		changed = true
	}
	if d.HasChange("default_audience") {
		req.DefaultAudience = d.Get("default_audience").(string)
		changed = true
	}

	if changed {
		if err := c.UpdateOIDCClient(clientID, req); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceOIDCClientRead(ctx, d, m)
}

func resourceOIDCClientDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteOIDCClient(clientID); err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
