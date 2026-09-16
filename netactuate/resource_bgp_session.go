package netactuate

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceBGPSessions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBGPSessionCreate,
		ReadContext:   resourceBGPSessionRead,
		DeleteContext: resourceBGPSessionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"group_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"ipv6": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Default:  true,
				Optional: true,
			},
			"redundant": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func resourceBGPSessionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	_, err := c.CreateBGPSessions(d.Get("mbpkgid").(int), d.Get("group_id").(int), d.Get("ipv6").(bool),
		d.Get("redundant").(bool))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(d.Get("mbpkgid").(int)))

	return nil
}

func resourceBGPSessionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbPkgID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sessions, err := c.GetBGPSessions(mbPkgID)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(sessions) == 0 {
		// Sessions cleared out of band (e.g. in the portal) -- drop from
		// state so Terraform offers to recreate rather than treating this
		// as an unmanaged no-op forever.
		d.SetId("")
		return nil
	}

	// Recover imported fields to avoid spurious changes to ForceNew attributes.
	// All three are now reliably recoverable from the session list itself:
	// - group_id: every session carries its own group_id directly.
	// - ipv6: true iff any returned session is on the IPv6 address family.
	// - redundant: this platform creates TWO sessions per enabled address
	//   family when redundant is requested (vs one otherwise), so a
	//   per-family session count > 1 means redundant was in effect.
	var diags diag.Diagnostics
	setValue("mbpkgid", mbPkgID, d, &diags)
	setValue("group_id", sessions[0].GroupID, d, &diags)
	v4Count, v6Count := 0, 0
	for _, s := range sessions {
		if s.IsProviderIPTypeV4() {
			v4Count++
		} else {
			v6Count++
		}
	}
	setValue("ipv6", v6Count > 0, d, &diags)
	setValue("redundant", v4Count > 1 || v6Count > 1, d, &diags)
	return diags
}

func resourceBGPSessionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	mbPkgID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sessions, err := c.GetBGPSessions(mbPkgID)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, s := range sessions {
		if err := c.DeleteBGPSession(s.ID); err != nil {
			return diag.Errorf("failed to delete BGP session %d: %s", s.ID, err)
		}
	}

	// BGP teardown isn't instant -- poll until the sessions actually clear
	// so a ForceNew recreate (group_id, ipv6, or redundant changing) doesn't
	// collide with a session still winding down.
	const (
		bgpDeleteTimeout  = 2 * time.Minute
		bgpDeletePollWait = 5 * time.Second
	)
	deadline := time.Now().Add(bgpDeleteTimeout)
	for {
		remaining, err := c.GetBGPSessions(mbPkgID)
		if err != nil {
			return diag.FromErr(err)
		}
		if len(remaining) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return diag.Errorf("timed out waiting for BGP sessions on %d to clear", mbPkgID)
		}
		time.Sleep(bgpDeletePollWait)
	}
}
