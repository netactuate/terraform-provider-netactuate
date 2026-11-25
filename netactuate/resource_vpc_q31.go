package netactuate

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func vpcQ31CountSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeInt, Computed: true, Description: "Floating IPv4 address count reported by vapi3."}
}
func readVPCQ31(d *schema.ResourceData, v *gona.VPC, diags *diag.Diagnostics) {
	setValue("max_floating_ipv4s", v.MaxFloatingIPv4s, d, diags)
	setValue("used_floating_ipv4s", v.UsedFloatingIPv4s, d, diags)
}
