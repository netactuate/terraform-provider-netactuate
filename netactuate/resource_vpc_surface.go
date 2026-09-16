package netactuate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceVPCNameservers() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCNameserversPut,
		ReadContext:   resourceVPCNameserversRead,
		UpdateContext: resourceVPCNameserversPut,
		DeleteContext: resourceVPCNameserversDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"vpc_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "VPC ID whose DHCP nameservers are managed."},
			"ipv4":   {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "IPv4 nameservers announced by DHCP."},
			"ipv6":   {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "IPv6 nameservers announced by DHCP."},
		},
	}
}

func resourceVPCNameserversPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID := d.Get("vpc_id").(int)
	req := &gona.ReplaceVPCNameserversRequest{
		Nameservers: append(stringsToNameservers(d.Get("ipv4").([]interface{})), stringsToNameservers(d.Get("ipv6").([]interface{}))...),
	}
	if _, err := m.(*ProviderClients).V3.ReplaceVPCNameservers(vpcID, req); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(vpcID))
	return resourceVPCNameserversRead(ctx, d, m)
}

func resourceVPCNameserversRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	ns, err := m.(*ProviderClients).V3.GetVPCNameservers(vpcID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("vpc_id", vpcID, d, &diags)
	setValue("ipv4", flattenNameservers(ns.IPv4), d, &diags)
	setValue("ipv6", flattenNameservers(ns.IPv6), d, &diags)
	return diags
}

func resourceVPCNameserversDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := m.(*ProviderClients).V3.ReplaceVPCNameservers(vpcID, &gona.ReplaceVPCNameserversRequest{Nameservers: []gona.VPCNameserver{}}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceVPCStandbyGateway() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCStandbyGatewayCreate,
		ReadContext:   resourceVPCStandbyGatewayRead,
		DeleteContext: resourceVPCStandbyGatewayDelete,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Schema: map[string]*schema.Schema{
			"vpc_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "VPC ID that should have a redundant standby gateway."},
		},
	}
}

func resourceVPCStandbyGatewayCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID := d.Get("vpc_id").(int)
	if err := m.(*ProviderClients).V3.AddVPCStandbyGateway(vpcID); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(vpcID))
	return resourceVPCStandbyGatewayRead(ctx, d, m)
}

func resourceVPCStandbyGatewayRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := m.(*ProviderClients).V3.GetVPC(vpcID); err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	setValue("vpc_id", vpcID, d, &diags)
	return diags
}

func resourceVPCStandbyGatewayDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := m.(*ProviderClients).V3.DeleteVPCStandbyGateway(vpcID); err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceVPCBackendTemplateBackends() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCBackendTemplateBackendsPut,
		ReadContext:   resourceVPCBackendTemplateBackendsRead,
		UpdateContext: resourceVPCBackendTemplateBackendsPut,
		DeleteContext: resourceVPCBackendTemplateBackendsDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceVPCBackendTemplateBackendsImport},
		Schema: map[string]*schema.Schema{
			"vpc_id":              {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "VPC ID that owns the backend template."},
			"backend_template_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Backend template ID whose backends are managed."},
			"backend_host": {Type: schema.TypeList, Optional: true, Description: "Complete replacement set of backend hosts for the template.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"backend_host_id": {Type: schema.TypeInt, Computed: true, Description: "Backend host ID assigned by the API."},
				"name":            {Type: schema.TypeString, Required: true, Description: "Backend host name."},
				"address":         {Type: schema.TypeString, Required: true, Description: "Backend host address."},
			}}},
		},
	}
}

func resourceVPCBackendTemplateBackendsPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID := d.Get("vpc_id").(int)
	templateID := d.Get("backend_template_id").(int)
	req := &gona.ReplaceVPCBackendsRequest{BackendHosts: []gona.VPCBackend{}}
	if v, ok := d.GetOk("backend_host"); ok {
		for _, raw := range v.([]interface{}) {
			host := raw.(map[string]interface{})
			req.BackendHosts = append(req.BackendHosts, gona.VPCBackend{Name: host["name"].(string), Address: host["address"].(string)})
		}
	}
	if _, err := m.(*ProviderClients).V3.ReplaceVPCBackends(vpcID, templateID, req); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d/%d", vpcID, templateID))
	return resourceVPCBackendTemplateBackendsRead(ctx, d, m)
}

func resourceVPCBackendTemplateBackendsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, templateID, err := parseSurfaceTwoPartID(d.Id(), "vpcId/backendTemplateId")
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid import ID %q, expected vpcId/backendTemplateId: %w", d.Id(), err))
	}
	backends, err := m.(*ProviderClients).V3.ListVPCBackends(vpcID, templateID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	out := make([]map[string]interface{}, len(backends))
	for i, backend := range backends {
		address := backend.InternalAddress
		if address == "" {
			address = backend.Address
		}
		out[i] = map[string]interface{}{"backend_host_id": backend.BackendHostID, "name": backend.Name, "address": address}
	}
	var diags diag.Diagnostics
	setValue("vpc_id", vpcID, d, &diags)
	setValue("backend_template_id", templateID, d, &diags)
	setValue("backend_host", out, d, &diags)
	return diags
}

func resourceVPCBackendTemplateBackendsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcID, templateID, err := parseSurfaceTwoPartID(d.Id(), "vpcId/backendTemplateId")
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := m.(*ProviderClients).V3.ReplaceVPCBackends(vpcID, templateID, &gona.ReplaceVPCBackendsRequest{BackendHosts: []gona.VPCBackend{}}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceVPCBackendTemplateBackendsImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	vpcID, templateID, err := parseSurfaceTwoPartID(d.Id(), "vpcId/backendTemplateId")
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected vpcId/backendTemplateId: %w", d.Id(), err)
	}
	d.Set("vpc_id", vpcID)
	d.Set("backend_template_id", templateID)
	return []*schema.ResourceData{d}, nil
}
