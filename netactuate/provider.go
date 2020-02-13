package netactuate

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Provider something
func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"NETACTUATE_TOKEN",
					"NETACTUATE_ACCESS_TOKEN",
				}, nil),
				Description: "The token key for API operations.",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETACTUATE_API_URL", "https://api.netactuate.com"),
				Description: "The URL to use for the NetActuate API.",
			},
			"spaces_access_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPACES_ACCESS_KEY_ID", nil),
				Description: "The access key ID for Spaces API operations.",
			},
			"spaces_secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SPACES_SECRET_ACCESS_KEY", nil),
				Description: "The secret access key for Spaces API operations.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"netactuate_account":             dataSourceNetActuateAccount(),
			"netactuate_package":             dataSourceNetActuatePackage(),
			"netactuate_image":               dataSourceNetActuateImage(),
			"netactuate_record":              dataSourceNetActuateRecord(),
			"netactuate_sizes":               dataSourceNetActuateSizes(),
			"netactuate_ssh_key":             dataSourceNetActuateSSHKey(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"netactuate_package":                  resourceNetActuatePackage(),
			"netactuate_record":                   resourceNetActuateRecord(),
			"netactuate_ssh_key":                  resourceNetActuateSSHKey(),

		},
	}

	p.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(d, terraformVersion)
	}

	return p
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	config := Config{
		Token:            d.Get("token").(string),
		APIEndpoint:      d.Get("api_endpoint").(string),
		AccessID:         d.Get("spaces_access_id").(string),
		SecretKey:        d.Get("spaces_secret_key").(string),
		TerraformVersion: terraformVersion,
	}

	return config.Client()
}
