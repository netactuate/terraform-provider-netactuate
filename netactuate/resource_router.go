package netactuate

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	routerQueryDelay = 30 * time.Second
)

var routerMu sync.Map

func lockRouter(id int) func() {
	val, _ := routerMu.LoadOrStore(id, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return func() {
		time.Sleep(routerQueryDelay)
		mu.Unlock()
	}
}

func resourceRouter() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRouterCreate,
		ReadContext:   resourceRouterRead,
		UpdateContext: resourceRouterUpdate,
		DeleteContext: resourceRouterDelete,
		Schema: map[string]*schema.Schema{
			"router_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the cloud router.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name your cloud router whatever you wish.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for your cloud router.",
			},
			"ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IPv4 address assigned to the cloud router.",
			},
			"has_default_vrf": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the router has a default VRF.",
			},
			"default_vrf_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the default VRF.",
			},
			"mesh_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The mesh ID if the router is part of a mesh network.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the cloud router (e.g., 'offline', 'active').",
			},
			"plan": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"plan", "package_id"},
				Description:  "Plan/package name for the cloud router (e.g. \"VR2x2x25\"). Resolved to package_id on create.",
			},
			"package_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"plan", "package_id"},
				Description:  "The id of the package for the server which backs your cloud router.",
			},
			"location": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The location to create the cloud router.",
			},
			"location_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"location", "location_id"},
				Description:  "The id of the location to create the cloud router in.",
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The configuration version of the router.",
			},
			"updated_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date when the router was last updated.",
			},
			"can_join_magic_mesh": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the router can join a magic mesh.",
			},
		},
	}
}



func resourceRouterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	id, _ := strconv.Atoi(d.Id())

	router, err := c.GetRouterConfig(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", id, d, &diags)
	setValue("name", router.Metadata.Name, d, &diags)
	setValue("ipv4_address", router.Metadata.IPv4Address, d, &diags)
	setValue("has_default_vrf", router.Metadata.HasDefaultVrf, d, &diags)
	setValue("default_vrf_id", router.DefaultVrfID, d, &diags)
	setValue("mesh_id", router.Metadata.MeshID, d, &diags)
	setValue("status", router.Metadata.Status, d, &diags)
	setValue("version", router.Metadata.Version, d, &diags)
	setValue("updated_on", router.Metadata.UpdatedOn, d, &diags)
	setValue("can_join_magic_mesh", router.Metadata.CanJoinMagicMesh, d, &diags)
	return diags
}

func resourceRouterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	v2 := m.(*ProviderClients).V2
	c := m.(*ProviderClients).V3

	locationID, locDiag := getLocationID(d, v2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}
	var packageID int
	if planName, ok := d.GetOk("plan"); ok {
		id, pkgDiag := getPackageID(planName.(string), v2)
		if pkgDiag != nil {
			return diag.Diagnostics{*pkgDiag}
		}
		packageID = id
		d.Set("package_id", packageID)
	} else {
		packageID = d.Get("package_id").(int)
		plans, err := v2.GetPlans()
		if err == nil {
			for _, p := range plans {
				if p.ID == packageID {
					d.Set("plan", p.Name)
					break
				}
			}
		}
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	req := &gona.CreateRouterRequest{
		PackageID:   packageID,
		LocationID:  locationID,
		Name:        &name,
		Description: &description,
	}

	log.Printf("[DEBUG] Creating cloud router with request: %+v", req)

	router, err := c.CreateRouter(req)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.WaitForRouterReady(router.RouterID); err != nil {
		return diag.Errorf("Router %d created but failed to become ready: %s", router.RouterID, err)
	}

	d.SetId(strconv.Itoa(router.RouterID))

	log.Printf("[DEBUG] Cloud router created with ID: %d", router.RouterID)

	return resourceRouterRead(ctx, d, m)
}

func resourceRouterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.HasChange("name") && !d.HasChange("description") {
		return nil
	}

	unlock := lockRouter(id)
	defer unlock()

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	req := &gona.UpdateRouterRequest{
		Name:        &name,
		Description: &description,
	}

	_, err = c.UpdateRouter(id, req)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRouterRead(ctx, d, m)
}

func resourceRouterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(id)
	defer unlock()

	err = c.DeleteRouter(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
