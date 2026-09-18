package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceBGPGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBGPGroupCreate,
		ReadContext:   resourceBGPGroupRead,
		DeleteContext: resourceBGPGroupDelete,
		Description:   "Creates a BGP group. The API has no delete endpoint for BGP groups, so deleting this resource removes it from Terraform state only and leaves the group on the account.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "BGP group name.",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "BGP group description.",
			},
			"group_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "BGP group type. The API defaults this to anycast when omitted.",
			},
			"bgp_group_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "API assigned BGP group ID.",
			},
		},
	}
}

func resourceBGPGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	req := &gona.CreateBGPGroupRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}
	if v, ok := d.GetOk("group_type"); ok {
		req.GroupType = v.(string)
	}

	group, err := c.CreateBGPGroup(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(group.ID))
	return resourceBGPGroupRead(ctx, d, m)
}

func resourceBGPGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	group, err := c.GetBGPGroup(id)
	if err != nil {
		if gona.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setBGPGroupState(group, d, &diags)
	return diags
}

func resourceBGPGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("deleting a BGP group is not supported: the NetActuate API has no delete endpoint for BGP group %s. Release it through support, then remove it from state with `terraform state rm`.", d.Id())
}

func setBGPGroupState(group *gona.BGPGroup, d *schema.ResourceData, diags *diag.Diagnostics) {
	setValue("bgp_group_id", group.ID, d, diags)
	setValue("name", group.Name, d, diags)
	setValue("description", group.Description, d, diags)
	setValue("group_type", group.GroupType, d, diags)
}
