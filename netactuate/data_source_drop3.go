package netactuate

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
	"strconv"
)

func drop3ListSource(key string, fields map[string]*schema.Schema, vpc bool, read func(*schema.ResourceData, *ProviderClients) ([]map[string]interface{}, error)) *schema.Resource {
	args := map[string]*schema.Schema{key: {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: fields}}}
	if vpc {
		args["vpc_id"] = drop2IDSchema()
		args["vpc_id"].ForceNew = false
	}
	return &schema.Resource{Schema: args, ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		rows, e := read(d, m.(*ProviderClients))
		if e != nil {
			return drop2Error(e)
		}
		id := "security-groups"
		if vpc {
			id = strconv.Itoa(d.Get("vpc_id").(int))
		}
		d.SetId(id)
		return drop3Set(d, map[string]interface{}{key: rows})
	}}
}
func drop3GroupSource(vpc bool) *schema.Resource {
	fields := map[string]*schema.Schema{}
	for _, k := range []string{"security_group_id", "default_for_vpc_id", "member_count"} {
		fields[k] = drop3Computed(schema.TypeInt)
	}
	for _, k := range []string{"name", "description", "created_on", "modified_on"} {
		fields[k] = drop3Computed(schema.TypeString)
	}
	fields["is_default"] = drop3Computed(schema.TypeBool)
	return drop3ListSource("security_groups", fields, vpc, func(d *schema.ResourceData, c *ProviderClients) ([]map[string]interface{}, error) {
		var rows []gona.SecurityGroup
		var e error
		if vpc {
			rows, e = c.V3.ListVPCSecurityGroups(d.Get("vpc_id").(int))
		} else {
			rows, e = c.V3.ListSecurityGroups()
		}
		if e != nil {
			return nil, e
		}
		out := []map[string]interface{}{}
		for _, r := range rows {
			out = append(out, map[string]interface{}{"security_group_id": r.SecurityGroupID, "default_for_vpc_id": r.DefaultForVPCID, "member_count": r.MemberCount, "name": r.Name, "description": r.Description, "is_default": r.IsDefault, "created_on": r.CreatedOn, "modified_on": r.ModifiedOn})
		}
		return out, nil
	})
}
func dataSourceSecurityGroups() *schema.Resource    { return drop3GroupSource(false) }
func dataSourceVPCSecurityGroups() *schema.Resource { return drop3GroupSource(true) }
func dataSourceVPCSubnets() *schema.Resource {
	fields := resourceVPCSubnet().Schema
	delete(fields, "vpc_id")
	for _, s := range fields {
		s.Required = false
		s.Optional = false
		s.Computed = true
		s.ForceNew = false
		s.ValidateFunc = nil
	}
	return drop3ListSource("subnets", fields, true, func(d *schema.ResourceData, c *ProviderClients) ([]map[string]interface{}, error) {
		rows, e := c.V3.ListVPCSubnets(d.Get("vpc_id").(int))
		if e != nil {
			return nil, e
		}
		out := []map[string]interface{}{}
		for _, r := range rows {
			out = append(out, drop3SubnetValues(&r))
		}
		return out, nil
	})
}
func dataSourceVPCServers() *schema.Resource {
	fields := resourceVPCServerAttachment().Schema
	delete(fields, "vpc_id")
	for _, s := range fields {
		s.Required = false
		s.Optional = false
		s.Computed = true
		s.ForceNew = false
		s.Default = nil
		s.ValidateFunc = nil
		s.ValidateDiagFunc = nil
	}
	fields["fqdn"] = drop3Computed(schema.TypeString)
	return drop3ListSource("servers", fields, true, func(d *schema.ResourceData, c *ProviderClients) ([]map[string]interface{}, error) {
		rows, e := c.V3.ListVPCServers(d.Get("vpc_id").(int))
		if e != nil {
			return nil, e
		}
		out := []map[string]interface{}{}
		for _, r := range rows {
			v := drop3ServerValues(&r)
			v["fqdn"] = r.FQDN
			out = append(out, v)
		}
		return out, nil
	})
}
