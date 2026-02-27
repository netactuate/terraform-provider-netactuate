package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceMagicMeshRouter() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMagicMeshRouterCreate,
		ReadContext:   resourceMagicMeshRouterRead,
		DeleteContext: resourceMagicMeshRouterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceMagicMeshRouterImport,
		},
		Schema: map[string]*schema.Schema{
			"mesh_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the magic mesh.",
			},
			"router_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the cloud router to add to the mesh. Only routers with canJoinMagicMesh=true are eligible.",
			},
		},
	}
}

func resourceMagicMeshRouterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID := d.Get("mesh_id").(int)
	routerID := d.Get("router_id").(int)

	req := &gona.AddMeshRouterRequest{
		RouterID: routerID,
	}

	err := c.AddRouterToMesh(meshID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", meshID, routerID))

	return resourceMagicMeshRouterRead(ctx, d, m)
}

func resourceMagicMeshRouterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID, routerID, err := parseMeshRouterID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	routers, err := c.ListMeshRouters(meshID)
	if err != nil {
		d.SetId("")
		return nil
	}

	found := false
	for _, r := range routers {
		if r.RouterID == routerID {
			found = true
			break
		}
	}

	if !found {
		log.Printf("[WARN] Router %d not found in mesh %d, removing from state", routerID, meshID)
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics
	setValue("mesh_id", meshID, d, &diags)
	setValue("router_id", routerID, d, &diags)

	return diags
}

func resourceMagicMeshRouterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID, routerID, err := parseMeshRouterID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.RemoveRouterFromMesh(meshID, routerID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceMagicMeshRouterImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	meshID, routerID, err := parseMeshRouterID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"meshId/routerId\" (e.g. \"42/10\")", d.Id())
	}

	d.SetId(fmt.Sprintf("%d/%d", meshID, routerID))
	d.Set("mesh_id", meshID)
	d.Set("router_id", routerID)

	return []*schema.ResourceData{d}, nil
}

func parseMeshRouterID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid mesh router ID %q, expected \"meshId/routerId\"", id)
	}
	meshID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid mesh_id %q: %w", parts[0], err)
	}
	routerID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid router_id %q: %w", parts[1], err)
	}
	return meshID, routerID, nil
}
