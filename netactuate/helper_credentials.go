package netactuate

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// setCredentialValue writes a credential into state ONLY when the API actually returned one.
//
// WHY. Storage credentials are issued at CREATE and are NOT returned by any read.
// GET /storage/buckets/{id} answers with "credentials": {} even for a bucket created moments
// earlier. Writing those empty values into state makes the FIRST REFRESH destroy the access
// key, secret key, user key and endpoints the customer was issued, and anything referencing
// them silently becomes empty.
//
// That is silent state corruption applied to credentials, which is worse than losing an
// ordinary attribute: the value cannot be recovered by reading the API again, only by
// destroying and recreating the resource.
func setCredentialValue(key string, apiValue interface{}, d *schema.ResourceData, diags *diag.Diagnostics) {
	switch v := apiValue.(type) {
	case string:
		if v == "" {
			return
		}
	case []string:
		if len(v) == 0 {
			return
		}
	}
	setValue(key, apiValue, d, diags)
}
