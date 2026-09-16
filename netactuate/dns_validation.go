package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var dnsRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "PTR", "SPF", "SRV", "TXT"}

func dnsMxPriorityDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if diff.Get("type").(string) == "MX" {
		if _, ok := diff.GetOk("priority"); !ok {
			return fmt.Errorf("priority is required for MX records")
		}
	}
	return nil
}
