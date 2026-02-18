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

func resourceVPCSSHKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVPCSSHKeyCreate,
		ReadContext:   resourceVPCSSHKeyRead,
		UpdateContext: resourceVPCSSHKeyUpdate,
		DeleteContext: resourceVPCSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCSSHKeyImport,
		},
		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VPC",
			},
			"ssh_key_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The account-level SSH key ID",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the SSH key is enabled for this VPC (default: true)",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the SSH key",
			},
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fingerprint of the SSH key",
			},
			"public_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The public key",
			},
		},
	}
}

func resourceVPCSSHKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID := d.Get("vpc_id").(int)
	sshKeyID := d.Get("ssh_key_id").(int)
	enabled := d.Get("enabled").(bool)

	if err := c.EnableVPCSSHKey(vpcID, sshKeyID, enabled); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, sshKeyID))
	log.Printf("[DEBUG] SSH key %d set enabled=%v for VPC %d", sshKeyID, enabled, vpcID)

	return resourceVPCSSHKeyRead(ctx, d, m)
}

func resourceVPCSSHKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, sshKeyID, err := parseVPCSSHKeyID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	key, err := c.GetVPCSSHKey(vpcID, sshKeyID)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SSH key %d in VPC %d not found, removing from state", sshKeyID, vpcID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("vpc_id", vpcID, d, &diags)
	setValue("ssh_key_id", key.GetID(), d, &diags)
	setValue("name", key.Name, d, &diags)
	setValue("fingerprint", key.Fingerprint, d, &diags)
	setValue("public_key", key.PublicKey, d, &diags)
	setValue("enabled", key.IsEnabled(), d, &diags)

	return diags
}

func resourceVPCSSHKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, sshKeyID, err := parseVPCSSHKeyID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("enabled") {
		enabled := d.Get("enabled").(bool)
		if err := c.EnableVPCSSHKey(vpcID, sshKeyID, enabled); err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[DEBUG] SSH key %d set enabled=%v for VPC %d", sshKeyID, enabled, vpcID)
	}

	return resourceVPCSSHKeyRead(ctx, d, m)
}

func resourceVPCSSHKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	vpcID, sshKeyID, err := parseVPCSSHKeyID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting SSH key %d from VPC %d", sshKeyID, vpcID)

	if err := c.DeleteVPCSSHKey(vpcID, sshKeyID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SSH key %d in VPC %d already removed", sshKeyID, vpcID)
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceVPCSSHKeyImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	vpcID, sshKeyID, err := parseVPCSSHKeyID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q, expected \"vpcId/sshKeyId\": %w", d.Id(), err)
	}

	d.SetId(fmt.Sprintf("%d/%d", vpcID, sshKeyID))
	d.Set("vpc_id", vpcID)
	d.Set("ssh_key_id", sshKeyID)

	return []*schema.ResourceData{d}, nil
}

func parseVPCSSHKeyID(id string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid VPC SSH key ID %q, expected \"vpcId/sshKeyId\"", id)
	}
	vpcID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vpc_id %q: %w", parts[0], err)
	}
	sshKeyID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid ssh_key_id %q: %w", parts[1], err)
	}
	return vpcID, sshKeyID, nil
}
