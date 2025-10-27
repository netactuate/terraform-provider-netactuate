package netactuate

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func setValue(key string, value any, d *schema.ResourceData, diags *diag.Diagnostics) {
	err := d.Set(key, value)
	if err != nil {
		*diags = append(*diags, diag.Diagnostic{Severity: diag.Error, Summary: err.Error()})
	}
}

func updateValue(key string, value any, d *schema.ResourceData, diags *diag.Diagnostics) {
	_, exists := d.GetOk(key)
	if exists {
		setValue(key, value, d, diags)
	}
}
