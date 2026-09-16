package netactuate

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func resourceNKEDNSAddon() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKEDNSAddonCreate,
		ReadContext:   resourceNKEDNSAddonRead,
		UpdateContext: resourceNKEDNSAddonUpdate,
		DeleteContext: resourceNKEDNSAddonDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNKEDNSAddonImport,
		},
		CustomizeDiff: customdiff.All(
			func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
				return validateNKEAddonPlan(ctx, d, m, nkeDNSAddonType)
			},
		),
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The NKE cluster ID",
			},
			"zone": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DNS zone to attach to the cluster",
			},
			"mode": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"full-zone", "subzone"}, false)),
				Description:      "DNS addon mode",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Addon version",
			},
			"channel": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Addon channel",
			},
			"dns_zone_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "NKE DNS zone ID",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Addon state",
			},
			"display_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Addon display name",
			},
			"update_available": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether an addon update is available",
			},
			"installed_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Addon installation timestamp",
			},
		},
	}
}

func resourceNKEDNSAddonCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	req := &gona.CreateNKEAddonRequest{
		AddonType: nkeDNSAddonType,
		Config: gona.NKEDNSAddonWriteConfig{
			Zone: d.Get("zone").(string),
			Mode: d.Get("mode").(string),
		},
	}
	if v, ok := d.GetOk("version"); ok {
		req.Version = v.(string)
	}
	if v, ok := d.GetOk("channel"); ok {
		req.Channel = v.(string)
	}

	if _, err := c.CreateClusterAddon(clusterID, req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(nkeAddonStateID(clusterID, nkeDNSAddonType))
	return resourceNKEDNSAddonRead(ctx, d, m)
}

func resourceNKEDNSAddonRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clusterID, addonType, err := parseNKEAddonStateID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	addon, err := c.GetClusterAddon(clusterID, addonType)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NKE cluster %d addon %s not found, removing from state", clusterID, addonType)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("cluster_id", clusterID, d, &diags)
	setValue("version", addon.Version, d, &diags)
	setValue("channel", addon.Channel, d, &diags)
	setValue("state", addon.State, d, &diags)
	setValue("display_name", addon.DisplayName, d, &diags)
	setValue("update_available", addon.UpdateAvailable, d, &diags)
	setValue("installed_on", addon.Timestamps.InstalledOn, d, &diags)

	desiredZone := d.Get("zone").(string)
	var matched *gona.NKEDNSAddonZone
	for i := range addon.Config.Zones {
		if addon.Config.Zones[i].Zone == desiredZone || desiredZone == "" {
			matched = &addon.Config.Zones[i]
			break
		}
	}
	if matched == nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "NKE DNS addon zone not found in read response",
			Detail:   "The addon read response did not include zone " + strconv.Quote(desiredZone) + " in config.zones.",
		})
	}

	setValue("dns_zone_id", matched.DNSZoneID, d, &diags)
	setValue("zone", matched.Zone, d, &diags)
	setValue("mode", matched.Mode, d, &diags)
	return diags
}

func resourceNKEDNSAddonUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	req := &gona.UpdateNKEAddonRequest{
		Config: gona.NKEDNSAddonWriteConfig{
			Zone: d.Get("zone").(string),
			Mode: d.Get("mode").(string),
		},
	}
	if v, ok := d.GetOk("version"); ok {
		req.Version = v.(string)
	}
	if v, ok := d.GetOk("channel"); ok {
		req.Channel = v.(string)
	}

	if _, err := c.UpdateClusterAddon(clusterID, nkeDNSAddonType, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceNKEDNSAddonRead(ctx, d, m)
}

func resourceNKEDNSAddonDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	if err := c.DeleteClusterAddon(clusterID, nkeDNSAddonType); err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceNKEDNSAddonImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	clusterID, addonType, err := parseNKEAddonStateID(d.Id())
	if err != nil || addonType != nkeDNSAddonType {
		return nil, fmt.Errorf("invalid import ID %q, expected \"cluster_id/%s\"", d.Id(), nkeDNSAddonType)
	}

	d.SetId(nkeAddonStateID(clusterID, addonType))
	d.Set("cluster_id", clusterID)

	return []*schema.ResourceData{d}, nil
}
