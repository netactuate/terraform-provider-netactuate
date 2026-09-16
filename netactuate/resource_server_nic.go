package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceServerNIC() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerNICCreate,
		ReadContext:   resourceServerNICRead,
		UpdateContext: resourceServerNICUpdate,
		DeleteContext: resourceServerNICDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceServerNICImport,
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Server package ID to attach the NIC to.",
			},
			"customer_vlan_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Existing customer VLAN ID to attach.",
			},
			"attach_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "NIC ordering on the server.",
			},
			"nic_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "NIC ID assigned by the API.",
			},
		},
	}
}

func resourceServerNICCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgID := d.Get("mbpkgid").(int)
	customerVLANID := d.Get("customer_vlan_id").(int)

	created, err := c.AttachServerNIC(mbpkgID, &gona.ServerNICAttachRequest{
		CustomerVLANID: customerVLANID,
	})
	if err != nil {
		return diag.Errorf("attaching VLAN %d NIC to server %d: %s", customerVLANID, mbpkgID, err)
	}

	nicID := created.NICID
	if nicID == 0 {
		nic, err := findServerNICByCustomerVLAN(c, mbpkgID, customerVLANID)
		if err != nil {
			return diag.FromErr(err)
		}
		nicID = nic.NICID
	}

	d.SetId(formatServerNICID(mbpkgID, nicID))
	return resourceServerNICRead(ctx, d, m)
}

func resourceServerNICRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbpkgID, nicID, err := parseServerNICID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	nic, err := findServerNICByID(c, mbpkgID, nicID)
	if err != nil {
		return diag.FromErr(err)
	}
	if nic == nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics
	setValue("mbpkgid", mbpkgID, d, &diags)
	setValue("nic_id", nic.NICID, d, &diags)
	setValue("customer_vlan_id", nic.CustomerVLANID, d, &diags)
	setValue("attach_order", nic.AttachOrder, d, &diags)
	return diags
}

func resourceServerNICUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbpkgID, nicID, err := parseServerNICID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.ServerNICUpdateRequest{
		MBPkgID:        mbpkgID,
		CustomerVLANID: d.Get("customer_vlan_id").(int),
		AttachOrder:    d.Get("attach_order").(int),
	}
	if _, err := c.UpdateServerNIC(nicID, req); err != nil {
		return diag.Errorf("updating server NIC %d on server %d: %s", nicID, mbpkgID, err)
	}

	return resourceServerNICRead(ctx, d, m)
}

func resourceServerNICDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbpkgID, nicID, err := parseServerNICID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DetachServerNIC(mbpkgID, nicID); err != nil {
		return diag.Errorf("detaching server NIC %d from server %d: %s", nicID, mbpkgID, err)
	}
	return nil
}

func resourceServerNICImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	mbpkgID, nicID, err := parseServerNICID(d.Id())
	if err != nil {
		return nil, err
	}
	if err := d.Set("mbpkgid", mbpkgID); err != nil {
		return nil, err
	}
	if err := d.Set("nic_id", nicID); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func findServerNICByID(c *gona.Client, mbpkgID, nicID int) (*gona.ServerNIC, error) {
	nics, err := c.GetServerNICs(mbpkgID)
	if err != nil {
		return nil, err
	}
	for i := range nics {
		if nics[i].NICID == nicID {
			return &nics[i], nil
		}
	}
	return nil, nil
}

func findServerNICByCustomerVLAN(c *gona.Client, mbpkgID, customerVLANID int) (*gona.ServerNIC, error) {
	nics, err := c.GetServerNICs(mbpkgID)
	if err != nil {
		return nil, err
	}
	var match *gona.ServerNIC
	for i := range nics {
		if nics[i].CustomerVLANID != customerVLANID {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple NICs on server %d match customer_vlan_id %d", mbpkgID, customerVLANID)
		}
		match = &nics[i]
	}
	if match == nil {
		return nil, fmt.Errorf("no NIC matching customer_vlan_id %d found on server %d after attach", customerVLANID, mbpkgID)
	}
	if match.NICID == 0 {
		return nil, fmt.Errorf("NIC matching customer_vlan_id %d on server %d did not include a nic_id", customerVLANID, mbpkgID)
	}
	return match, nil
}

func formatServerNICID(mbpkgID, nicID int) string {
	return fmt.Sprintf("%d/%d", mbpkgID, nicID)
}

func parseServerNICID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid server NIC ID %q, expected \"mbpkgid/nic_id\"", id)
	}
	mbpkgID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid mbpkgid %q: %w", parts[0], err)
	}
	nicID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid nic_id %q: %w", parts[1], err)
	}
	return mbpkgID, nicID, nil
}
