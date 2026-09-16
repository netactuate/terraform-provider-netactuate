//go:build acctest

package netactuate

import "testing"

func TestAccNetactuateNKELBServices_contractUnobservable(t *testing.T) {
	t.Skip("criterion 7 cannot be observed in this provider build: netactuate_nke_lb_services is not registered, enable_cloud_lb is not in the NKE cluster schema, and gona exposes no NKE load balancer services API helper. A live Service and cloud-lb assertion need that surface before this acceptance test can be made executable.")
}
