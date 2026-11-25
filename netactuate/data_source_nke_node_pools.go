package netactuate

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceNKENodePools() *schema.Resource {
	fields := resourceNKENodePool().Schema
	delete(fields, "cluster_id")
	var computed func(map[string]*schema.Schema)
	computed = func(fields map[string]*schema.Schema) {
		for _, s := range fields {
			s.Required = false
			s.Optional = false
			s.Computed = true
			s.ForceNew = false
			s.ValidateDiagFunc = nil
			s.MaxItems = 0
			if child, ok := s.Elem.(*schema.Resource); ok {
				computed(child.Schema)
			}
		}
	}
	computed(fields)
	fields["node_pool_id"] = &schema.Schema{Type: schema.TypeInt, Computed: true, Description: "Pool ID."}
	return &schema.Resource{ReadContext: dataSourceNKENodePoolsRead, Schema: map[string]*schema.Schema{
		"cluster_id": {Type: schema.TypeInt, Required: true, Description: "Cluster ID to list pools for."},
		"node_pools": {Type: schema.TypeList, Computed: true, Description: "Node pools in the cluster.", Elem: &schema.Resource{Schema: fields}},
	}}
}
func dataSourceNKENodePoolsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get("cluster_id").(int)
	pools, err := m.(*ProviderClients).V3.ListNKENodePools(id)
	if err != nil {
		return diag.FromErr(err)
	}
	rows := make([]interface{}, 0, len(pools))
	for _, p := range pools {
		v := flattenNKENodePool(&p)
		v["node_pool_id"] = p.NodePoolID
		rows = append(rows, v)
	}
	if err = d.Set("node_pools", rows); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(id))
	return nil
}
