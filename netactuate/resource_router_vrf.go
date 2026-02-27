package netactuate

import (
	"context"
	"strconv"

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
			StateContext: schema.ImportStatePassthroughContext,
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

	createRequest := gona.CreateRouterVRFRequest{
		Name:        &name,
		Description: &description,
	}

	vrf, err := c.CreateRouterVRF(routerID, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(vrf.VrfID))

	return resourceRouterVRFRead(ctx, d, m)
}

func resourceRouterVRFRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID, err := strconv.Atoi(d.Id())
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

	return diags
}

func resourceRouterVRFUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	routerID := d.Get("router_id").(int)
	vrfID, err := strconv.Atoi(d.Id())
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

	routerID := d.Get("router_id").(int)
	vrfID, err := strconv.Atoi(d.Id())

	unlock := lockRouter(routerID)
	defer unlock()

	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteRouterVRF(routerID, vrfID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
