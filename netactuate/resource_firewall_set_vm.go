package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirewallSetVM() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallSetVMCreate,
		ReadContext:   resourceFirewallSetVMRead,
		DeleteContext: resourceFirewallSetVMDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"firewall_set_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the firewall set to attach the VM to",
			},
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The VM package ID to attach",
			},
			"interface_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     0,
				Description: "The network interface ID (default 0)",
			},
			"set_priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     0,
				Description: "Priority of the firewall set for this VM (default 0)",
			},
			"hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hostname of the attached VM",
			},
			"location": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Location name of the VM",
			},
			"iata_code": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IATA code of the VM location",
			},
		},
	}
}

func resourceFirewallSetVMCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID := d.Get("firewall_set_id").(int)
	mbpkgid := d.Get("mbpkgid").(int)
	interfaceID := d.Get("interface_id").(int)
	setPriority := d.Get("set_priority").(int)

	_, err := c.AttachFirewallSetVM(setID, mbpkgid, interfaceID, setPriority)
	if err != nil {
		return diag.Errorf("failed to attach VM %d to firewall set %d: %s", mbpkgid, setID, err)
	}

	d.SetId(fmt.Sprintf("%d/%d", setID, mbpkgid))

	return resourceFirewallSetVMRead(ctx, d, m)
}

func resourceFirewallSetVMRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID, mbpkgid, err := parseFirewallSetVMID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	vms, err := c.GetFirewallSetVMs(setID)
	if err != nil {
		return diag.Errorf("failed to list VMs for firewall set %d: %s", setID, err)
	}

	var found bool
	for _, vm := range vms {
		if vm.Mbpkgid == mbpkgid {
			found = true
			var diags diag.Diagnostics
			setValue("firewall_set_id", setID, d, &diags)
			setValue("mbpkgid", mbpkgid, d, &diags)
			setValue("interface_id", vm.InterfaceID, d, &diags)
			setValue("set_priority", vm.SetPriority, d, &diags)
			setValue("hostname", vm.Hostname, d, &diags)
			setValue("location", vm.Location, d, &diags)
			setValue("iata_code", vm.IATACode, d, &diags)
			return diags
		}
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceFirewallSetVMDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	setID, mbpkgid, err := parseFirewallSetVMID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DetachFirewallSetVM(setID, mbpkgid); err != nil {
		return diag.Errorf("failed to detach VM %d from firewall set %d: %s", mbpkgid, setID, err)
	}

	return nil
}

func parseFirewallSetVMID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid firewall set VM ID format %q, expected {setId}/{mbpkgid}", id)
	}
	setID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid set ID %q: %s", parts[0], err)
	}
	mbpkgid, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid mbpkgid %q: %s", parts[1], err)
	}
	return setID, mbpkgid, nil
}
