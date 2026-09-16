package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirewallSetVMs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFirewallSetVMsRead,
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Firewall set ID whose VM attachments will be listed",
			},
			"vms": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of VMs attached to the firewall set",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall set VM attachment ID",
						},
						"mbpkgid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "VM package ID",
						},
						"interface_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Network interface ID",
						},
						"firewall_set_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall set ID",
						},
						"set_priority": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Firewall set priority for this VM",
						},
						"created": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM attachment creation timestamp",
						},
						"last_updated": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM attachment last update timestamp",
						},
						"iata_code": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IATA code of the VM location",
						},
						"location": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM location name",
						},
						"hostname": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM hostname",
						},
					},
				},
			},
		},
	}
}

func dataSourceFirewallSetVMsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	setID := d.Get("firewall_set_id").(int)

	vms, err := c.GetFirewallSetVMs(setID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	vmsList := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		vmsList[i] = map[string]interface{}{
			"id":              vm.ID,
			"mbpkgid":         vm.Mbpkgid,
			"interface_id":    vm.InterfaceID,
			"firewall_set_id": vm.FirewallSetID,
			"set_priority":    vm.SetPriority,
			"created":         vm.Created,
			"last_updated":    vm.LastUpdated,
			"iata_code":       vm.IATACode,
			"location":        vm.Location,
			"hostname":        vm.Hostname,
		}
	}

	setValue("vms", vmsList, d, &diags)
	d.SetId(fmt.Sprintf("firewall-set-%d-vms", setID))

	return diags
}
