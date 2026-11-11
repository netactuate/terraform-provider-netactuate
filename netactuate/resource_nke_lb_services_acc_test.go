//go:build acctest

package netactuate

import "testing"

func TestAccNetactuateNKELBServices(t *testing.T) {
	t.Skip("needs a live NKE cluster with the VPC LB add-on and a LoadBalancer Service")
}
