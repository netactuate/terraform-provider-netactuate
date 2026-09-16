package netactuate

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudRoutingMeshes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingMeshesRead,
		Schema: map[string]*schema.Schema{
			"meshes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Magic meshes visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"mesh_id":     {Type: schema.TypeInt, Computed: true, Description: "Magic mesh ID."},
					"name":        {Type: schema.TypeString, Computed: true, Description: "Magic mesh name."},
					"description": {Type: schema.TypeString, Computed: true, Description: "Magic mesh description."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingMeshesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meshes, err := m.(*ProviderClients).V3.ListMagicMeshes()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(meshes))
	for i, mesh := range meshes {
		description := ""
		if mesh.Description != nil {
			description = *mesh.Description
		}
		out[i] = map[string]interface{}{
			"mesh_id":     mesh.MeshID,
			"name":        mesh.Name,
			"description": description,
		}
	}
	var diags diag.Diagnostics
	setValue("meshes", out, d, &diags)
	d.SetId("cloud-routing-meshes")
	return diags
}

func dataSourceCloudRoutingRouters() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRoutersRead,
		Schema: map[string]*schema.Schema{
			"routers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Cloud routers visible to the account.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"name":        {Type: schema.TypeString, Computed: true, Description: "Cloud router name."},
					"description": {Type: schema.TypeString, Computed: true, Description: "Cloud router description."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingRoutersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routers, err := m.(*ProviderClients).V3.ListRouters()
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(routers))
	for i, router := range routers {
		description := ""
		if router.Description != nil {
			description = *router.Description
		}
		out[i] = map[string]interface{}{
			"name":        router.Name,
			"description": description,
		}
	}
	var diags diag.Diagnostics
	setValue("routers", out, d, &diags)
	d.SetId("cloud-routing-routers")
	return diags
}

func dataSourceCloudRoutingRouterInterfaces() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterInterfacesRead,
		Schema: map[string]*schema.Schema{
			"router_id":  {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"interfaces": rawJSONDataSourceSchema("Cloud router configuration interfaces as compact JSON."),
		},
	}
}

func dataSourceCloudRoutingRouterInterfacesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	raw, err := m.(*ProviderClients).V3.ListRouterConfigInterfaces(routerID)
	if err != nil {
		return diag.FromErr(err)
	}
	out, err := compactSurfaceJSON(raw)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("interfaces", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-interfaces", routerID))
	return diags
}

func dataSourceCloudRoutingRouterVRFs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudRoutingRouterVRFsRead,
		Schema: map[string]*schema.Schema{
			"router_id": {Type: schema.TypeInt, Required: true, Description: "Cloud router ID."},
			"vrfs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "VRFs configured on the cloud router.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"vrf_id":      {Type: schema.TypeInt, Computed: true, Description: "VRF ID."},
					"name":        {Type: schema.TypeString, Computed: true, Description: "VRF name."},
					"description": {Type: schema.TypeString, Computed: true, Description: "VRF description."},
				}},
			},
		},
	}
}

func dataSourceCloudRoutingRouterVRFsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	routerID := d.Get("router_id").(int)
	resp, err := m.(*ProviderClients).V3.ListRouterVRFs(routerID)
	if err != nil {
		return diag.FromErr(err)
	}
	keys := make([]string, 0, len(resp))
	for key := range resp {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		vrf := resp[key]
		out[i] = map[string]interface{}{"vrf_id": vrf.VrfID, "name": vrf.Name, "description": vrf.Description}
	}
	var diags diag.Diagnostics
	setValue("vrfs", out, d, &diags)
	d.SetId(fmt.Sprintf("cloud-router-%d-vrfs", routerID))
	return diags
}
