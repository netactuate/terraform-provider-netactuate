package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceSshKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSshKeyRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceSshKeyRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	c := m.(*gona.Client)

	sshKey, err := c.GetSSHKey(d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("name", sshKey.Name, d, &diags)
	setValue("key", sshKey.Key, d, &diags)
	setValue("fingerprint", sshKey.Fingerprint, d, &diags)

	if diags == nil {
		d.SetId(strconv.Itoa(sshKey.ID))
	}

	return diags
}
