package netactuate

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	tries       = 100
	intervalSec = 5
)

var (
	credentialKeys = []string{"password", "ssh_key_id", "ssh_key"}
	locationKeys   = []string{"location", "location_id"}
	imageKeys      = []string{"image", "image_id"}
	billingKeys    = []string{"package_billing_contract_id", "package_billing_opt_in"}

	hostnameRegex = fmt.Sprintf("(%[1]s\\.)*%[1]s$", fmt.Sprintf("(%[1]s|%[1]s%[2]s*%[1]s)", "[a-zA-Z0-9]", "[a-zA-Z0-9\\-]"))
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerCreate,
		ReadContext:   resourceServerRead,
		UpdateContext: resourceServerUpdate,
		DeleteContext: resourceServerDelete,
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
			"plan": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"package_billing": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
				Default:  "usage",
			},
			"package_billing_opt_in": {
				Type:         schema.TypeString,
				ExactlyOneOf: billingKeys,
				ForceNew:     false,
				Optional:     true,
			},
			"package_billing_contract_id": {
				Type:         schema.TypeString,
				ExactlyOneOf: billingKeys,
				ForceNew:     false,
				Optional:     true,
			},
			"location": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: locationKeys,
				StateFunc: func(val any) string {
					return strings.ToUpper(val.(string))
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" || strings.EqualFold(strings.ToUpper(old), strings.Fields(new)[0]) {
						return true
					}
					return false
				},
			},
			"location_id": {
				Type:         schema.TypeInt,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: locationKeys,
				Computed:     true,
			},
			"image": {
				Type:         schema.TypeString,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: imageKeys,
			},
			"image_id": {
				Type:         schema.TypeInt,
				ForceNew:     false,
				Optional:     true,
				ExactlyOneOf: imageKeys,
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
			"cloud_config": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"user_data_base64": {
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
			},
			"primary_ipv4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_ipv6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"params": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Additional JSON formatted parameters to be passed to the server creation and management API",
			},
		},
		CustomizeDiff: customdiff.Sequence(
			customdiff.ComputedIf("primary_ipv4", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location_id") || d.HasChange("image") || d.HasChange("image_id") || d.HasChange("hostname")
			}),
			customdiff.ComputedIf("primary_ipv6", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("location_id") || d.HasChange("image") || d.HasChange("image_id") || d.HasChange("hostname")
			}),
		),
	}
}

func resourceServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	locationId, imageId, diags := getParams(d, c)
	if diags != nil {
		return diags
	}
	diags = diag.Diagnostics{}

	req := &gona.CreateServerRequest{
		Plan:                     d.Get("plan").(string),
		Location:                 locationId,
		Image:                    imageId,
		FQDN:                     d.Get("hostname").(string),
		SSHKey:                   d.Get("ssh_key").(string),
		SSHKeyID:                 d.Get("ssh_key_id").(int),
		Password:                 d.Get("password").(string),
		PackageBilling:           d.Get("package_billing").(string),
		PackageBillingContractId: d.Get("package_billing_contract_id").(string),
		CloudConfig:              base64.StdEncoding.EncodeToString([]byte(d.Get("cloud_config").(string))),
		ScriptContent:            base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string))),
		Params:                   d.Get("params").(string), // Handle the new params field

	}

	if userData64, ok := d.GetOk("user_data_base64"); ok {
		req.ScriptContent = userData64.(string)
	}

	var packageValue = d.Get("package_billing")
	if packageValue == "package" {
		optIn, ok := d.GetOk("package_billing_opt_in")
		if !ok {
			return diag.Errorf("when package_billing is set to package, package_billing_opt_in must be set to yes")
		}

		if optIn.(string) != "yes" {
			return diag.Errorf("when package_billing is set to package, package_billing_opt_in must be set to yes")
		}
	}

	if packageValue == "usage" {
		contractID, ok := d.GetOk("package_billing_contract_id")
		if !ok {
			return diag.Errorf("package_billing_contract_id must be set to your contract ID with NetActuate")
		}

		if len(contractID.(string)) == 0 {
			return diag.Errorf("package_billing_contract_id must be set to your contract ID with NetActuate")
		}
	}

	s, err := c.CreateServer(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(s.ServerID))
	d.Set("params", req.Params) // Store params in the state file

	if _, err := wait4Status(s.ServerID, "RUNNING", c); err != nil {
		return err
	}

	server, err := c.GetServer(s.ServerID)
	if err != nil {
		return diag.FromErr(err)
	}
	setValue("primary_ipv4", server.PrimaryIPv4, d, &diags)
	setValue("primary_ipv6", server.PrimaryIPv6, d, &diags)

	return nil
}

func resourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := c.GetServer(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	if server.Installed == 0 {
		setValue("hostname", "", d, &diags)
		updateValue("image_id", 0, d, &diags)
		updateValue("image", "", d, &diags)
	} else {
		setValue("hostname", server.Name, d, &diags)
		updateValue("image_id", server.OSID, d, &diags)
		updateValue("image", server.OS, d, &diags)
	}

	setValue("plan", server.Package, d, &diags)
	updateValue("location_id", server.LocationID, d, &diags)

	// Safely derive the first field of server.Location, if present
	locParts := strings.Fields(server.Location)
	if len(locParts) > 0 {
		updateValue("location", locParts[0], d, &diags)
	}

	_, existsLocID := d.GetOk("location_id")
	_, existsLoc := d.GetOk("location")
	if !existsLocID && !existsLoc && len(locParts) > 0 {
		setValue("location", locParts[0], d, &diags)
	}

	// Similarly for image
	_, existsImgID := d.GetOk("image_id")
	_, existsImg := d.GetOk("image")
	if !existsImgID && !existsImg {
		setValue("image", server.OS, d, &diags)
	}

	setValue("primary_ipv4", server.PrimaryIPv4, d, &diags)
	setValue("primary_ipv6", server.PrimaryIPv6, d, &diags)

	return diags
}
func resourceServerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*gona.Client)
	// Rebuild on these property changes
	if d.HasChange("location") || d.HasChange("location_id") || d.HasChange("image") || d.HasChange("image_id") || d.HasChange("hostname") || d.HasChange("params") {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		oldHostRaw, _ := d.GetChange("hostname")
		oldHost := oldHostRaw.(string)
		if oldHost != "" {
			jobID, err := c.DeleteServer(id, false)
			if err != nil {
				return diag.FromErr(err)
			}
			if err := c.WaitForJob(id, jobID, "delete"); err != nil {
				return diag.Errorf("waiting for delete job in update: %s", err)
			}
		}

		// Existing rebuild logic follows...
		locationId, imageId, diags := getParams(d, c)
		if diags != nil {
			return diags
		}
		req := &gona.BuildServerRequest{
			Plan:                     d.Get("plan").(string),
			Location:                 locationId,
			Image:                    imageId,
			FQDN:                     d.Get("hostname").(string),
			SSHKey:                   d.Get("ssh_key").(string),
			SSHKeyID:                 d.Get("ssh_key_id").(int),
			Password:                 d.Get("password").(string),
			PackageBilling:           d.Get("package_billing").(string),
			PackageBillingContractId: d.Get("package_billing_contract_id").(string),
			CloudConfig:              base64.StdEncoding.EncodeToString([]byte(d.Get("cloud_config").(string))),
			ScriptContent:            base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string))),
			Params:                   d.Get("params").(string),
		}

		if _, err := c.BuildServer(id, req); err != nil {
			return diag.FromErr(err)
		}

		if d.HasChange("params") {
			d.Set("params", req.Params)
		}

		// Wait until the server is running (can replace with WaitForJob for build if desired)
		if _, err := wait4Status(id, "RUNNING", c); err != nil {
			return err
		}
	}

	return resourceServerRead(ctx, d, m)
}

func resourceServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    c := m.(*gona.Client)
    id, err := strconv.Atoi(d.Id())
    if err != nil {
        return diag.FromErr(err)
    }

    // 1) Kick off delete, get back a Job ID or an API error
    jobID, err := c.DeleteServer(id, true)
    if err != nil {
        // directly surface the API's error message and fields
        return diag.Errorf("failed to delete server %d: %s", id, err)
    }

    // 2) Poll until that job completes (status==5)
    if err := c.WaitForJob(id, jobID, "delete"); err != nil {
        return diag.Errorf("error waiting for delete job %d: %s", jobID, err)
    }

    // 3) Tell Terraform it's gone
    d.SetId("")
    return nil
}

func wait4Status(serverId int, status string, client *gona.Client) (server gona.Server, d diag.Diagnostics) {
	for i := 0; i < tries; i++ {
		server, err := client.GetServer(serverId)

		// Special-case deletion: when waiting for TERMINATED, treat either a real
		// TERMINATED or a blank status (due to the 422/invalid-mbpkgid) as success.
		// if status == "TERMINATED" && err == nil && (server.ServerStatus == status || server.ServerStatus == "") {
		// 	return server, nil
		// }

		if err != nil && i >= 5 {
			// Retry errors on first few attempts, since sometimes calling GetServer
			// immediately after creating a server returns an error
			// ("mbpkgid must be a valid mbpkgid").
			return server, diag.FromErr(err)
		}

		if err == nil && server.ServerStatus == status {
			return server, nil
		}

		time.Sleep(intervalSec * time.Second)
	}

	return server, diag.Errorf("Timeout of waiting the server to obtain %q status", status)
}

func getParams(d *schema.ResourceData, client *gona.Client) (int, int, diag.Diagnostics) {
	var diags diag.Diagnostics
	locationId, ld := getLocation(d, client)
	if ld != nil {
		diags = append(diags, *ld)
	}
	imageId, exists := d.GetOk("image_id")
	if !exists {
		image, d := getImageByName(d.Get("image").(string), client)
		if d != nil {
			diags = append(diags, *d)
		} else {
			imageId = image.ID
		}
	}

	return locationId, imageId.(int), diags
}

func getLocation(d *schema.ResourceData, client *gona.Client) (int, *diag.Diagnostic) {
	locationId, exists := d.GetOk("location_id")
	if exists {
		return locationId.(int), nil
	}

	requestLocation := d.Get("location").(string)
	if requestLocation == "" {
		return 0, &diag.Errorf("Please provide a location or location_id")[0]
	}

	locations, err := client.GetLocations()
	if err != nil {
		return 0, &diag.FromErr(err)[0]
	}

	for _, location := range locations {
		if location.Name == requestLocation {
			return location.ID, nil
		}
		if strings.EqualFold(strings.Fields(location.Name)[0], strings.Fields(requestLocation)[0]) {
			return location.ID, nil
		}
	}

	return 0, &diag.Errorf("Provided location %q doesn't exist", locationId)[0]
}

func getImageByName(name string, client *gona.Client) (*gona.OS, *diag.Diagnostic) {
	oss, err := client.GetOSs()
	if err != nil {
		return nil, &diag.FromErr(err)[0]
	}

	for _, os := range oss {
		if os.Os == name {
			return &os, nil
		}
	}

	return nil, &diag.Errorf("Provided image %q doesn't exist", name)[0]
}
