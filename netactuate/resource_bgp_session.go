package netactuate

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
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
	c := m.(*gona.Client)

	_, err := c.CreateBGPSessions(d.Get("mbpkgid").(int), d.Get("group_id").(int), d.Get("ipv6").(bool),
		d.Get("redundant").(bool))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(d.Get("mbpkgid").(int)))

	return nil
}

func resourceBGPSessionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Do nothing
	return nil
}

func resourceBGPSessionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    c       := m.(*gona.Client)
    mbPkgID := d.Get("mbpkgid").(int)

    // 1) List all sessions
    sessions, err := c.GetBGPSessions(mbPkgID)
    if err != nil {
        return diag.FromErr(err)
    }

    // 2) Delete each session
    for _, s := range sessions {
        if err := c.DeleteBGPSession(s.ID); err != nil {
            return diag.Errorf("failed to delete BGP session %d: %s", s.ID, err)
        }
    }

    // 3) Poll until no sessions remain
    deadline := time.Now().Add(2 * time.Minute)
    for {
        if time.Now().After(deadline) {
            return diag.Errorf("timed out waiting for BGP sessions on %d to clear", mbPkgID)
        }
        remaining, err := c.GetBGPSessions(mbPkgID)
        if err != nil {
            return diag.FromErr(err)
        }
        if len(remaining) == 0 {
            break
        }
        time.Sleep(5 * time.Second)
    }

    // 4) Done
    d.SetId("")
    return nil
}
