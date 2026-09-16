package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	routerQueryDelay = 60 * time.Second
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
		Description:   "Beta. The cloud router family is in beta: behaviour and schema may change. Manages a cloud router.",
		CreateContext: resourceRouterCreate,
		ReadContext:   resourceRouterRead,
		UpdateContext: resourceRouterUpdate,
		DeleteContext: resourceRouterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterImport,
		},
		// A cloud router provisions in about five minutes. Measured on the test account:
		// lax-test 4m59s and dfw-test 3m21s, each build step a minute or two apart.
		//
		// Longer than that means the build has STALLED, not that it is slow, and the
		// timeout should surface that rather than wait it out. Two stall shapes matter:
		// cdg-test hung between "Hardware provisioned" and "Cloud Router software
		// updated" for 3h10m before completing the remaining steps in 97 seconds, and a
		// test router hung between "Cloud Router connectivity established" and "Cloud
		// Router configured" for over three hours and never completed.
		//
		// 15 minutes is three times the normal build and still fails fast on a stall.
		// Operators can raise it per resource with a timeouts block if they choose.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
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
				Type:             schema.TypeString,
				ForceNew:         true,
				Optional:         true,
				Computed:         true,
				ExactlyOneOf:     []string{"location", "location_id"},
				DiffSuppressFunc: suppressLocationDiff,
				Description:      "The location to create the cloud router.",
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
	id, err := parseRouterID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	config, err := c.GetRouterConfig(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	router, err := c.GetRouter(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("router_id", id, d, &diags)
	setValue("name", config.Metadata.Name, d, &diags)
	if router.Description != nil {
		setValue("description", *router.Description, d, &diags)
	}
	setValue("ipv4_address", config.Metadata.IPv4Address, d, &diags)
	setValue("has_default_vrf", config.Metadata.HasDefaultVrf, d, &diags)
	setValue("default_vrf_id", config.DefaultVrfID, d, &diags)
	setValue("mesh_id", config.Metadata.MeshID, d, &diags)
	setValue("status", config.Metadata.Status, d, &diags)
	if config.Metadata.Location != nil {
		setLocationPreserveFormat(config.Metadata.Location.Name, d, &diags)
		setValue("location_id", config.Metadata.Location.ID, d, &diags)
	}
	setValue("version", config.Metadata.Version, d, &diags)
	setValue("updated_on", config.Metadata.UpdatedOn, d, &diags)
	setValue("can_join_magic_mesh", config.Metadata.CanJoinMagicMesh, d, &diags)
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

	d.SetId(strconv.Itoa(router.RouterID))

	// Honour the resource's create timeout instead of the SDK's 10 minute default.
	// Router provisioning on this platform is not reliably fast: one router on the test
	// account took 3h12m to become ready. A 10 minute give-up marks a perfectly healthy
	// router as failed, and the next apply then destroys and recreates it.
	if err := c.WaitForRouterReadyTimeout(router.RouterID, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.Errorf("Router %d created but failed to become ready: %s", router.RouterID, err)
	}

	log.Printf("[DEBUG] Cloud router created with ID: %d", router.RouterID)

	return resourceRouterRead(ctx, d, m)
}

func resourceRouterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := parseRouterID(d.Id())
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

	id, err := parseRouterID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	unlock := lockRouter(id)
	defer unlock()

	err = c.DeleteRouter(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceRouterImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	routerID, err := parseRouterID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"routerId\"", d.Id())
	}

	d.SetId(strconv.Itoa(routerID))
	d.Set("router_id", routerID)

	return []*schema.ResourceData{d}, nil
}

func parseRouterID(id string) (int, error) {
	routerID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid router ID %q, expected \"routerId\": %w", id, err)
	}
	return routerID, nil
}
