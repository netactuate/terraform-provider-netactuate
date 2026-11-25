package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func drop3BGPSchema(computed bool) *schema.Schema {
	fields := map[string]*schema.Schema{
		"peer_asn":     {Type: schema.TypeInt, Required: true, ValidateFunc: validation.IntBetween(1, 4294967295)},
		"max_prefix":   {Type: schema.TypeInt, Optional: true, Computed: true, ValidateFunc: validation.IntBetween(1, 1000)},
		"password":     {Type: schema.TypeString, Optional: true, Sensitive: true, ValidateFunc: validation.StringLenBetween(8, 80)},
		"has_password": drop3Computed(schema.TypeBool), "gateway_asn": drop3Computed(schema.TypeInt),
	}
	for _, k := range []string{"gateway_address", "peer_address", "state", "state_message", "applied_on"} {
		fields[k] = drop3Computed(schema.TypeString)
	}
	if computed {
		delete(fields, "password")
		for _, s := range fields {
			s.Required = false
			s.Optional = false
			s.Computed = true
			s.ValidateFunc = nil
		}
	}
	limit := 1
	if computed {
		limit = 0
	}
	return &schema.Schema{Type: schema.TypeList, Optional: !computed, Computed: computed, MaxItems: limit, Elem: &schema.Resource{Schema: fields}}
}
func drop3ExpandBGP(d *schema.ResourceData) *gona.VPCAttachmentBGPRequest {
	rows := d.Get("bgp").([]interface{})
	if len(rows) == 0 || rows[0] == nil {
		return nil
	}
	r := rows[0].(map[string]interface{})
	return &gona.VPCAttachmentBGPRequest{PeerASN: r["peer_asn"].(int), MaxPrefix: r["max_prefix"].(int), Password: r["password"].(string)}
}
func drop3FlattenBGP(r *gona.VPCAttachmentBGP) []interface{} {
	if r == nil {
		return []interface{}{}
	}
	return []interface{}{map[string]interface{}{"peer_asn": r.PeerASN, "max_prefix": r.MaxPrefix, "has_password": r.HasPassword, "gateway_asn": r.GatewayASN, "gateway_address": r.GatewayAddress, "peer_address": r.PeerAddress, "state": r.State, "state_message": r.StateMessage, "applied_on": r.AppliedOn}}
}
func drop3ReadBGP(d *schema.ResourceData, r *gona.VPCAttachmentBGP) diag.Diagnostics {
	rows := drop3FlattenBGP(r)
	if len(rows) > 0 {
		old := d.Get("bgp").([]interface{})
		if len(old) > 0 && old[0] != nil {
			rows[0].(map[string]interface{})["password"] = old[0].(map[string]interface{})["password"]
		}
	}
	return drop3Set(d, map[string]interface{}{"bgp": rows})
}
func drop3UpdateBGP(ctx context.Context, d *schema.ResourceData, c *gona.V3Client, v, a int) error {
	if !d.HasChange("bgp") {
		return nil
	}
	req := drop3ExpandBGP(d)
	if req == nil {
		return c.DeleteVPCAttachmentBGP(v, a)
	}
	return c.PutVPCAttachmentBGP(v, a, req)
}
