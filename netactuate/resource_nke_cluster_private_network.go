package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
	"strconv"
)

func resourceNKEClusterPrivateNetwork() *schema.Resource {
	return &schema.Resource{CreateContext: resourceNKEClusterPrivateNetworkCreate, ReadContext: resourceNKEClusterPrivateNetworkRead, DeleteContext: resourceNKEClusterPrivateNetworkDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("cluster_id", false, false)}, Schema: map[string]*schema.Schema{"cluster_id": drop2IDSchema(), "vpc_id": drop2IDSchema(), "state": drop2String(true), "nodes_requiring_roll": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeInt}}}}
}
func resourceNKEClusterPrivateNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("cluster_id").(int)
	c := m.(*ProviderClients).V3
	if _, err := c.AddNKEPrivateNetwork(id, &gona.AddNKEPrivateNetworkRequest{VPCID: d.Get("vpc_id").(int)}); err != nil {
		return drop2Error(err)
	}
	d.SetId(strconv.Itoa(id))
	if err := drop2Wait(ctx, d.Timeout(schema.TimeoutCreate), func() (bool, error) {
		r, err := c.GetNKEPrivateNetwork(id)
		if err != nil {
			return false, err
		}
		return r.State == "applied" || r.State == "roll_required", nil
	}); err != nil {
		return drop2Error(err)
	}
	return resourceNKEClusterPrivateNetworkRead(ctx, d, m)
}
func resourceNKEClusterPrivateNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return drop2Error(err)
	}
	r, err := m.(*ProviderClients).V3.GetNKEPrivateNetwork(id)
	if err != nil {
		return drop2ReadError(d, err)
	}
	if r.State == "none" {
		d.SetId("")
		return nil
	}
	var diags diag.Diagnostics
	for k, val := range map[string]interface{}{"cluster_id": id, "vpc_id": r.VPCID, "state": r.State, "nodes_requiring_roll": r.NodesRequiringRoll} {
		setValue(k, val, d, &diags)
	}
	if r.State == "roll_required" {
		diags = append(diags, diag.Diagnostic{Severity: diag.Warning, Summary: "Worker nodes require a roll", Detail: "Roll the nodes listed in nodes_requiring_roll to apply their private network."})
	}
	return diags
}
func resourceNKEClusterPrivateNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{{Severity: diag.Warning, Summary: "Private network removed from Terraform state", Detail: "The API has no detach operation. The cluster remains attached to its private network."}}
}
