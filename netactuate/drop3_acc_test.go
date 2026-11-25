//go:build acctest

package netactuate

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func drop3AccConfig(t *testing.T, kind, name string, step int) string {
	t.Helper()
	fixture := drop2AccFixture(t, name, kind)
	group := fmt.Sprintf("\nresource \"netactuate_security_group\" \"parent\" { name = %q }\n", name+"-parent")
	vpc := "netactuate_vpc.test.vpc_id"
	switch kind {
	case "security_group":
		if step > 0 {
			name += "-renamed"
		}
		return fixture + fmt.Sprintf("\nresource \"netactuate_security_group\" \"test\" { name = %q\n description = \"Q40 lifecycle\" }\n", name)
	case "security_group_rule":
		port := 443
		if step > 0 {
			port = 8443
		}
		return fixture + group + fmt.Sprintf(`
resource "netactuate_security_group_rule" "test" {
 security_group_id = netactuate_security_group.parent.security_group_id
 ip_version = 4
 protocol = "tcp"
 port_start = %d
 port_end = %d
 network = "192.0.2.0/24"
}
resource "netactuate_security_group_rule" "source" {
 security_group_id = netactuate_security_group.parent.security_group_id
 ip_version = 4
 source_security_group_id = netactuate_security_group.parent.security_group_id
}
`, port, port)
	case "security_group_member":
		vm := testAccEnvInt(t, "NETACTUATE_ACC_Q40_MBPKGID")
		return fixture + group + fmt.Sprintf(`
resource "netactuate_vpc_server_attachment" "fixture" {
 vpc_id = %s
 mbpkgid = %d
}
resource "netactuate_security_group_member" "test" {
 security_group_id = netactuate_security_group.parent.security_group_id
 vpc_id = %s
 mbpkgid = %d
 depends_on = [netactuate_vpc_server_attachment.fixture]
}
`, vpc, vm, vpc, vm)
	case "vpc_security_group_enforcement":
		return fixture + fmt.Sprintf("\nresource \"netactuate_vpc_security_group_enforcement\" \"test\" { vpc_id = %s\n enabled = true }\n", vpc)
	case "vpc_subnet":
		return fixture + fmt.Sprintf("\nresource \"netactuate_vpc_subnet\" \"test\" { vpc_id = %s\n network = \"172.31.251.0/24\"\n label = %q }\n", vpc, name)
	case "vpc_server_attachment":
		return fixture + fmt.Sprintf("\nresource \"netactuate_vpc_server_attachment\" \"test\" { vpc_id = %s\n mbpkgid = %d\n subnet_id = 0 }\n", vpc, testAccEnvInt(t, "NETACTUATE_ACC_Q40_MBPKGID"))
	case "vpc_attachment":
		bgp := ""
		if step == 1 || step == 2 {
			bgp = fmt.Sprintf("bgp { peer_asn = 65010\n max_prefix = %d }", 100+step)
		}
		return fixture + fmt.Sprintf(`
resource "netactuate_vpc_attachment" "test" {
 vpc_id = %s
 type = "customer-router"
 address = "10.77.0.20"
 description = %q
 accept = ["192.168.50.0/24"]
 %s
}
`, vpc, name, bgp)
	}
	panic("unknown Q40 resource")
}
func drop3AccResource(kind string) *schema.Resource {
	if kind == "vpc_attachment" {
		return resourceVPCAttachment()
	}
	return drop3Resources()[kind]
}
func drop3AccDestroy(s *terraform.State) error {
	c := testAccClients()
	for _, rs := range s.RootModule().Resources {
		kind := strings.TrimPrefix(rs.Type, "netactuate_")
		r := drop3AccResource(kind)
		if r == nil {
			continue
		}
		if kind == "vpc_security_group_enforcement" {
			v, e := strconv.Atoi(rs.Primary.ID)
			if e != nil {
				return e
			}
			out, e := c.V3.GetVPCSecurityGroupEnforcement(v)
			if e != nil && !gona.IsV3NotFound(e) {
				return e
			}
			if e == nil && out.Enabled {
				return fmt.Errorf("enforcement remains on")
			}
			continue
		}
		d := r.Data(rs.Primary)
		if ds := r.ReadContext(context.Background(), d, c); ds.HasError() {
			return fmt.Errorf("%v", ds)
		}
		if d.Id() != "" {
			return fmt.Errorf("%s still exists", rs.Type)
		}
	}
	return testAccCheckVPCDestroy(s)
}
func drop3AccRun(t *testing.T, kind string, oob bool) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC is required")
	}
	drop2AccPrecheck(t, false)
	name := "nah-q40-" + strings.TrimPrefix(testAccName("drop3"), "nah-")
	config := drop3AccConfig(t, kind, name, 0)
	address := "netactuate_" + kind + ".test"
	savedID := ""
	vpcID := 0
	check := func(s *terraform.State) error {
		rs := s.RootModule().Resources[address]
		if rs == nil || rs.Primary.ID == "" {
			return fmt.Errorf("missing resource")
		}
		id := rs.Primary.ID
		if savedID != "" {
			if kind == "security_group_rule" {
				if savedID == id {
					return fmt.Errorf("rule was not replaced")
				}
			} else if savedID != id {
				return fmt.Errorf("resource replaced")
			}
		}
		savedID = id
		vpcID, _ = testAccID(s, "netactuate_vpc.test")
		if kind == "security_group_member" {
			if rs.Primary.Attributes["enforcement_state"] == "" {
				return fmt.Errorf("missing enforcement_state")
			}
			parent := s.RootModule().Resources["netactuate_security_group.parent"]
			groupID, e := strconv.Atoi(parent.Primary.ID)
			if e != nil {
				return e
			}
			e = testAccClients().V3.DeleteSecurityGroup(groupID)
			if e == nil || !strings.Contains(e.Error(), "The security group has members.") {
				return fmt.Errorf("expected member dependency refusal: %v", e)
			}
		}
		return testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test")(s)
	}
	steps := []resource.TestStep{{Config: config, Check: check}}
	if oob {
		steps = append(steps, resource.TestStep{PreConfig: func() {
			r := drop3AccResource(kind)
			d := r.Data(&terraform.InstanceState{ID: savedID})
			c := testAccClients()
			if kind == "vpc_security_group_enforcement" {
				if e := c.V3.DeleteVPC(vpcID); e != nil {
					t.Fatal(e)
				}
				if e := drop2Wait(context.Background(), d.Timeout(schema.TimeoutDelete), func() (bool, error) {
					_, e := c.V3.GetVPC(vpcID)
					if gona.IsV3NotFound(e) {
						return true, nil
					}
					return false, e
				}); e != nil {
					t.Fatal(e)
				}
			} else {
				if ds := r.DeleteContext(context.Background(), d, c); ds.HasError() {
					t.Fatal(ds)
				}
			}
			d.SetId(savedID)
			if ds := r.DeleteContext(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			d.SetId(savedID)
			if ds := r.ReadContext(context.Background(), d, c); ds.HasError() || d.Id() != "" {
				t.Fatalf("read after deletion: %v", ds)
			}
		}, RefreshState: true, ExpectNonEmptyPlan: true, Check: testAccCheckGone(address)})
	} else {
		if kind == "security_group" || kind == "security_group_rule" {
			config = drop3AccConfig(t, kind, name, 1)
			steps = append(steps, resource.TestStep{Config: config, Check: check})
		}
		if kind == "vpc_attachment" {
			for i := 1; i <= 3; i++ {
				config = drop3AccConfig(t, kind, name, i)
				steps = append(steps, resource.TestStep{Config: config, Check: check})
			}
		}
		steps = append(steps, resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}, resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: false})
		if kind == "vpc_security_group_enforcement" {
			steps = append(steps, resource.TestStep{Config: drop2AccFixture(t, name, kind), Check: func(s *terraform.State) error {
				out, e := testAccClients().V3.GetVPCSecurityGroupEnforcement(vpcID)
				if e != nil {
					return e
				}
				if out.Enabled {
					return fmt.Errorf("destroy did not disable enforcement")
				}
				return nil
			}})
		}
		if kind == "vpc_subnet" {
			overlap := config + fmt.Sprintf("\nresource \"netactuate_vpc_subnet\" \"overlap\" { vpc_id = netactuate_vpc.test.vpc_id\n network = \"172.31.251.0/24\"\n label = %q\n depends_on = [netactuate_vpc_subnet.test] }\n", name+"-overlap")
			steps = append(steps, resource.TestStep{Config: overlap, ExpectError: regexp.MustCompile(`(?i)overlap`)}, resource.TestStep{Config: config})
		}
	}
	resource.Test(t, resource.TestCase{PreCheck: func() { drop2AccPrecheck(t, false) }, ProviderFactories: testAccProviderFactories, CheckDestroy: drop3AccDestroy, Steps: steps})
}

func TestAccNetactuateSecurityGroup_lifecycle(t *testing.T) { drop3AccRun(t, "security_group", false) }

func TestAccNetactuateSecurityGroup_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "security_group", true)
}

func TestAccNetactuateSecurityGroupRule_lifecycle(t *testing.T) {
	drop3AccRun(t, "security_group_rule", false)
}

func TestAccNetactuateSecurityGroupRule_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "security_group_rule", true)
}

func TestAccNetactuateSecurityGroupMember_lifecycle(t *testing.T) {
	drop3AccRun(t, "security_group_member", false)
}

func TestAccNetactuateSecurityGroupMember_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "security_group_member", true)
}

func TestAccNetactuateVPCSecurityGroupEnforcement_lifecycle(t *testing.T) {
	drop3AccRun(t, "vpc_security_group_enforcement", false)
}

func TestAccNetactuateVPCSecurityGroupEnforcement_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "vpc_security_group_enforcement", true)
}

func TestAccNetactuateVPCSubnet_lifecycle(t *testing.T) { drop3AccRun(t, "vpc_subnet", false) }

func TestAccNetactuateVPCSubnet_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "vpc_subnet", true)
}

func TestAccNetactuateVPCServerAttachment_lifecycle(t *testing.T) {
	drop3AccRun(t, "vpc_server_attachment", false)
}

func TestAccNetactuateVPCServerAttachment_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "vpc_server_attachment", true)
}

func TestAccNetactuateVPCAttachmentBGP_lifecycle(t *testing.T) {
	drop3AccRun(t, "vpc_attachment", false)
}

func TestAccNetactuateVPCAttachmentBGP_OutOfBandDeleteIdempotent(t *testing.T) {
	drop3AccRun(t, "vpc_attachment", true)
}
