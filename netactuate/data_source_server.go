package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceServer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"plan_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"package": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"location_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"image": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"ip_v4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_v6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	server, err := c.GetServer(d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("hostname", server.Name, d, &diags)
	setValue("package", server.Package, d, &diags)
	setValue("plan_id", server.PlanID, d, &diags)
	setValue("location_id", server.LocationID, d, &diags)
	setValue("image", server.OS, d, &diags)
	setValue("image_id", server.OSID, d, &diags)
	setValue("ip_v4", server.PrimaryIPv4, d, &diags)
	setValue("ip_v6", server.PrimaryIPv6, d, &diags)
	setValue("status", server.ServerStatus, d, &diags)
	setValue("state", server.PowerStatus, d, &diags)

	if diags == nil {
		d.SetId(strconv.Itoa(server.ID))
	}

	return diags
}
