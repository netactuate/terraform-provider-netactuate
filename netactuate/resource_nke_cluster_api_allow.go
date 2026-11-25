package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
	"net"
	"strings"
)

func validateDrop2IPv4(v interface{}, key string) ([]string, []error) {
	s := v.(string)
	ip := net.ParseIP(s)
	if strings.Contains(s, "/") {
		var err error
		ip, _, err = net.ParseCIDR(s)
		if err != nil {
			return nil, []error{fmt.Errorf("%s must be an IPv4 address or CIDR", key)}
		}
	}
	if ip == nil || ip.To4() == nil {
		return nil, []error{fmt.Errorf("%s must be an IPv4 address or CIDR", key)}
	}
	return nil, nil
}
func resourceNKEClusterAPIAllow() *schema.Resource {
	return &schema.Resource{CreateContext: resourceNKEClusterAPIAllowCreate, ReadContext: resourceNKEClusterAPIAllowRead, DeleteContext: resourceNKEClusterAPIAllowDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("cluster_id", true, true)}, Schema: map[string]*schema.Schema{
		"cluster_id": drop2IDSchema(), "cidr": {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validateDrop2IPv4), DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return old == new+"/32" }}, "label": {Type: schema.TypeString, Optional: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 255))}}}
}
func resourceNKEClusterAPIAllowCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := d.Get("cluster_id").(int)
	r, err := m.(*ProviderClients).V3.CreateNKEAPIAllowEntry(c, &gona.CreateNKEAPIAllowEntryRequest{Network: d.Get("cidr").(string), Description: d.Get("label").(string)})
	if err != nil {
		return drop2Error(err)
	}
	d.SetId(fmt.Sprintf("%d/%s", c, r.AllowListEntryID))
	return resourceNKEClusterAPIAllowRead(ctx, d, m)
}
func resourceNKEClusterAPIAllowRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, id, err := drop2Parts(d)
	if err != nil {
		return drop2Error(err)
	}
	rows, err := m.(*ProviderClients).V3.ListNKEAPIAllowList(c)
	if err != nil {
		return drop2ReadError(d, err)
	}
	for _, r := range rows {
		if r.AllowListEntryID == id {
			var diags diag.Diagnostics
			setValue("cluster_id", c, d, &diags)
			setValue("cidr", r.Network, d, &diags)
			setValue("label", r.Description, d, &diags)
			return diags
		}
	}
	d.SetId("")
	return nil
}
func resourceNKEClusterAPIAllowDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, id, err := drop2Parts(d)
	if err != nil {
		return drop2Error(err)
	}
	err = m.(*ProviderClients).V3.DeleteNKEAPIAllowEntry(c, id)
	if err != nil && !gona.IsV3NotFound(err) {
		return drop2Error(err)
	}
	d.SetId("")
	return nil
}
