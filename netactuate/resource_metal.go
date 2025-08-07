package netactuate

import (
	"context"
	//"encoding/base64"
	//"fmt"
	"regexp"
	//"strconv"
	//"strings"
	//"time"
    "log"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
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

func resourceMetalCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	//c := m.(*gona.Client)

//     var diags diag.Diagnostics
// 	if diags != nil {
// 		return diags
// 	}
// 	diags = diag.Diagnostics{}


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


	//s, err := c.CreateMetal(req)
	//if err != nil {
	//	return diag.FromErr(err)
	//}

	//d.SetId(strconv.Itoa(s.MetalID))
	//d.Set("params", req.Params) // Store params in the state file

	//if _, err := wait4Status(s.MetalID, "RUNNING", c); err != nil {
	//	return err
	//}

	//metal, err := c.GetMetal(s.MetalID)
	//if err != nil {
	//	return diag.FromErr(err)
	//}
	//setValue("primary_ipv4", metal.PrimaryIPv4, d, &diags)
	//setValue("primary_ipv6", metal.PrimaryIPv6, d, &diags)
	log.Printf("[DEBUG] CreateMetalRequest: %+v", req)

	return nil
}

func resourceMetalRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    log.Println("[DEBUG] Read called")

    return nil
}

func resourceMetalUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    log.Println("[DEBUG] Dummy Update called")
    return nil
}

func resourceMetalDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    log.Println("[DEBUG] Dummy Delete called")
    d.SetId("")
    return nil
}
