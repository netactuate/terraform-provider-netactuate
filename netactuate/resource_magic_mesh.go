package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceMagicMesh() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMagicMeshCreate,
		ReadContext:   resourceMagicMeshRead,
		UpdateContext: resourceMagicMeshUpdate,
		DeleteContext: resourceMagicMeshDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"mesh_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the magic mesh.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name for the magic mesh.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description for the magic mesh.",
			},
		},
	}
}

func resourceMagicMeshCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	req := &gona.CreateMagicMeshRequest{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		desc := v.(string)
		req.Description = &desc
	}

	resp, err := c.CreateMagicMesh(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(resp.MeshID))

	var diags diag.Diagnostics
	setValue("mesh_id", resp.MeshID, d, &diags)

	readDiags := resourceMagicMeshRead(ctx, d, m)
	diags = append(diags, readDiags...)
	return diags
}

func resourceMagicMeshRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	mesh, err := c.GetMagicMesh(meshID)
	if err != nil {
		d.SetId("")
		return nil
	}

	var diags diag.Diagnostics

	setValue("mesh_id", mesh.MeshID, d, &diags)
	setValue("name", mesh.Name, d, &diags)
	if mesh.Description != nil {
		setValue("description", *mesh.Description, d, &diags)
	}

	return diags
}

func resourceMagicMeshUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateMagicMeshRequest{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		req.Name = &name
	}

	if d.HasChange("description") {
		desc := d.Get("description").(string)
		req.Description = &desc
	}

	err = c.UpdateMagicMesh(meshID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceMagicMeshRead(ctx, d, m)
}

func resourceMagicMeshDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	meshID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteMagicMesh(meshID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
