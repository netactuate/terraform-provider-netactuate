package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceAccessControlSubnet() *schema.Resource {
	return &schema.Resource{
		Description:   accessControlSubnetWarning,
		CreateContext: resourceAccessControlSubnetCreate,
		ReadContext:   resourceAccessControlSubnetRead,
		UpdateContext: resourceAccessControlSubnetUpdate,
		DeleteContext: resourceAccessControlSubnetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: accessControlSubnetFieldsSchema(true),
	}
}

func resourceAccessControlSubnetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	subnet, err := c.CreateAccessControlSubnet(&gona.CreateAccessControlSubnetRequest{
		Label:  d.Get("label").(string),
		Subnet: d.Get("subnet").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	if subnet == nil || subnet.ID == "" {
		return diag.Errorf("access control subnet %q was created without an ID", d.Get("label").(string))
	}

	d.SetId(subnet.ID.String())
	return resourceAccessControlSubnetRead(ctx, d, m)
}

func resourceAccessControlSubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	subnet, err := c.GetAccessControlSubnet(d.Id())
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setAccessControlSubnetState(subnet, d, &diags)
	return diags
}

func resourceAccessControlSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.UpdateAccessControlSubnetRequest{}
	if d.HasChange("label") {
		label := d.Get("label").(string)
		req.Label = &label
	}
	if d.HasChange("subnet") {
		subnet := d.Get("subnet").(string)
		req.Subnet = &subnet
	}

	if _, err := c.UpdateAccessControlSubnet(d.Id(), req); err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return resourceAccessControlSubnetRead(ctx, d, m)
}

func resourceAccessControlSubnetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	if err := c.DeleteAccessControlSubnet(d.Id()); err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func setAccessControlSubnetState(subnet *gona.AccessControlSubnet, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("access_control_subnet_id", subnet.ID.String(), d, diags)
	setValue("label", subnet.Label, d, diags)
	setValue("subnet", subnet.Subnet, d, diags)
}
