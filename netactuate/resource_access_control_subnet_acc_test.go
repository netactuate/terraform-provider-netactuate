//go:build acctest

package netactuate

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAccessControlSubnetsDataSource_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "netactuate_access_control_subnets" "all" {}
`,
				Check: resource.TestMatchResourceAttr(
					"data.netactuate_access_control_subnets.all",
					"subnets.#",
					regexp.MustCompile(`^\d+$`),
				),
			},
		},
	})
}

func TestAccAccessControlSubnetResource_basic(t *testing.T) {
	// Creating an access control subnet on the live account is a decision for the
	// account owner, not something a test run makes on its behalf.
	testAccPreCheck(t, "NETACTUATE_ACC_ALLOW_SUBNET_WRITE")

	name := testAccName("uac-subnet")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_ALLOW_SUBNET_WRITE") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGone("netactuate_access_control_subnet.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessControlSubnetConfig(name, "192.0.2.0/24"),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_access_control_subnet", "netactuate_access_control_subnet.test"),
					resource.TestCheckResourceAttr("netactuate_access_control_subnet.test", "label", name),
					resource.TestCheckResourceAttr("netactuate_access_control_subnet.test", "subnet", "192.0.2.0/24"),
					resource.TestMatchResourceAttr("netactuate_access_control_subnet.test", "access_control_subnet_id", regexp.MustCompile(`^.+$`)),
				),
			},
			{
				Config: testAccAccessControlSubnetConfig(updatedName, "198.51.100.0/24"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_access_control_subnet.test", "label", updatedName),
					resource.TestCheckResourceAttr("netactuate_access_control_subnet.test", "subnet", "198.51.100.0/24"),
				),
			},
		},
	})
}

func testAccAccessControlSubnetConfig(label, subnet string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_access_control_subnet" "test" {
  label  = %q
  subnet = %q
}
`, label, subnet)
}
