package netactuate

import (
	"context"
	//"encoding/base64"
	//"fmt"
	"regexp"
	"strconv"
	//"strings"
	"time"
    "log"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	metalIntervalSec = 5
)

func resourceMetal() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMetalCreate,
		ReadContext:   resourceMetalRead,
		UpdateContext: resourceMetalUpdate,
		DeleteContext: resourceMetalDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				ForceNew: false,
				Required: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					var diags diag.Diagnostics

					match, err := regexp.MatchString(hostnameRegex, i.(string))
					if err != nil {
						diags = diag.FromErr(err)
					} else if !match {
						diags = diag.Errorf("%q is not a valid hostname", i)
					}

					return diags
				},
			},
			"location": {
				Type:         schema.TypeInt,
				ForceNew:     false,
				Optional:     true,
				Computed:     true,
			},
            "device_id": {
                Type:     schema.TypeInt,
                Required: true,
                Description: "Device ID",
            },
            "profile": {
                Type:     schema.TypeInt,
                Required: true,
                Description: "Profile ID",
            },
            "build_script": {
                Type:     schema.TypeString,
                Optional: true,
                Description: "Build script content",
            },
			"password": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Sensitive:    true,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
			},
			"ssh_key_id": {
				Type:         schema.TypeInt,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
			},
			"ssh_key": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: credentialKeys,
			},
            "disklayout": {
                Type:     schema.TypeInt,
                Optional: true,
                Description: "Disk layout ID",
            },
            "primary_ipv4": {
                Type:     schema.TypeString,
                Computed: true,
            },
            "primary_ipv6": {
                Type:     schema.TypeString,
                Computed: true,
            },
		},
		CustomizeDiff: customdiff.Sequence(
			customdiff.ComputedIf("primary_ipv4", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location") || d.HasChange("hostname")
			}),
			customdiff.ComputedIf("primary_ipv6", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location")  || d.HasChange("hostname")
			}),
		),
	}
}

func wait4BuildStatus(buildID int, timeoutMinutes int, client *gona.Client) diag.Diagnostics {
    if timeoutMinutes == 0 {
        timeoutMinutes = 45
    }

    start := time.Now()

    for {
        job, err := client.GetMetalBuildStatus(buildID)
        if err != nil {
            return diag.FromErr(err)
        }

        if job.Status == "Failed" || job.Status == "Archived" {
            return diag.Errorf("Build #%d failed with status: %s", buildID, job.Status)
        }

        if job.Status == "Complete" {
            return nil
        }

        elapsed := time.Since(start)
        if elapsed.Minutes() >= float64(timeoutMinutes) {
            return diag.Errorf("timeout waiting for job #%d to complete (waited %v minutes)", buildID, int(elapsed.Minutes()))
        }

        time.Sleep(metalIntervalSec * time.Second)
    }
}


func resourceMetalCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    c := m.(*gona.Client)

    var diags diag.Diagnostics
	if diags != nil {
		return diags
	}
	diags = diag.Diagnostics{}


	req := &gona.CreateMetalRequest{
		Location: d.Get("location").(int),
		Device:                   d.Get("device_id").(int),
		SSHKey:                   d.Get("ssh_key").(string),
		SSHKeyID:                 d.Get("ssh_key_id").(int),
		Password:                 d.Get("password").(string),
		BuildScript:              d.Get("build_script").(string),
		DiskLayout:               d.Get("disklayout").(int),
		Profile:                  d.Get("profile").(int),
        Hostname:                 d.Get("hostname").(string),
	}


	s, err := c.CreateMetal(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(s.MBPKGID))

    if d := wait4BuildStatus(s.Build, 45, c); d != nil {
        return d
    }

	metal, err := c.GetMetal(s.MBPKGID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] METAL: %+v", metal)

	setValue("primary_ipv4", metal.PrimaryIP, d, &diags)
	setValue("primary_ipv6", metal.PrimaryIPv6, d, &diags)
	log.Printf("[DEBUG] CreateMetalRequest: %+v", req)

	return nil
}

func resourceMetalRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    client := m.(*gona.Client)

    id, err := strconv.Atoi(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    metal, err := client.GetMetal(id)
    if err != nil {
        return diag.FromErr(err)
    }

    var diags diag.Diagnostics

    if metal.NPSInstalled == 0 {
        d.Set("hostname", "")
        d.Set("primary_ipv4", "")
        d.Set("primary_ipv6", "")
    } else {
        d.Set("hostname", metal.Hostname)
        d.Set("primary_ipv4", metal.PrimaryIP)
        if metal.PrimaryIPv6 != nil {
            d.Set("primary_ipv6", *metal.PrimaryIPv6)
        } else {
            d.Set("primary_ipv6", "")
        }
    }
    d.Set("location", metal.DatacenterID)
    d.Set("device_id", metal.ID)

    return diags
}

func anyChange(d *schema.ResourceData, fields ...string) bool {
	for _, f := range fields {
		if d.HasChange(f) {
			return true
		}
	}
	return false
}

func resourceMetalUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	if !anyChange(d, "profile", "build_script", "hostname", "disklayout") {
		log.Println("[DEBUG] No relevant changes, skipping update")
		return nil
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Updating metal resource ID: %d\n", id)
    req := &gona.BuildMetalRequest {
        MBPKGID:                  id,
        SSHKey:                   d.Get("ssh_key").(string),
        SSHKeyID:                 d.Get("ssh_key_id").(int),
        Password:                 d.Get("password").(string),
        BuildScript:              d.Get("build_script").(string),
        DiskLayout:               d.Get("disklayout").(int),
        Profile:                  d.Get("profile").(int),
        Hostname:                 d.Get("hostname").(string),
    }

	s, err := c.BuildMetal(id, req)
	if err != nil {
		return diag.FromErr(err)
	}

    if d := wait4BuildStatus(s.Build, 45, c); d != nil {
        return d
    }

	return resourceMetalRead(ctx, d, m)
}

func resourceMetalDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

    id, err := strconv.Atoi(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

	log.Printf("[DEBUG] Deleting metal with ID: %d", id)
	agreeValue := "true"
    commentsValue := "Delete from terraform"
    req := &gona.CancelRequest{
        MBPKGID:    id,
        CancelType: "Immediate",
        Agree:      agreeValue,
        Comments:   &commentsValue,
    }


    if _, err := c.CancelPackage(req); err != nil {
        return diag.FromErr(err)
    }

    d.SetId("")
    return nil
}
