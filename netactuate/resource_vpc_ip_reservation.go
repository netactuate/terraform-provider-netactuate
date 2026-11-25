package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceVPCIPReservation() *schema.Resource {
	return &schema.Resource{CreateContext: resourceVPCIPReservationCreate, ReadContext: resourceVPCIPReservationRead, DeleteContext: resourceVPCIPReservationDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("vpc_id", true, false)}, Schema: map[string]*schema.Schema{
		"vpc_id":      drop2IDSchema(),
		"description": {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 255))},
		"address":     {Type: schema.TypeString, Optional: true, Computed: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address)},
		"in_use":      {Type: schema.TypeBool, Computed: true}, "reserved_on": drop2String(true)}}
}
func resourceVPCIPReservationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v := d.Get("vpc_id").(int)
	id, err := m.(*ProviderClients).V3.CreateVPCIPReservation(v, &gona.CreateVPCIPReservationRequest{Description: d.Get("description").(string), Address: d.Get("address").(string)})
	if err != nil {
		return drop2Error(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", v, id))
	return resourceVPCIPReservationRead(ctx, d, m)
}
func resourceVPCIPReservationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, err := drop2Parts(d)
	if err != nil {
		return drop2Error(err)
	}
	rows, err := m.(*ProviderClients).V3.ListVPCIPReservationsByType(v, "customerReserved")
	if err != nil {
		return drop2ReadError(d, err)
	}
	for _, r := range rows {
		if strconv.Itoa(r.IPReservationID) == id {
			var diags diag.Diagnostics
			for k, val := range map[string]interface{}{"vpc_id": v, "description": r.Description, "address": r.Address, "in_use": r.InUse, "reserved_on": r.ReservedOn} {
				setValue(k, val, d, &diags)
			}
			return diags
		}
	}
	d.SetId("")
	return nil
}
func resourceVPCIPReservationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, err := drop2Parts(d)
	if err != nil {
		return drop2Error(err)
	}
	n, err := strconv.Atoi(id)
	if err != nil {
		return drop2Error(err)
	}
	err = m.(*ProviderClients).V3.DeleteVPCIPReservation(v, n)
	if err != nil && !gona.IsV3NotFound(err) {
		return drop2Error(err)
	}
	d.SetId("")
	return nil
}
