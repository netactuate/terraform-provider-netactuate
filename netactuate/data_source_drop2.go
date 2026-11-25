package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNKEClusterAPIAllowList() *schema.Resource {
	fields := map[string]*schema.Schema{}
	for _, k := range []string{"allow_list_entry_id", "network", "description", "owner", "created_on"} {
		fields[k] = drop2String(true)
	}
	fields["is_platform"] = &schema.Schema{Type: schema.TypeBool, Computed: true}
	return &schema.Resource{Schema: map[string]*schema.Schema{"cluster_id": {Type: schema.TypeInt, Required: true}, "entries": {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: fields}}}, ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		id := d.Get("cluster_id").(int)
		rows, err := m.(*ProviderClients).V3.ListNKEAPIAllowList(id)
		if err != nil {
			return drop2Error(err)
		}
		out := []map[string]interface{}{}
		for _, r := range rows {
			out = append(out, map[string]interface{}{"allow_list_entry_id": r.AllowListEntryID, "network": r.Network, "description": r.Description, "is_platform": r.IsPlatform, "owner": r.Owner, "created_on": r.CreatedOn})
		}
		var diags diag.Diagnostics
		setValue("entries", out, d, &diags)
		d.SetId(fmt.Sprintf("%d", id))
		return diags
	}}
}
func dataSourceVPCAttachments() *schema.Resource {
	fields := resourceVPCAttachment().Schema
	delete(fields, "vpc_id")
	for _, s := range fields {
		s.Required = false
		s.Optional = false
		s.Computed = true
		s.ForceNew = false
		s.ValidateDiagFunc = nil
		s.MinItems = 0
		s.MaxItems = 0
	}
	fields["bgp"] = drop3BGPSchema(true)
	fields["attachment_id"] = &schema.Schema{Type: schema.TypeInt, Computed: true}
	return &schema.Resource{Schema: map[string]*schema.Schema{"vpc_id": {Type: schema.TypeInt, Required: true}, "attachments": {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: fields}}}, ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		id := d.Get("vpc_id").(int)
		c := m.(*ProviderClients).V3
		rows, err := c.ListVPCAttachments(id)
		if err != nil {
			return drop2Error(err)
		}
		out := []map[string]interface{}{}
		for _, r := range rows {
			a, err := c.GetVPCAttachment(id, r.ID)
			if err != nil {
				return drop2Error(err)
			}
			row := flattenDrop2Attachment(a)
			row["attachment_id"] = r.ID
			out = append(out, row)
		}
		var diags diag.Diagnostics
		setValue("attachments", out, d, &diags)
		d.SetId(fmt.Sprintf("%d", id))
		return diags
	}}
}
func drop2ReservationTypeSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Optional: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"customerReserved", "lbVip"}, false))}
}
func drop2ReservationsSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: map[string]*schema.Schema{"ip_reservation_id": {Type: schema.TypeInt, Computed: true}, "address": drop2String(true), "description": drop2String(true), "reserved_on": drop2String(true), "in_use": {Type: schema.TypeBool, Computed: true}}}}
}
func dataSourceVPCIPReservationsTypedRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("vpc_id").(int)
	kind := d.Get("type").(string)
	rows, err := m.(*ProviderClients).V3.ListVPCIPReservationsByType(id, kind)
	if err != nil {
		return drop2Error(err)
	}
	out := []map[string]interface{}{}
	for _, r := range rows {
		out = append(out, map[string]interface{}{"ip_reservation_id": r.IPReservationID, "address": r.Address, "description": r.Description, "reserved_on": r.ReservedOn, "in_use": r.InUse})
	}
	var diags diag.Diagnostics
	setValue("reservations", out, d, &diags)
	d.SetId(fmt.Sprintf("vpc-%d-ip-reservations-%s", id, kind))
	return diags
}
