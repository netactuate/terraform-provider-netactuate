package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSshKeys() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSshKeysRead,
		Schema: map[string]*schema.Schema{
			"sshkeys": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of SSH keys installed for the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "SSH key ID",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SSH key name",
						},
						"ssh_key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SSH public key body",
						},
						"fingerprint": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "SSH key fingerprint",
						},
					},
				},
			},
		},
	}
}

func dataSourceSshKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	sshKeys, err := c.GetSSHKeys()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	sshKeysList := make([]map[string]interface{}, len(sshKeys))
	for i, sshKey := range sshKeys {
		sshKeysList[i] = map[string]interface{}{
			"id":          sshKey.ID,
			"name":        sshKey.Name,
			"ssh_key":     sshKey.Key,
			"fingerprint": sshKey.Fingerprint,
		}
	}

	setValue("sshkeys", sshKeysList, d, &diags)
	d.SetId("sshkeys")

	return diags
}
