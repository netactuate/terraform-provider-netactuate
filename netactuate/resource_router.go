package netactuate

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

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
			"package_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the package for the server which backs your cloud router.",
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
	c := m.(*ProviderClients).V3

	locationID, locDiag := getLocationID(d, m.(*ProviderClients).V2)
	if locDiag != nil {
		return diag.Diagnostics{*locDiag}
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	req := &gona.CreateRouterRequest{
		PackageID:   d.Get("package_id").(int),
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
		log.Println("[DEBUG] No relevant changes, skipping update")
		return nil
	}

	log.Printf("[DEBUG] Updating cloud router with ID: %d", id)

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

	log.Printf("[DEBUG] Deleting cloud router with ID: %d", id)

	err = c.DeleteRouter(id)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Cloud router %d deleted successfully", id)

	return nil
}
