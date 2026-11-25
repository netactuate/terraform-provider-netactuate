package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
	"strconv"
	"strings"
)

func drop3Optional(t schema.ValueType, immutable bool) *schema.Schema {
	return &schema.Schema{Type: t, Optional: true, ForceNew: immutable}
}
func drop3Computed(t schema.ValueType) *schema.Schema { return &schema.Schema{Type: t, Computed: true} }
func drop3Set(d *schema.ResourceData, values map[string]interface{}) diag.Diagnostics {
	var out diag.Diagnostics
	for k, v := range values {
		setValue(k, v, d, &out)
	}
	return out
}
func drop3IDs(d *schema.ResourceData) (int, int, error) {
	a, b, e := drop2Parts(d)
	if e != nil {
		return 0, 0, e
	}
	n, e := strconv.Atoi(b)
	if e != nil || n <= 0 {
		return 0, 0, fmt.Errorf("invalid resource ID")
	}
	return a, n, nil
}
func drop3SingleID(d *schema.ResourceData) (int, error) {
	n, e := strconv.Atoi(d.Id())
	if e != nil || n <= 0 {
		return 0, fmt.Errorf("invalid resource ID")
	}
	return n, nil
}
func drop3DeleteResult(d *schema.ResourceData, e error) diag.Diagnostics {
	if e != nil && !gona.IsV3NotFound(e) {
		return drop2Error(e)
	}
	d.SetId("")
	return nil
}
func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{CreateContext: drop3GroupCreate, ReadContext: drop3GroupRead, UpdateContext: drop3GroupUpdate, DeleteContext: drop3GroupDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("security_group_id", false, false)}, Schema: map[string]*schema.Schema{
		"security_group_id": drop3Computed(schema.TypeInt),
		"name":              {Type: schema.TypeString, Required: true, ValidateFunc: validation.StringLenBetween(1, 64)},
		"description":       {Type: schema.TypeString, Optional: true, ValidateFunc: validation.StringLenBetween(0, 255)},
		"is_default":        drop3Computed(schema.TypeBool), "member_count": drop3Computed(schema.TypeInt)}}
}
func drop3GroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, e := m.(*ProviderClients).V3.CreateSecurityGroup(&gona.CreateSecurityGroupRequest{Name: d.Get("name").(string), Description: d.Get("description").(string)})
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(strconv.Itoa(id))
	return drop3GroupRead(ctx, d, m)
}
func drop3GroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, e := drop3SingleID(d)
	if e != nil {
		return drop2Error(e)
	}
	r, e := m.(*ProviderClients).V3.GetSecurityGroup(id)
	if e != nil {
		return drop2ReadError(d, e)
	}
	return drop3Set(d, map[string]interface{}{"security_group_id": id, "name": r.Name, "description": r.Description, "is_default": r.IsDefault, "member_count": r.MemberCount})
}
func drop3GroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, e := drop3SingleID(d)
	if e != nil {
		return drop2Error(e)
	}
	req := &gona.UpdateSecurityGroupRequest{}
	if d.HasChange("name") {
		v := d.Get("name").(string)
		req.Name = &v
	}
	if d.HasChange("description") {
		v := d.Get("description").(string)
		req.Description = &v
	}
	if e = m.(*ProviderClients).V3.UpdateSecurityGroup(id, req); e != nil {
		return drop2Error(e)
	}
	return drop3GroupRead(ctx, d, m)
}
func drop3GroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, e := drop3SingleID(d)
	if e != nil {
		return drop2Error(e)
	}
	return drop3DeleteResult(d, m.(*ProviderClients).V3.DeleteSecurityGroup(id))
}
func resourceSecurityGroupRule() *schema.Resource {
	fields := map[string]*schema.Schema{
		"security_group_id": drop2IDSchema(), "ip_version": {Type: schema.TypeInt, Required: true, ForceNew: true, ValidateFunc: validation.IntInSlice([]int{4, 6})},
		"direction": {Type: schema.TypeString, Optional: true, Default: "inbound", ForceNew: true, ValidateFunc: validation.StringInSlice([]string{"inbound"}, false)},
		"protocol":  {Type: schema.TypeString, Optional: true, Computed: true, ForceNew: true, DiffSuppressFunc: protocolCaseInsensitive}, "port_start": {Type: schema.TypeInt, Optional: true, ForceNew: true, RequiredWith: []string{"port_end"}, ValidateFunc: validation.IntBetween(1, 65535)}, "port_end": {Type: schema.TypeInt, Optional: true, ForceNew: true, RequiredWith: []string{"port_start"}, ValidateFunc: validation.IntBetween(1, 65535)},
		"network":                  {Type: schema.TypeString, Optional: true, ForceNew: true, ExactlyOneOf: []string{"network", "source_security_group_id"}, ValidateFunc: validation.IsCIDR},
		"source_security_group_id": {Type: schema.TypeInt, Optional: true, ForceNew: true, ExactlyOneOf: []string{"network", "source_security_group_id"}, ValidateFunc: validation.IntAtLeast(1)}, "description": drop3Optional(schema.TypeString, true)}
	return &schema.Resource{Schema: fields, CreateContext: drop3RuleCreate, ReadContext: drop3RuleRead, DeleteContext: drop3RuleDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("security_group_id", true, false)}}
}
func drop3RuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g := d.Get("security_group_id").(int)
	req := &gona.CreateSecurityGroupRuleRequest{IPVersion: d.Get("ip_version").(int), Direction: d.Get("direction").(string), Protocol: d.Get("protocol").(string), Network: d.Get("network").(string), SecurityGroupID: d.Get("source_security_group_id").(int), Description: d.Get("description").(string)}
	if v, ok := d.GetOk("port_start"); ok {
		req.Port = &gona.SecurityGroupPort{Start: v.(int), End: d.Get("port_end").(int)}
	}
	id, e := m.(*ProviderClients).V3.CreateSecurityGroupRule(g, req)
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(fmt.Sprintf("%d/%d", g, id))
	return drop3RuleRead(ctx, d, m)
}
func drop3RuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	rows, e := m.(*ProviderClients).V3.ListSecurityGroupRules(g)
	if e != nil {
		return drop2ReadError(d, e)
	}
	for _, r := range rows {
		if r.RuleID == id {
			start, end := r.PortStart, r.PortEnd
			if r.Port != nil {
				start, end = r.Port.Start, r.Port.End
			}
			return drop3Set(d, map[string]interface{}{"security_group_id": g, "ip_version": r.IPVersion, "direction": r.Direction, "protocol": r.Protocol, "port_start": start, "port_end": end, "network": r.Network, "source_security_group_id": r.SecurityGroupID, "description": r.Description})
		}
	}
	d.SetId("")
	return nil
}
func drop3RuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	return drop3DeleteResult(d, m.(*ProviderClients).V3.DeleteSecurityGroupRule(g, id))
}
func resourceSecurityGroupMember() *schema.Resource {
	return &schema.Resource{CreateContext: drop3MemberCreate, ReadContext: drop3MemberRead, DeleteContext: drop3MemberDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("security_group_id", true, false)}, Schema: map[string]*schema.Schema{
		"security_group_id": drop2IDSchema(), "vpc_id": drop2IDSchema(),
		"mbpkgid":      {Type: schema.TypeInt, Optional: true, ForceNew: true, ExactlyOneOf: []string{"mbpkgid", "node_pool_id"}, ValidateFunc: validation.IntAtLeast(1)},
		"node_pool_id": {Type: schema.TypeInt, Optional: true, ForceNew: true, ExactlyOneOf: []string{"mbpkgid", "node_pool_id"}, ValidateFunc: validation.IntAtLeast(1)}, "enforcement_state": drop3Computed(schema.TypeString)}}
}
func drop3MemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g := d.Get("security_group_id").(int)
	id, e := m.(*ProviderClients).V3.CreateSecurityGroupMember(g, &gona.CreateSecurityGroupMemberRequest{VPCID: d.Get("vpc_id").(int), MBPkgID: d.Get("mbpkgid").(int), NodePoolID: d.Get("node_pool_id").(int)})
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(fmt.Sprintf("%d/%d", g, id))
	return drop3MemberRead(ctx, d, m)
}
func drop3MemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	rows, e := m.(*ProviderClients).V3.ListSecurityGroupMembers(g)
	if e != nil {
		return drop2ReadError(d, e)
	}
	for _, r := range rows {
		if r.MemberID == id {
			return drop3Set(d, map[string]interface{}{"security_group_id": g, "vpc_id": r.VPCID, "mbpkgid": r.MBPkgID, "node_pool_id": r.NodePoolID, "enforcement_state": r.EnforcementState})
		}
	}
	d.SetId("")
	return nil
}
func drop3MemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	g, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	return drop3DeleteResult(d, m.(*ProviderClients).V3.DeleteSecurityGroupMember(g, id))
}
func resourceVPCSecurityGroupEnforcement() *schema.Resource {
	return &schema.Resource{CreateContext: drop3EnforcementSet, UpdateContext: drop3EnforcementSet, ReadContext: drop3EnforcementRead, DeleteContext: drop3EnforcementDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("vpc_id", false, false)}, Schema: map[string]*schema.Schema{"vpc_id": drop2IDSchema(), "enabled": {Type: schema.TypeBool, Optional: true, Default: true}, "enabled_on": drop3Computed(schema.TypeString)}}
}
func drop3EnforcementSet(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v := d.Get("vpc_id").(int)
	_, e := m.(*ProviderClients).V3.SetVPCSecurityGroupEnforcement(v, &gona.SetVPCSecurityGroupEnforcementRequest{Enabled: d.Get("enabled").(bool)})
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(strconv.Itoa(v))
	return drop3EnforcementRead(ctx, d, m)
}
func drop3EnforcementRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, e := drop3SingleID(d)
	if e != nil {
		return drop2Error(e)
	}
	r, e := m.(*ProviderClients).V3.GetVPCSecurityGroupEnforcement(v)
	if e != nil {
		return drop2ReadError(d, e)
	}
	return drop3Set(d, map[string]interface{}{"vpc_id": v, "enabled": r.Enabled, "enabled_on": r.EnabledOn})
}
func drop3EnforcementDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, e := drop3SingleID(d)
	if e != nil {
		return drop2Error(e)
	}
	_, e = m.(*ProviderClients).V3.SetVPCSecurityGroupEnforcement(v, &gona.SetVPCSecurityGroupEnforcementRequest{Enabled: false})
	return drop3DeleteResult(d, e)
}
func resourceVPCSubnet() *schema.Resource {
	fields := map[string]*schema.Schema{"vpc_id": drop2IDSchema(), "network": {Type: schema.TypeString, Required: true, ForceNew: true, ValidateFunc: validation.IsCIDR}, "label": {Type: schema.TypeString, Required: true, ForceNew: true, ValidateFunc: validation.StringLenBetween(1, 255)}, "subnet_id": drop3Computed(schema.TypeInt), "is_default": drop3Computed(schema.TypeBool), "prefix_length": drop3Computed(schema.TypeInt)}
	for _, k := range []string{"state", "state_message", "created_on", "ready_on", "delete_requested_on", "deleted_on"} {
		fields[k] = drop3Computed(schema.TypeString)
	}
	return &schema.Resource{Schema: fields, CreateContext: drop3SubnetCreate, ReadContext: drop3SubnetRead, DeleteContext: drop3SubnetDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("vpc_id", true, false)}}
}
func drop3SubnetValues(r *gona.VPCSubnet) map[string]interface{} {
	return map[string]interface{}{"subnet_id": r.SubnetID, "network": r.Network, "label": r.Label, "prefix_length": r.PrefixLength, "is_default": r.IsDefault, "state": r.State, "state_message": r.StateMessage, "created_on": r.CreatedOn, "ready_on": r.ReadyOn, "delete_requested_on": r.DeleteRequestedOn, "deleted_on": r.DeletedOn}
}
func drop3WaitSubnet(ctx context.Context, d *schema.ResourceData, c *gona.V3Client, v, id int, deleted bool) error {
	timeout := schema.TimeoutCreate
	if deleted {
		timeout = schema.TimeoutDelete
	}
	return drop2Wait(ctx, d.Timeout(timeout), func() (bool, error) {
		r, e := c.GetVPCSubnet(v, id)
		if deleted && gona.IsV3NotFound(e) {
			return true, nil
		}
		if e != nil {
			return false, e
		}
		if deleted {
			return r.DeletedOn != "", nil
		}
		if r.State == "failed" {
			return false, fmt.Errorf("%s", r.StateMessage)
		}
		return r.State == "ready" || r.ReadyOn != "" || r.State == "", nil
	})
}
func drop3SubnetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v := d.Get("vpc_id").(int)
	c := m.(*ProviderClients).V3
	id, e := c.CreateVPCSubnet(v, &gona.CreateVPCSubnetRequest{Network: d.Get("network").(string), Label: d.Get("label").(string)})
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(fmt.Sprintf("%d/%d", v, id))
	if e = drop3WaitSubnet(ctx, d, c, v, id, false); e != nil {
		return drop2Error(e)
	}
	return drop3SubnetRead(ctx, d, m)
}
func drop3SubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	r, e := m.(*ProviderClients).V3.GetVPCSubnet(v, id)
	if e != nil {
		return drop2ReadError(d, e)
	}
	if r.DeletedOn != "" {
		d.SetId("")
		return nil
	}
	values := drop3SubnetValues(r)
	values["vpc_id"] = v
	return drop3Set(d, values)
}
func drop3SubnetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	c := m.(*ProviderClients).V3
	e = c.DeleteVPCSubnet(v, id)
	if gona.IsV3NotFound(e) {
		return drop3DeleteResult(d, e)
	}
	if e != nil {
		return drop2Error(e)
	}
	return drop3DeleteResult(d, drop3WaitSubnet(ctx, d, c, v, id, true))
}
func resourceVPCServerAttachment() *schema.Resource {
	return &schema.Resource{CreateContext: drop3ServerCreate, ReadContext: drop3ServerRead, DeleteContext: drop3ServerDelete, Timeouts: drop2Timeouts(), Importer: &schema.ResourceImporter{StateContext: drop2Import("vpc_id", true, false)}, Schema: map[string]*schema.Schema{"vpc_id": drop2IDSchema(), "mbpkgid": drop2IDSchema(), "subnet_id": {Type: schema.TypeInt, Optional: true, Default: 0, ForceNew: true, ValidateFunc: validation.IntAtLeast(0)}, "nic_id": drop3Computed(schema.TypeInt), "state": drop3Computed(schema.TypeString), "address": drop3Computed(schema.TypeString), "membership": drop3Computed(schema.TypeString)}}
}
func drop3ServerValues(r *gona.VPCServer) map[string]interface{} {
	return map[string]interface{}{"mbpkgid": r.MBPkgID, "subnet_id": r.SubnetID, "nic_id": r.NICID, "state": r.State, "address": r.Address, "membership": r.Membership}
}
func drop3ServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id := d.Get("vpc_id").(int), d.Get("mbpkgid").(int)
	_, e := m.(*ProviderClients).V3.AttachVPCServer(v, &gona.AttachVPCServerRequest{MBPkgID: id, SubnetID: d.Get("subnet_id").(int)})
	if e != nil {
		return drop2Error(e)
	}
	d.SetId(fmt.Sprintf("%d/%d", v, id))
	return drop3ServerRead(ctx, d, m)
}
func drop3ServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	rows, e := m.(*ProviderClients).V3.ListVPCServers(v)
	if e != nil {
		return drop2ReadError(d, e)
	}
	for _, r := range rows {
		if r.MBPkgID == id {
			values := drop3ServerValues(&r)
			values["vpc_id"] = v
			return drop3Set(d, values)
		}
	}
	d.SetId("")
	return nil
}
func drop3ServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v, id, e := drop3IDs(d)
	if e != nil {
		return drop2Error(e)
	}
	return drop3DeleteResult(d, m.(*ProviderClients).V3.DetachVPCServer(v, id))
}

// protocolCaseInsensitive keeps a configured "tcp" from replacing a rule the API reads back as "TCP" (LD-241).
func protocolCaseInsensitive(_, old, new string, _ *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}
