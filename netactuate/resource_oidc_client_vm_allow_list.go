package netactuate

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceOIDCClientVMAllowList() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOIDCClientVMAllowListCreate,
		ReadContext:   resourceOIDCClientVMAllowListRead,
		DeleteContext: resourceOIDCClientVMAllowListDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oidc_client_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "OIDC client ID",
			},
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "VM package ID to allow",
			},
		},
	}
}

func resourceOIDCClientVMAllowListCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID := d.Get("oidc_client_id").(int)
	mbpkgid := d.Get("mbpkgid").(int)
	if err := c.AddOIDCClientVMs(clientID, []int{mbpkgid}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", clientID, mbpkgid))
	return resourceOIDCClientVMAllowListRead(ctx, d, m)
}

func resourceOIDCClientVMAllowListRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, mbpkgid, err := parseTwoPartID(d.Id(), "clientId", "mbpkgid")
	if err != nil {
		return diag.FromErr(err)
	}

	vms, err := c.GetOIDCClientVMs(clientID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] OIDC client %d not found for VM allow list entry %d, removing from state", clientID, mbpkgid)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	for _, vm := range vms {
		if vm.MBPkgID == mbpkgid {
			var diags diag.Diagnostics
			setValue("oidc_client_id", clientID, d, &diags)
			setValue("mbpkgid", mbpkgid, d, &diags)
			return diags
		}
	}

	d.SetId("")
	return nil
}

func resourceOIDCClientVMAllowListDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clientID, mbpkgid, err := parseTwoPartID(d.Id(), "clientId", "mbpkgid")
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.RemoveOIDCClientVM(clientID, mbpkgid); err != nil {
		if gona.IsV3NotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
