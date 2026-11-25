package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
	"strconv"
)

func resourceNKEClusterAPIAccess() *schema.Resource {
	return &schema.Resource{CreateContext: resourceNKEClusterAPIAccessCreate, ReadContext: resourceNKEClusterAPIAccessRead, UpdateContext: resourceNKEClusterAPIAccessUpdate, DeleteContext: resourceNKEClusterAPIAccessDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("cluster_id", false, false)}, Schema: map[string]*schema.Schema{
		"cluster_id": drop2IDSchema(), "public_enabled": {Type: schema.TypeBool, Optional: true, Default: true}, "private_endpoint_enabled": {Type: schema.TypeBool, Optional: true, Default: false}, "private_endpoint_address": drop2String(true), "private_dns_name": drop2String(true), "private_endpoint_state": drop2String(true), "state": drop2String(true)}}
}
func drop2WaitAPIAccess(ctx context.Context, d *schema.ResourceData, c *gona.V3Client, id int, timeout string) error {
	return drop2Wait(ctx, d.Timeout(timeout), func() (bool, error) {
		r, err := c.GetNKEAPIAccess(id)
		if err != nil {
			return false, err
		}
		if r.State == "failed" || r.PrivateEndpointState == "failed" {
			return false, fmt.Errorf("API access failed: %s", r.FailureReason)
		}
		return r.State != "applying" && r.PrivateEndpointState != "provisioning", nil
	})
}
func drop2PutAPIAccess(ctx context.Context, d *schema.ResourceData, m interface{}, timeout string) diag.Diagnostics {
	id := d.Get("cluster_id").(int)
	pub, priv := d.Get("public_enabled").(bool), d.Get("private_endpoint_enabled").(bool)
	c := m.(*ProviderClients).V3
	if _, err := c.UpdateNKEAPIAccess(id, &gona.UpdateNKEAPIAccessRequest{PublicEnabled: &pub, PrivateEndpointEnabled: &priv}); err != nil {
		return drop2Error(err)
	}
	d.SetId(strconv.Itoa(id))
	if err := drop2WaitAPIAccess(ctx, d, c, id, timeout); err != nil {
		return drop2Error(err)
	}
	return resourceNKEClusterAPIAccessRead(ctx, d, m)
}
func resourceNKEClusterAPIAccessCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return drop2PutAPIAccess(ctx, d, m, schema.TimeoutCreate)
}
func resourceNKEClusterAPIAccessUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return drop2PutAPIAccess(ctx, d, m, schema.TimeoutUpdate)
}
func resourceNKEClusterAPIAccessRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return drop2Error(err)
	}
	r, err := m.(*ProviderClients).V3.GetNKEAPIAccess(id)
	if err != nil {
		return drop2ReadError(d, err)
	}
	var diags diag.Diagnostics
	for k, v := range map[string]interface{}{"cluster_id": id, "public_enabled": r.PublicEnabled, "private_endpoint_enabled": r.PrivateEndpointEnabled, "private_endpoint_address": r.PrivateEndpointAddress, "private_dns_name": r.PrivateDNSName, "private_endpoint_state": r.PrivateEndpointState, "state": r.State} {
		setValue(k, v, d, &diags)
	}
	return diags
}
func resourceNKEClusterAPIAccessDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return drop2Error(err)
	}
	pub, priv := true, false
	c := m.(*ProviderClients).V3
	_, err = c.UpdateNKEAPIAccess(id, &gona.UpdateNKEAPIAccessRequest{PublicEnabled: &pub, PrivateEndpointEnabled: &priv})
	if gona.IsV3NotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return drop2Error(err)
	}
	if err = drop2WaitAPIAccess(ctx, d, c, id, schema.TimeoutDelete); err != nil && !gona.IsV3NotFound(err) {
		return drop2Error(err)
	}
	d.SetId("")
	return nil
}
