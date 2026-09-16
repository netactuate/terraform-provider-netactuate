package netactuate

import (
	"context"
	"github.com/netactuate/gona/gona"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirewallSet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallSetCreate,
		ReadContext:   resourceFirewallSetRead,
		UpdateContext: resourceFirewallSetUpdate,
		DeleteContext: resourceFirewallSetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the firewall set",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the firewall set",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the firewall set rules are active",
			},
		},
	}
}

func resourceFirewallSetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	enabled := d.Get("enabled").(bool)

	set, err := c.CreateFirewallSet(name, description, enabled)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(set.ID))

	return resourceFirewallSetRead(ctx, d, m)
}

func resourceFirewallSetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	set, err := c.GetFirewallSet(id)
	if err != nil {
		if gona.IsNotFound(err) {
			// The set was deleted out of band. vAPI2 reports that as a 422 on the
			// firewall_set_id field rather than a 404, which the SDK now maps to a
			// NotFoundError. Without this the resource hard errors on every refresh and
			// the only way out is terraform state rm on a security resource.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("name", set.Name, d, &diags)
	setValue("description", set.Description, d, &diags)
	setValue("enabled", set.Enabled, d, &diags)

	return diags
}

func resourceFirewallSetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	enabled := d.Get("enabled").(bool)

	if _, err := c.UpdateFirewallSet(id, name, description, enabled); err != nil {
		return diag.FromErr(err)
	}

	// Sync to VMs when enabled/disabled changes
	if d.HasChange("enabled") {
		if err := c.SyncFirewallSetRules(id); err != nil {
			return diag.Errorf("sync failed after update: %s", err)
		}
	}

	return resourceFirewallSetRead(ctx, d, m)
}

func resourceFirewallSetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteFirewallSet(id); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
