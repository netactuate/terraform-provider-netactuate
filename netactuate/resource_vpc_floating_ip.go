package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceVPCFloatingIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCFloatingIPCreate,
		ReadContext:   resourceVPCFloatingIPRead,
		UpdateContext: resourceVPCFloatingIPUpdate,
		DeleteContext: resourceVPCFloatingIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCFloatingIPImport,
		},
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the parent VPC",
			},
			"floating_ip_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The floating IP ID assigned by the API",
			},
			"ip_version": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{4, 6})),
				Description:      "IP version (4 or 6)",
			},
			"ptr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "PTR record for the floating IP",
			},
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The assigned floating IP address",
			},
			"is_primary": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is the primary floating IP for its IP version",
			},
		},
	}
}

func resourceVPCFloatingIPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)

	req := &gona.CreateVPCFloatingIPRequest{
		IPVersion: d.Get("ip_version").(int),
	}

	if v, ok := d.GetOk("ptr"); ok {
		req.PTR = v.(string)
	}

	result, err := c.CreateVPCFloatingIP(vpcID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, result.FloatingIPID))
	log.Printf("[DEBUG] Floating IP %d created in VPC %d", result.FloatingIPID, vpcID)

	return resourceVPCFloatingIPRead(ctx, d, m)
}

func resourceVPCFloatingIPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, fipID, err := parseFloatingIPID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	fip, err := c.GetVPCFloatingIP(vpcID, fipID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Floating IP %d in VPC %d not found, removing from state", fipID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("floating_ip_id", fip.FloatingIPID, d, &diags)
	setValue("ip_version", fip.IPVersion, d, &diags)
	setValue("ptr", fip.PTR, d, &diags)
	setValue("address", fip.Address, d, &diags)
	setValue("is_primary", fip.IsPrimary, d, &diags)

	return diags
}

func resourceVPCFloatingIPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, fipID, err := parseFloatingIPID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("ptr") {
		req := &gona.UpdateVPCFloatingIPRequest{
			PTR: d.Get("ptr").(string),
		}
		if err := c.UpdateVPCFloatingIP(vpcID, fipID, req); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceVPCFloatingIPRead(ctx, d, m)
}

func resourceVPCFloatingIPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, fipID, err := parseFloatingIPID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting floating IP %d from VPC %d", fipID, vpcID)

	if err := c.DeleteVPCFloatingIP(vpcID, fipID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Floating IP %d in VPC %d already deleted", fipID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceVPCFloatingIPImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	vpcID, fipID, err := parseFloatingIPID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"vpcId/floatingIpId\": %w", d.Id(), err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, fipID))
	d.Set("vpc_id", vpcID)

	return []*schema.ResourceData{d}, nil
}

func parseFloatingIPID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid floating IP ID %q, expected \"vpcId/floatingIpId\"", id)
	}
	vpcID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vpc_id %q: %w", parts[0], err)
	}
	fipID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid floating_ip_id %q: %w", parts[1], err)
	}
	return vpcID, fipID, nil
}
