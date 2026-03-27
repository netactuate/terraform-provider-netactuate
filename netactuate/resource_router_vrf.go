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

func resourceRouterVRF() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterVRFCreate,
		ReadContext:   resourceRouterVRFRead,
		UpdateContext: resourceRouterVRFUpdate,
		DeleteContext: resourceRouterVRFDelete,
		Description:   "Manages an additional VRF on a cloud router. Routers enrolled in netactuate_magic_mesh_router can only use the default VRF and cannot have additional VRFs.",
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			if !diff.NewValueKnown("router_id") {
				return nil
			}
			routerID := diff.Get("router_id").(int)
			c := meta.(*ProviderClients).V3
			return ensureRouterNotInMagicMesh(c, routerID)
		},
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router.",
			},
			"vrf_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the VRF.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A name for the VRF.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the VRF.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterVRFImport,
		},
	}
}

func resourceRouterVRFCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	routerID := d.Get("router_id").(int)

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	unlock := lockRouter(routerID)
	defer unlock()

	if err := ensureRouterNotInMagicMesh(c, routerID); err != nil {
		return diag.FromErr(err)
	}

	createRequest := gona.CreateRouterVRFRequest{
		Name:        &name,
		Description: &description,
	}

	vrf, err := c.CreateRouterVRF(routerID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrf.VrfID))

	return resourceRouterVRFRead(ctx, d, m)
}

func resourceRouterVRFRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, err := parseRouterVRFImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	vrf, err := c.GetRouterVRF(routerID, vrfID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("router_id", routerID, d, &diags)
	setValue("vrf_id", vrf.VrfID, d, &diags)
	setValue("name", vrf.Name, d, &diags)
	setValue("description", vrf.Description, d, &diags)
	d.SetId(fmt.Sprintf("%d/%d", routerID, vrf.VrfID))

	return diags
}

func resourceRouterVRFUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, err := parseRouterVRFImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.HasChanges("name", "description") {
		return nil
	}

	unlock := lockRouter(routerID)
	defer unlock()

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	updateRequest := gona.UpdateRouterVRFRequest{
		Name:        &name,
		Description: &description,
	}

	_, err = c.UpdateRouterVRF(routerID, vrfID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterVRFRead(ctx, d, m)
}

func resourceRouterVRFDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID, vrfID, err := parseRouterVRFImportID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(routerID)
	defer unlock()

	err = c.DeleteRouterVRF(routerID, vrfID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceRouterVRFImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, vrfID, err := parseRouterVRFImportID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d", routerID, vrfID))
	d.Set("router_id", routerID)
	d.Set("vrf_id", vrfID)

	return []*schema.ResourceData{d}, nil
}

func parseRouterVRFImportID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid VRF ID %q, expected \"routerId/vrfId\"", id)
	}

	routerID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid router_id %q: %w", parts[0], err)
	}

	vrfID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vrf_id %q: %w", parts[1], err)
	}

	return routerID, vrfID, nil
}
