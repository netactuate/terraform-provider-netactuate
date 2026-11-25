package netactuate

import (
	"errors"
	"testing"
)

func TestVPCDeleteHeldByVMs(t *testing.T) {
	cases := map[string]bool{
		"API error on DELETE /vpcs/367: HTTP 400: You currently have servers connected to this VPC. Detach them before deleting it.": true,
		"API error on DELETE /vpcs/1: HTTP 400: There are virtual machines in this VPC.":                                             true,
		"API error on DELETE /vpcs/1: HTTP 403: forbidden":                                                                           false,
	}
	for msg, want := range cases {
		if got := vpcDeleteHeldByVMs(errors.New(msg)); got != want {
			t.Errorf("vpcDeleteHeldByVMs(%q) = %v, want %v", msg, got, want)
		}
	}
}

// vapi3 fills advertise with the VPC network when it is omitted, so the attribute must be
// Computed or every config without it plans a change (LD-237).
func TestVPCAttachmentAdvertiseComputed(t *testing.T) {
	s := resourceVPCAttachment().Schema["advertise"]
	if s == nil || !s.Optional || !s.Computed {
		t.Fatalf("advertise must be Optional and Computed, got %+v", s)
	}
}
