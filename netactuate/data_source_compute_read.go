package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceBootProfiles() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBootProfilesRead,
		Schema: map[string]*schema.Schema{
			"boot_profiles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of cloud boot profiles. The API field named id is exposed as boot_profile_id because Terraform reserves id for resource state.",
				Elem: &schema.Resource{
					Schema: bootProfileSchema(),
				},
			},
		},
	}
}

func dataSourceServerDisks() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerDisksRead,
		Schema: map[string]*schema.Schema{
			"mbpkgid": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Server package ID whose cloud disks should be read.",
			},
			"disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of cloud disks for the server. The live endpoint returned only empty lists when this data source was added, so no per-disk fields are exposed yet.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{},
				},
			},
		},
	}
}

func dataSourceDedicatedOSProfiles() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedOSProfilesRead,
		Schema: map[string]*schema.Schema{
			"os_profiles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of dedicated OS profiles. The API field named id is exposed as os_id because Terraform reserves id for resource state.",
				Elem: &schema.Resource{
					Schema: dedicatedOSSchema(),
				},
			},
		},
	}
}

func dataSourceDedicatedRescueOS() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedRescueOSRead,
		Schema: map[string]*schema.Schema{
			"os_profiles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of dedicated rescue OS profiles. The API field named id is exposed as os_id because Terraform reserves id for resource state.",
				Elem: &schema.Resource{
					Schema: dedicatedOSSchema(),
				},
			},
		},
	}
}

func dataSourceDedicatedDiskLayouts() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDedicatedDiskLayoutsRead,
		Schema: map[string]*schema.Schema{
			"os_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Dedicated OS profile ID to read disk layouts for.",
			},
			"disk_layouts": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of dedicated disk layouts. The API field named id is exposed as layout_id because Terraform reserves id for resource state.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"layout_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Dedicated disk layout ID from the API field named id.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Dedicated disk layout name.",
						},
						"profile": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Dedicated disk layout profile.",
						},
						"min_disks": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Minimum number of disks required for the layout.",
						},
					},
				},
			},
		},
	}
}

func dataSourceBootProfilesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	// /cloud/kernels is an alias that returns the same payload as /cloud/boot-profiles.
	// Keep one data source so users do not get the same rows under two names.
	profiles, err := c.GetBootProfiles()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("boot_profiles", flattenBootProfiles(profiles), d, &diags)
	d.SetId("boot-profiles")
	return diags
}

func dataSourceServerDisksRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	mbpkgid := d.Get("mbpkgid").(int)

	disks, err := c.GetServerDisks(mbpkgid)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	// The endpoint can return empty lists, so preserve only the list shape
	// until the API returns fields to model.
	setValue("disks", flattenServerDisks(disks), d, &diags)
	d.SetId(fmt.Sprintf("server-disks-%d", mbpkgid))
	return diags
}

func dataSourceDedicatedOSProfilesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	profiles, err := c.GetDedicatedOSProfiles()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("os_profiles", flattenDedicatedOSProfiles(profiles), d, &diags)
	d.SetId("dedicated-os-profiles")
	return diags
}

func dataSourceDedicatedRescueOSRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	profiles, err := c.GetDedicatedRescueOS()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("os_profiles", flattenDedicatedRescueOS(profiles), d, &diags)
	d.SetId("dedicated-rescue-os")
	return diags
}

func dataSourceDedicatedDiskLayoutsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2
	osID := d.Get("os_id").(int)

	layouts, err := c.GetDedicatedDiskLayouts(osID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("disk_layouts", flattenDedicatedDiskLayouts(layouts), d, &diags)
	d.SetId(fmt.Sprintf("dedicated-disk-layouts-%d", osID))
	return diags
}

func bootProfileSchema() map[string]*schema.Schema {
	stringFields := map[string]string{
		"name":           "Boot profile name.",
		"type":           "Boot profile type.",
		"description":    "Boot profile description.",
		"builder":        "Boot profile builder.",
		"kernel":         "Boot profile kernel value.",
		"boot":           "Boot profile boot value.",
		"serial":         "Boot profile serial value.",
		"disk_represent": "Boot profile disk representation value.",
		"last_updated":   "Boot profile last updated timestamp.",
		"extra":          "Boot profile extra value, empty when the API returns null.",
		"vncdisplay":     "Boot profile VNC display value, empty when the API returns null.",
		"disk_root":      "Boot profile disk root value, empty when the API returns null.",
		"bootloader":     "Boot profile bootloader value, empty when the API returns null.",
		"ramdisk":        "Boot profile ramdisk value, empty when the API returns null.",
		"initrd":         "Boot profile initrd value, empty when the API returns null.",
		"created":        "Boot profile created timestamp, empty when the API returns null.",
	}
	intFields := map[string]string{
		"boot_profile_id": "Boot profile ID from the API field named id.",
		"image_template":  "Boot profile image template id, an int as the API returns it.",
		"pae":             "PAE flag as returned by the API, using 1 or 0.",
		"acpi":            "ACPI flag as returned by the API, using 1 or 0.",
		"apic":            "APIC flag as returned by the API, using 1 or 0.",
		"xlocaltime":      "Local time flag as returned by the API, using 1 or 0.",
		"sdl":             "SDL flag as returned by the API, using 1 or 0.",
		"vnc":             "VNC flag as returned by the API, using 1 or 0.",
		"vncconsole":      "VNC console flag as returned by the API, using 1 or 0.",
		"vncunused":       "VNC unused flag as returned by the API, using 1 or 0.",
		"hide":            "Hide flag as returned by the API, using 1 or 0.",
		"kvm":             "KVM flag as returned by the API, using 1 or 0.",
	}

	result := make(map[string]*schema.Schema, len(stringFields)+len(intFields))
	for name, description := range stringFields {
		result[name] = &schema.Schema{Type: schema.TypeString, Computed: true, Description: description}
	}
	for name, description := range intFields {
		result[name] = &schema.Schema{Type: schema.TypeInt, Computed: true, Description: description}
	}
	return result
}

func dedicatedOSSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"os_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Dedicated OS profile ID from the API field named id.",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Dedicated OS profile name.",
		},
		"group_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Dedicated OS profile group name.",
		},
		"tags": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Tags associated with the dedicated OS profile.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"disklayouts": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Disk layouts normalized to a list with layout_id and name across OS and rescue OS endpoints.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"layout_id": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Disk layout ID from the API object key or id field.",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Disk layout name.",
					},
				},
			},
		},
		"scripts": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Scripts normalized to a list with script_id and name across OS and rescue OS endpoints.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"script_id": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Script ID from the API object key or id field.",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Script name.",
					},
				},
			},
		},
		"default_disklayout": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Default disk layout ID, or 0 when the API returns null.",
		},
		"default_scripts": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Default script IDs.",
			Elem:        &schema.Schema{Type: schema.TypeInt},
		},
		"allow_ssh_keys": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "SSH key allowance flag as returned by the API, using 1 or 0.",
		},
		"set_root_password": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Root password flag as returned by the API, using 1 or 0.",
		},
		"rescue_image": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Rescue image flag as returned by the API, using 1 or 0.",
		},
		"public": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Public visibility flag as returned by the API, using 1 or 0.",
		},
		"enabled": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Enabled flag as returned by the API, using 1 or 0.",
		},
		"created": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Creation timestamp.",
		},
		"last_updated": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Last updated timestamp.",
		},
		"profile_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Dedicated profile ID returned by the API field named profile_id.",
		},
		"arch": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "OS architecture.",
		},
		"flavor": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "OS flavor.",
		},
		"location_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Location ID, or 0 when the API returns null.",
		},
	}
}

func flattenBootProfiles(profiles []gona.BootProfile) []map[string]interface{} {
	result := make([]map[string]interface{}, len(profiles))
	for i, profile := range profiles {
		result[i] = map[string]interface{}{
			"boot_profile_id": profile.BootProfileID,
			"name":            profile.Name,
			"type":            profile.Type,
			"description":     profile.Description,
			"builder":         profile.Builder,
			"kernel":          profile.Kernel,
			"boot":            profile.Boot,
			"serial":          profile.Serial,
			"disk_represent":  profile.DiskRepresent,
			"image_template":  profile.ImageTemplate,
			"last_updated":    profile.LastUpdated,
			"extra":           stringPtrOrEmpty(profile.Extra),
			"vncdisplay":      stringPtrOrEmpty(profile.VNCDisplay),
			"disk_root":       stringPtrOrEmpty(profile.DiskRoot),
			"bootloader":      stringPtrOrEmpty(profile.Bootloader),
			"ramdisk":         stringPtrOrEmpty(profile.Ramdisk),
			"initrd":          stringPtrOrEmpty(profile.Initrd),
			"created":         stringPtrOrEmpty(profile.Created),
			"pae":             profile.PAE,
			"acpi":            profile.ACPI,
			"apic":            profile.APIC,
			"xlocaltime":      profile.XLocaltime,
			"sdl":             profile.SDL,
			"vnc":             profile.VNC,
			"vncconsole":      profile.VNCConsole,
			"vncunused":       profile.VNCUnused,
			"hide":            profile.Hide,
			"kvm":             profile.KVM,
		}
	}
	return result
}

func flattenServerDisks(disks []gona.ServerDisk) []map[string]interface{} {
	result := make([]map[string]interface{}, len(disks))
	for i := range disks {
		result[i] = map[string]interface{}{}
	}
	return result
}

func flattenDedicatedOSProfiles(profiles []gona.DedicatedOSProfile) []map[string]interface{} {
	result := make([]map[string]interface{}, len(profiles))
	for i, profile := range profiles {
		result[i] = flattenDedicatedOSProfile(profile.OSID, profile.Name, profile.GroupName, profile.Tags, profile.DiskLayouts, profile.Scripts, profile.DefaultDiskLayout, profile.DefaultScripts, profile.AllowSSHKeys, profile.SetRootPassword, profile.RescueImage, profile.Public, profile.Enabled, profile.Created, profile.LastUpdated, profile.ProfileID, profile.Arch, profile.Flavor, profile.LocationID)
	}
	return result
}

func flattenDedicatedRescueOS(profiles []gona.DedicatedRescueOS) []map[string]interface{} {
	result := make([]map[string]interface{}, len(profiles))
	for i, profile := range profiles {
		result[i] = flattenDedicatedOSProfile(profile.OSID, profile.Name, profile.GroupName, profile.Tags, profile.DiskLayouts, profile.Scripts, profile.DefaultDiskLayout, profile.DefaultScripts, profile.AllowSSHKeys, profile.SetRootPassword, profile.RescueImage, profile.Public, profile.Enabled, profile.Created, profile.LastUpdated, profile.ProfileID, profile.Arch, profile.Flavor, profile.LocationID)
	}
	return result
}

func flattenDedicatedOSProfile(osID int, name string, groupName string, tags []string, diskLayouts []gona.DedicatedIDName, scripts []gona.DedicatedIDName, defaultDiskLayout int, defaultScripts []int, allowSSHKeys int, setRootPassword int, rescueImage int, public int, enabled int, created string, lastUpdated string, profileID int, arch string, flavor string, locationID *int) map[string]interface{} {
	return map[string]interface{}{
		"os_id":              osID,
		"name":               name,
		"group_name":         groupName,
		"tags":               tags,
		"disklayouts":        flattenIDNames(diskLayouts, "layout_id"),
		"scripts":            flattenIDNames(scripts, "script_id"),
		"default_disklayout": defaultDiskLayout,
		"default_scripts":    defaultScripts,
		"allow_ssh_keys":     allowSSHKeys,
		"set_root_password":  setRootPassword,
		"rescue_image":       rescueImage,
		"public":             public,
		"enabled":            enabled,
		"created":            created,
		"last_updated":       lastUpdated,
		"profile_id":         profileID,
		"arch":               arch,
		"flavor":             flavor,
		"location_id":        intPtrOrZero(locationID),
	}
}

func flattenDedicatedDiskLayouts(layouts []gona.DedicatedDiskLayout) []map[string]interface{} {
	result := make([]map[string]interface{}, len(layouts))
	for i, layout := range layouts {
		result[i] = map[string]interface{}{
			"layout_id": layout.LayoutID,
			"name":      layout.Name,
			"profile":   layout.Profile,
			"min_disks": layout.MinDisks,
		}
	}
	return result
}

func flattenIDNames(items []gona.DedicatedIDName, idKey string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		result[i] = map[string]interface{}{
			idKey:  item.ID,
			"name": item.Name,
		}
	}
	return result
}

func stringPtrOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intPtrOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
