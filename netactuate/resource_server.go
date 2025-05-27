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

// resourceServerUpdate handles in-place rebuilds, POP moves, and hostname
// renames. If you change POP, it will:
//   1) delete any existing BGP sessions
//   2) delete the server (cancel=false) so we can unlink
//   3) unlink the server from its old IP
//   4) call BuildServer(...) with the new location/image
//   5) wait for RUNNING
//   6) recreate BGP sessions (if any)
//
// Hostname-only changes still go via DeleteServer(id,false) → wait → Build.
func resourceServerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    c := m.(*gona.Client)
    var diags diag.Diagnostics

    // Only proceed with rebuild if specific fields have changed
    if d.HasChange("location") || d.HasChange("location_id") || d.HasChange("image") ||
        d.HasChange("image_id") || d.HasChange("hostname") || d.HasChange("params") {

        // Parse Terraform resource ID
        id, err := strconv.Atoi(d.Id())
        if err != nil {
            diags = append(diags, diag.FromErr(err)...)
            return diags
        }

        // Resolve new target IDs
        newLocID, newImgID, getParamsDiags := getParams(d, c)
        if getParamsDiags != nil {
            return getParamsDiags
        }

        // Determine if unlinking is required (for location or location_id changes)
        unlinkRequired := false
        if d.HasChange("location") {
            oldLocRaw, _ := d.GetChange("location")
            if oldLocRaw.(string) != "" {
                unlinkRequired = true
            }
            // Clear location_id so getParams() will re-resolve it from the new string
            d.Set("location_id", 0)
        }
        if d.HasChange("location_id") {
            oldIDRaw, _ := d.GetChange("location_id")
            if oldIDRaw.(int) != 0 {
                unlinkRequired = true
            }
        }

        // Debug: Log preconditions for BGP and unlinking
        diags = append(diags, diag.Diagnostic{
            Severity: diag.Warning,
            Summary:  "Update preconditions",
            Detail:   fmt.Sprintf("unlinkRequired=%t, HasChange(location)=%t, HasChange(location_id)=%t, HasChange(hostname)=%t",
                unlinkRequired, d.HasChange("location"), d.HasChange("location_id"), d.HasChange("hostname")),
        })

        // If we’re moving, tear down BGP first
        var hadSessions, hasIPv6, isRedundant bool
        var groupID int
        if unlinkRequired {
            sessions, err := c.GetBGPSessions(id)
            if err != nil {
                diags = append(diags, diag.FromErr(err)...)
                return diags
            }
            hadSessions = len(sessions) > 0
            if hadSessions {
                groupID = sessions[0].GroupID
                counts := make(map[string]int)
                for _, s := range sessions {
                    counts[s.ProviderIPType]++
                }
                hasIPv6 = counts[string(gona.IPv6)] > 0
                for _, cnt := range counts {
                    if cnt > 1 {
                        isRedundant = true
                        break
                    }
                }
                for _, s := range sessions {
                    if err := c.DeleteBGPSession(s.ID); err != nil {
                        diags = append(diags, diag.Diagnostic{
                            Severity: diag.Error,
                            Summary:  "Failed to delete BGP session",
                            Detail:   fmt.Sprintf("Failed to delete BGP session %d: %s", s.ID, err),
                        })
                        return diags
                    }
                }
                // Wait for BGP sessions to clear
                deadline := time.Now().Add(2 * time.Minute)
                for {
                    if time.Now().After(deadline) {
                        diags = append(diags, diag.Diagnostic{
                            Severity: diag.Error,
                            Summary:  "Timed out waiting for BGP sessions to clear",
                            Detail:   fmt.Sprintf("Timed out waiting for BGP sessions to clear on server %d", id),
                        })
                        return diags
                    }
                    rem, err := c.GetBGPSessions(id)
                    if err != nil {
                        diags = append(diags, diag.FromErr(err)...)
                        return diags
                    }
                    if len(rem) == 0 {
                        break
                    }
                    time.Sleep(5 * time.Second)
                }
            }

            // Delete the server (cancel=false) so we can unlink
            jobID, err := c.DeleteServer(id, false)
            if err != nil {
                diags = append(diags, diag.Diagnostic{
                    Severity: diag.Error,
                    Summary:  "Failed to delete server before move",
                    Detail:   fmt.Sprintf("Failed to delete server %d before move: %s", id, err),
                })
                return diags
            }
            if err := waitForJob(c, id, jobID); err != nil {
                diags = append(diags, diag.Diagnostic{
                    Severity: diag.Error,
                    Summary:  "Failed waiting for delete job in move",
                    Detail:   fmt.Sprintf("Waiting for delete job in move: %s", err),
                })
                return diags
            }

            // Unlink the server
            if err := c.UnlinkServer(id); err != nil {
                diags = append(diags, diag.Diagnostic{
                    Severity: diag.Error,
                    Summary:  "Failed to unlink server",
                    Detail:   fmt.Sprintf("Failed to unlink server %d: %s", id, err),
                })
                return diags
            }
        }

        // If hostname changed, delete & wait
        if d.HasChange("hostname") {
            oldHostRaw, _ := d.GetChange("hostname")
            if oldHostRaw.(string) != "" {
                jobID, err := c.DeleteServer(id, false)
                if err != nil {
                    diags = append(diags, diag.FromErr(err)...)
                    return diags
                }
                if err := waitForJob(c, id, jobID); err != nil {
                    diags = append(diags, diag.Diagnostic{
                        Severity: diag.Error,
                        Summary:  "Failed waiting for delete job in rename",
                        Detail:   fmt.Sprintf("Waiting for delete job in rename: %s", err),
                    })
                    return diags
                }
            }
        }

        // Rebuild (either same POP or new POP)
        diags = append(diags, diag.Diagnostic{
            Severity: diag.Warning,
            Summary:  "Rebuilding server",
            Detail:   fmt.Sprintf("Rebuilding server with ID=%d, Location=%d, Image=%d", id, newLocID, newImgID),
        })
        sshKeyIDRaw := fmt.Sprint(d.Get("ssh_key_id"))
        sshKeyID, err := strconv.Atoi(sshKeyIDRaw)
        if err != nil {
            diags = append(diags, diag.Diagnostic{
                Severity: diag.Error,
                Summary:  "Invalid SSH key ID",
                Detail:   fmt.Sprintf("Invalid ssh_key_id %q", sshKeyIDRaw),
            })
            return diags
        }
        billingContract := fmt.Sprint(d.Get("package_billing_contract_id"))

        req := &gona.BuildServerRequest{
            Plan:                     d.Get("plan").(string),
            Location:                 newLocID,
            Image:                    newImgID,
            FQDN:                     d.Get("hostname").(string),
            SSHKey:                   d.Get("ssh_key").(string),
            SSHKeyID:                 sshKeyID,
            Password:                 d.Get("password").(string),
            PackageBilling:           d.Get("package_billing").(string),
            PackageBillingContractId: billingContract,
            CloudConfig:              base64.StdEncoding.EncodeToString([]byte(d.Get("cloud_config").(string))),
            ScriptContent:            base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string))),
            Params:                   d.Get("params").(string),
        }
        if _, err := c.BuildServer(id, req); err != nil {
            diags = append(diags, diag.FromErr(err)...)
            return diags
        }
        // Fix for wait4Status returning diag.Diagnostics
        if _, waitDiags := wait4Status(id, "RUNNING", c); len(waitDiags) > 0 {
            diags = append(diags, waitDiags...)
            return diags
        }

        // Recreate BGP sessions if—and only if—we unlinked AND had sessions
        if unlinkRequired && hadSessions {
            diags = append(diags, diag.Diagnostic{
                Severity: diag.Warning,
                Summary:  "Attempting to recreate BGP sessions",
                Detail:   fmt.Sprintf("ServerID=%d, GroupID=%d, HasIPv6=%t, IsRedundant=%t", id, groupID, hasIPv6, isRedundant),
            })

            // Call CreateBGPSessions, expecting a single *gona.BGPSession
            newSession, err := c.CreateBGPSessions(id, groupID, hasIPv6, isRedundant)
            if err != nil {
                diags = append(diags, diag.Diagnostic{
                    Severity: diag.Error,
                    Summary:  "Failed to recreate BGP sessions",
                    Detail:   fmt.Sprintf("Error recreating BGP sessions for server %d: %s", id, err),
                })
                return diags
            }

            // Debug successful creation
            sessionCount := 0
            if newSession != nil {
                sessionCount = 1
            }
            diags = append(diags, diag.Diagnostic{
                Severity: diag.Warning,
                Summary:  "BGP sessions recreated",
                Detail:   fmt.Sprintf("Created %d BGP session(s) for server %d: %+v", sessionCount, id, newSession),
            })
        }
    }

    // Finally, re-sync all fields
    return resourceServerRead(ctx, d, m)
}


func resourceServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*gona.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// 1) Kick off the delete, get back a job ID
	jobID, err := client.DeleteServer(id, true)
	if err != nil {
		return diag.Errorf("failed to delete server %d: %s", id, err)
	}

	// 2) Poll that job until status==5 (success) or timeout
	if err := waitForJob(client, id, jobID); err != nil {
		return diag.Errorf("error waiting for delete job %d: %s", jobID, err)
	}

	// 3) Mark Terraform resource as gone
	d.SetId("")
	return nil
}

// waitForJob polls GetJob until job.Status == 5 or we exhaust jobTries.
// Reuses the same pattern and constants you have for wait4Status.
func waitForJob(client *gona.Client, serverID, jobID int) error {
	for i := 0; i < tries; i++ {
		job, err := client.GetJob(serverID, jobID)
		if err != nil {
			return fmt.Errorf("polling job %d: %w", jobID, err)
		}
		if job.Status == 5 {
			return nil
		}
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
	return fmt.Errorf("timed out waiting for job %d after %d attempts", jobID, tries)
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

// getParams resolves “location” and “image” into their numeric IDs.
// It first checks for a non-zero location_id or image_id, then
// falls back to looking up by name if needed.
func getParams(d *schema.ResourceData, client *gona.Client) (int, int, diag.Diagnostics) {
    var diags diag.Diagnostics

    // 1) Location
    var locationId int
    if v, ok := d.GetOk("location_id"); ok && v.(int) != 0 {
        locationId = v.(int)
    } else if name := d.Get("location").(string); name != "" {
        id, ld := getLocation(d, client)
        if ld != nil {
            diags = append(diags, *ld)
        }
        locationId = id
    } else {
        // explicit zero if nothing set
        locationId = d.Get("location_id").(int)
    }

    // 2) Image
    var imageId int
    if v, ok := d.GetOk("image_id"); ok && v.(int) != 0 {
        imageId = v.(int)
    } else if name := d.Get("image").(string); name != "" {
        os, idDiag := getImageByName(name, client)
        if idDiag != nil {
            diags = append(diags, *idDiag)
        } else {
            imageId = os.ID
        }
    } else {
        imageId = d.Get("image_id").(int)
    }

    return locationId, imageId, diags
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
