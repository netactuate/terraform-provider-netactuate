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

func resourceVPCBackendTemplate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCBackendTemplateCreate,
		ReadContext:   resourceVPCBackendTemplateRead,
		UpdateContext: resourceVPCBackendTemplateUpdate,
		DeleteContext: resourceVPCBackendTemplateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCBackendTemplateImport,
		},
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the parent VPC",
			},
			"backend_template_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The backend template ID assigned by the API",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the backend template",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the backend template",
			},
			"backend_host": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Backend hosts in this template",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"backend_host_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The backend host ID assigned by the API",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the backend host",
						},
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "IP address of the backend host",
						},
					},
				},
			},
		},
	}
}

func resourceVPCBackendTemplateCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)

	req := &gona.CreateVPCBackendTemplateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	if v, ok := d.GetOk("backend_host"); ok {
		hosts := v.([]interface{})
		for _, h := range hosts {
			host := h.(map[string]interface{})
			req.BackendHosts = append(req.BackendHosts, gona.VPCBackend{
				Name:    host["name"].(string),
				Address: host["address"].(string),
			})
		}
	}

	result, err := c.CreateVPCBackendTemplate(vpcID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, result.BackendTemplateID))
	log.Printf("[DEBUG] Backend template %d created in VPC %d", result.BackendTemplateID, vpcID)

	return resourceVPCBackendTemplateRead(ctx, d, m)
}

func resourceVPCBackendTemplateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, templateID, err := parseBackendTemplateID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	tmpl, err := c.GetVPCBackendTemplate(vpcID, templateID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Backend template %d in VPC %d not found, removing from state", templateID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("backend_template_id", tmpl.BackendTemplateID, d, &diags)
	setValue("name", tmpl.Name, d, &diags)
	setValue("description", tmpl.Description, d, &diags)

	hosts := make([]map[string]interface{}, len(tmpl.BackendHosts))
	for i, h := range tmpl.BackendHosts {
		addr := h.InternalAddress
		if addr == "" {
			addr = h.Address
		}
		hosts[i] = map[string]interface{}{
			"backend_host_id": h.BackendHostID,
			"name":            h.Name,
			"address":         addr,
		}
	}
	setValue("backend_host", hosts, d, &diags)

	return diags
}

func resourceVPCBackendTemplateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, templateID, err := parseBackendTemplateID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.ReplaceVPCBackendTemplateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	if v, ok := d.GetOk("backend_host"); ok {
		hosts := v.([]interface{})
		for _, h := range hosts {
			host := h.(map[string]interface{})
			req.BackendHosts = append(req.BackendHosts, gona.VPCBackend{
				Name:    host["name"].(string),
				Address: host["address"].(string),
			})
		}
	}

	if _, err := c.ReplaceVPCBackendTemplate(vpcID, templateID, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceVPCBackendTemplateRead(ctx, d, m)
}

func resourceVPCBackendTemplateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, templateID, err := parseBackendTemplateID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting backend template %d from VPC %d", templateID, vpcID)

	if err := c.DeleteVPCBackendTemplate(vpcID, templateID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] Backend template %d in VPC %d already deleted", templateID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceVPCBackendTemplateImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	vpcID, templateID, err := parseBackendTemplateID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"vpcId/backendTemplateId\": %w", d.Id(), err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, templateID))
	d.Set("vpc_id", vpcID)

	return []*schema.ResourceData{d}, nil
}

func parseBackendTemplateID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid backend template ID %q, expected \"vpcId/backendTemplateId\"", id)
	}
	vpcID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vpc_id %q: %w", parts[0], err)
	}
	templateID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid backend_template_id %q: %w", parts[1], err)
	}
	return vpcID, templateID, nil
}
