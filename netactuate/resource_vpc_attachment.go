package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
	"strconv"
)

func resourceVPCAttachment() *schema.Resource {
	return &schema.Resource{CreateContext: resourceVPCAttachmentCreate, ReadContext: resourceVPCAttachmentRead, UpdateContext: resourceVPCAttachmentUpdate, DeleteContext: resourceVPCAttachmentDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("vpc_id", true, false)}, Schema: map[string]*schema.Schema{
		"vpc_id": drop2IDSchema(), "type": {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"customer-router", "cloud-router"}, false))},
		"address": {Type: schema.TypeString, Optional: true, Computed: true, ForceNew: true}, "cloud_router_id": {Type: schema.TypeInt, Optional: true, ForceNew: true}, "vrf_id": {Type: schema.TypeInt, Optional: true, ForceNew: true}, "description": {Type: schema.TypeString, Optional: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 255))},
		"accept":    {Type: schema.TypeSet, Required: true, MinItems: 1, MaxItems: 32, Elem: &schema.Schema{Type: schema.TypeString, ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR)}},
		"advertise": {Type: schema.TypeSet, Optional: true, MaxItems: 32, Elem: &schema.Schema{Type: schema.TypeString, ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR)}}, "state": drop2String(true), "state_message": drop2String(true)}}
}
func drop2AttachmentIDs(d *schema.ResourceData) (int, int, error) {
	v, id, err := drop2Parts(d)
	if err != nil {
		return 0, 0, err
	}
	a, err := strconv.Atoi(id)
	return v, a, err
}
func drop2WaitAttachment(ctx context.Context, d *schema.ResourceData, c *gona.V3Client, v, a int, timeout string, deleted bool) error {
	return drop2Wait(ctx, d.Timeout(timeout), func() (bool, error) {
		r, err := c.GetVPCAttachment(v, a)
		if deleted && gona.IsV3NotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if deleted {
			return false, nil
		}
		if r.State == "failed" {
			return false, fmt.Errorf("attachment failed: %s", r.StateMessage)
		}
		return r.State == "ready", nil
	})
}
func resourceVPCAttachmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v := d.Get("vpc_id").(int)
	c := m.(*ProviderClients).V3
	id, err := c.CreateVPCAttachment(v, &gona.CreateVPCAttachmentRequest{Type: d.Get("type").(string), Address: d.Get("address").(string), RouterID: d.Get("cloud_router_id").(int), VRFID: d.Get("vrf_id").(int), Description: d.Get("description").(string), Accept: drop2Strings(d, "accept"), Advertise: drop2Strings(d, "advertise")})
	if err != nil {
		return drop2Error(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", v, id))
	if err = drop2WaitAttachment(ctx, d, c, v, id, schema.TimeoutCreate, false); err != nil {
		return drop2Error(err)
	}
	return resourceVPCAttachmentRead(ctx, d, m)
}
func flattenDrop2Attachment(r *gona.VPCAttachment) map[string]interface{} {
	return map[string]interface{}{"type": r.Type, "address": r.Address, "cloud_router_id": r.RouterID, "vrf_id": r.VRFID, "description": r.Description, "accept": r.Accept, "advertise": r.Advertise, "state": r.State, "state_message": r.StateMessage}
}
func resourceVPCAttachmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, a, err := drop2AttachmentIDs(d)
	if err != nil {
		return drop2Error(err)
	}
	r, err := m.(*ProviderClients).V3.GetVPCAttachment(v, a)
	if err != nil {
		return drop2ReadError(d, err)
	}
	var diags diag.Diagnostics
	setValue("vpc_id", v, d, &diags)
	for k, val := range flattenDrop2Attachment(r) {
		setValue(k, val, d, &diags)
	}
	return diags
}

// Add before removing to preserve the last accept prefix. At the API's 32-prefix
// limit, remove an obsolete prefix first to make room for its replacement.
func drop2SyncPrefixes(c *gona.V3Client, v, a int, wanted map[string][]string) error {
	rows, err := c.ListVPCAttachmentPrefixes(v, a)
	if err != nil {
		return err
	}
	for _, direction := range []string{"accept", "advertise"} {
		desired := map[string]bool{}
		for _, network := range wanted[direction] {
			desired[network] = true
		}
		present := map[string]bool{}
		obsolete := []gona.VPCAttachmentPrefix{}
		for _, r := range rows {
			if r.Direction != direction || r.DeleteRequestedOn != "" {
				continue
			}
			present[r.Network] = true
			if !desired[r.Network] {
				obsolete = append(obsolete, r)
			}
		}
		count := len(present)
		remove := func() error {
			if len(obsolete) == 0 {
				return fmt.Errorf("no obsolete %s prefix available to replace", direction)
			}
			r := obsolete[0]
			if err := c.DeleteVPCAttachmentPrefix(v, a, r.ID); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
			obsolete = obsolete[1:]
			count--
			return nil
		}
		for _, network := range wanted[direction] {
			if present[network] {
				continue
			}
			if count >= 32 {
				if err := remove(); err != nil {
					return err
				}
			}
			if _, err := c.CreateVPCAttachmentPrefix(v, a, &gona.CreateVPCAttachmentPrefixRequest{Network: network, Direction: direction}); err != nil {
				return err
			}
			count++
		}
		for len(obsolete) > 0 {
			if err := remove(); err != nil {
				return err
			}
		}
	}
	return nil
}
func resourceVPCAttachmentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, a, err := drop2AttachmentIDs(d)
	if err != nil {
		return drop2Error(err)
	}
	c := m.(*ProviderClients).V3
	if err = drop2SyncPrefixes(c, v, a, map[string][]string{"accept": drop2Strings(d, "accept"), "advertise": drop2Strings(d, "advertise")}); err != nil {
		return drop2Error(err)
	}
	if err = drop2WaitAttachment(ctx, d, c, v, a, schema.TimeoutUpdate, false); err != nil {
		return drop2Error(err)
	}
	return resourceVPCAttachmentRead(ctx, d, m)
}
func resourceVPCAttachmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, a, err := drop2AttachmentIDs(d)
	if err != nil {
		return drop2Error(err)
	}
	c := m.(*ProviderClients).V3
	err = c.DeleteVPCAttachment(v, a)
	if gona.IsV3NotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return drop2Error(err)
	}
	if err = drop2WaitAttachment(ctx, d, c, v, a, schema.TimeoutDelete, true); err != nil {
		return drop2Error(err)
	}
	d.SetId("")
	return nil
}
