//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateVPCBackendTemplate_ClearBackendHostsAPIAndPlan(t *testing.T) {
	name := testAccName("backend-template")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	configWithHosts := testAccVPCBackendTemplateConfig(name, locationID, true)
	configWithoutHosts := testAccVPCBackendTemplateConfig(name, locationID, false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: configWithHosts,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
					testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name, 2),
					testAccCheckVPCBackendTemplatesDataSourceContains("data.netactuate_vpc_backend_templates.test", "netactuate_vpc_backend_template.test"),
				),
			},
			{
				Config: configWithoutHosts,
				Check:  testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name, 0),
			},
			{
				ResourceName:      "netactuate_vpc_backend_template.test",
				ImportState:       true,
				ImportStateIdFunc: testAccBackendTemplateImportID("netactuate_vpc_backend_template.test"),
				ImportStateVerify: true,
			},
			{
				Config:   configWithoutHosts,
				PlanOnly: true,
			},
		},
	})
}

func testAccTrackBackendTemplateFromState(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, templateID, err := testAccBackendTemplateID(s, address)
		if err != nil {
			return err
		}
		testAccTrack("netactuate_vpc_backend_template", templateID)
		return nil
	}
}

func testAccCheckVPCBackendTemplateAPI(address, name string, backendHostCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, templateID, err := testAccBackendTemplateID(s, address)
		if err != nil {
			return err
		}
		tmpl, err := testAccClients().V3.GetVPCBackendTemplate(vpcID, templateID)
		if err != nil {
			return err
		}
		if tmpl.Name != name {
			return fmt.Errorf("API backend template name = %q, want %q", tmpl.Name, name)
		}
		if len(tmpl.BackendHosts) != backendHostCount {
			return fmt.Errorf("API backend host count = %d, want %d", len(tmpl.BackendHosts), backendHostCount)
		}
		return nil
	}
}

func testAccCheckVPCBackendTemplatesDataSourceContains(dataAddress, resourceAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, templateID, err := testAccBackendTemplateID(s, resourceAddress)
		if err != nil {
			return err
		}
		dataResource, ok := s.RootModule().Resources[dataAddress]
		if !ok || dataResource.Primary == nil {
			return fmt.Errorf("%s not found in state", dataAddress)
		}
		countRaw, ok := dataResource.Primary.Attributes["templates.#"]
		if !ok {
			return fmt.Errorf("%s.templates count is missing", dataAddress)
		}
		count, err := strconv.Atoi(countRaw)
		if err != nil {
			return fmt.Errorf("%s.templates count %q is invalid: %w", dataAddress, countRaw, err)
		}
		if count == 0 {
			return fmt.Errorf("%s.templates is empty", dataAddress)
		}
		wantID := strconv.Itoa(templateID)
		for i := 0; i < count; i++ {
			if dataResource.Primary.Attributes[fmt.Sprintf("templates.%d.backend_template_id", i)] == wantID {
				return nil
			}
		}
		return fmt.Errorf("%s.templates does not contain %s template ID %d", dataAddress, resourceAddress, templateID)
	}
}

func testAccCheckVPCBackendTemplateDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc_backend_template" {
			continue
		}
		vpcID, templateID, err := parseBackendTemplateID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetVPCBackendTemplate(vpcID, templateID); err == nil {
			return fmt.Errorf("VPC backend template still exists: %s", rs.Primary.ID)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccBackendTemplateID(s *terraform.State, address string) (int, int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, 0, fmt.Errorf("not found: %s", address)
	}
	return parseBackendTemplateID(rs.Primary.ID)
}

func testAccBackendTemplateImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		vpcID, templateID, err := testAccBackendTemplateID(s, address)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d/%d", vpcID, templateID), nil
	}
}

func testAccVPCBackendTemplateConfig(name string, locationID int, withHosts bool) string {
	backendHosts := ""
	if withHosts {
		backendHosts = `
  backend_host {
    name    = "nah-backend-one"
    address = "192.0.2.10"
  }

  backend_host {
    name    = "nah-backend-two"
    address = "192.0.2.11"
  }
`
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_backend_template" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  name        = %q
  description = "nah acceptance backend template"
%s}

data "netactuate_vpc_backend_templates" "test" {
  vpc_id = netactuate_vpc.test.vpc_id

  depends_on = [netactuate_vpc_backend_template.test]
}
`, name, name, locationID, name, backendHosts)
}
