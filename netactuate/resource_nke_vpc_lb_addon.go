package netactuate

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// nkeVPCLBAddonRetryInterval is a var so tests can shorten it.
var nkeVPCLBAddonRetryInterval = 10 * time.Second

// The add-on's operator gives a floating IP to each Service with loadBalancerClass
// "netactuate.com/vpc-lb" and each Ingress with ingressClassName "netactuate".
func resourceNKEVPCLBAddon() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNKEVPCLBAddonCreate,
		ReadContext:   resourceNKEVPCLBAddonRead,
		DeleteContext: resourceNKEVPCLBAddonDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNKEVPCLBAddonImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The NKE cluster ID",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Addon version",
			},
			"channel": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Addon channel",
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

func resourceNKEVPCLBAddonCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	req := &gona.CreateNKEAddonRequest{
		AddonType: nkeVPCLBAddonType,
	}

	// Replacement can race asynchronous teardown; retry only "still being removed" responses.
	for {
		if err := ctx.Err(); err != nil {
			return diag.Errorf("NKE cluster %d VPC LB addon: not enabled by the create timeout: %s", clusterID, err)
		}
		_, err := c.CreateClusterAddon(clusterID, req)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "still being removed") {
			return diag.FromErr(err)
		}
		log.Printf("[DEBUG] NKE cluster %d VPC LB addon: previous instance still being removed, retrying", clusterID)
		select {
		case <-ctx.Done():
			return diag.Errorf("NKE cluster %d VPC LB addon: the previous instance was still being removed at the create timeout: %s", clusterID, err)
		case <-time.After(nkeVPCLBAddonRetryInterval):
		}
	}

	d.SetId(nkeAddonStateID(clusterID, nkeVPCLBAddonType))
	return resourceNKEVPCLBAddonRead(ctx, d, m)
}

func resourceNKEVPCLBAddonRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	return diags
}

func resourceNKEVPCLBAddonDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	if err := c.DeleteClusterAddon(clusterID, nkeVPCLBAddonType); err != nil {
		if gona.IsV3NotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func resourceNKEVPCLBAddonImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	clusterID, addonType, err := parseNKEAddonStateID(d.Id())
	if err != nil || addonType != nkeVPCLBAddonType {
		return nil, fmt.Errorf("invalid import ID %q, expected \"cluster_id/%s\"", d.Id(), nkeVPCLBAddonType)
	}

	d.SetId(nkeAddonStateID(clusterID, addonType))
	d.Set("cluster_id", clusterID)

	return []*schema.ResourceData{d}, nil
}
