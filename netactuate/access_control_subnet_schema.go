package netactuate

import (
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const accessControlSubnetWarning = "WARNING: Adding the first user access control subnet can restrict account access to that subnet. Include your own current address or network before applying."

func accessControlSubnetFieldsSchema(configurable bool) map[string]*schema.Schema {
	idSchema := &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Access control subnet ID returned by the API.",
	}
	labelSchema := &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Human-readable label for the access control subnet.",
	}
	subnetSchema := &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Source network in CIDR notation.",
	}

	if configurable {
		labelSchema.Computed = false
		labelSchema.Required = true
		subnetSchema.Computed = false
		subnetSchema.Required = true
		subnetSchema.ValidateDiagFunc = validateAccessControlSubnetCIDR
		subnetSchema.Description = accessControlSubnetWarning + " Source network in CIDR notation."
	}

	return map[string]*schema.Schema{
		"access_control_subnet_id": idSchema,
		"label":                    labelSchema,
		"subnet":                   subnetSchema,
	}
}

func validateAccessControlSubnetCIDR(v interface{}, p cty.Path) diag.Diagnostics {
	s, ok := v.(string)
	if !ok || s == "" {
		return diag.Diagnostics{diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Invalid access control subnet",
			Detail:        "subnet must be a non-empty CIDR string.",
			AttributePath: p,
		}}
	}
	if _, _, err := net.ParseCIDR(s); err != nil {
		return diag.Diagnostics{diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Invalid access control subnet",
			Detail:        fmt.Sprintf("%q is not valid CIDR notation.", s),
			AttributePath: p,
		}}
	}
	return nil
}
