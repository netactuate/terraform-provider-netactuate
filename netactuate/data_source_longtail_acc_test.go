//go:build acctest

package netactuate

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateIPTransitDataSources_hydrateFromAPI(t *testing.T) {
	steps := []resource.TestStep{
		{
			Config: testAccIPTransitServicesDataSourceConfig(),
			Check:  testAccCheckIPTransitServicesDataSourceFromAPI("data.netactuate_iptransit_services.all", nil),
		},
	}
	if serviceID, ok := testAccOptionalEnvInt(t, "NETACTUATE_ACC_IPTRANSIT_SERVICE_ID"); ok {
		steps = append(steps, resource.TestStep{
			Config: testAccIPTransitServiceDataSourcesConfig(serviceID),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckIPTransitServiceDataSourceFromAPI("data.netactuate_iptransit_service.test"),
				testAccCheckIPTransitIPsDataSourceFromAPI("data.netactuate_iptransit_ips.test"),
				testAccCheckIPTransitPortsDataSourceFromAPI("data.netactuate_iptransit_ports.test"),
			),
		})
	} else {
		t.Log("NETACTUATE_ACC_IPTRANSIT_SERVICE_ID is unset because this account has no IP transit services, so singular IP transit data source coverage is skipped")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps:             steps,
	})
}

func TestAccNetactuateTransportDataSources_hydrateFromAPI(t *testing.T) {
	steps := []resource.TestStep{
		{
			Config: testAccTransportServicesDataSourceConfig(),
			Check:  testAccCheckTransportServicesDataSourceFromAPI("data.netactuate_transport_services.all", nil),
		},
	}
	if serviceID, ok := testAccOptionalEnvInt(t, "NETACTUATE_ACC_TRANSPORT_SERVICE_ID"); ok {
		steps = append(steps, resource.TestStep{
			Config: testAccTransportServiceDataSourcesConfig(serviceID),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckTransportServiceDataSourceFromAPI("data.netactuate_transport_service.test"),
				testAccCheckTransportPortsDataSourceFromAPI("data.netactuate_transport_ports.test"),
			),
		})
	} else {
		t.Log("NETACTUATE_ACC_TRANSPORT_SERVICE_ID is unset because this account has no transport services, so singular transport data source coverage is skipped")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps:             steps,
	})
}

func testAccCheckIPTransitServicesDataSourceFromAPI(address string, filter *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		services, err := testAccClients().V2.GetIPTransitServices(gona.ServiceListOptions{ServiceID: filter})
		if err != nil {
			return err
		}
		return testAccCheckServiceRows(address, "services", len(services), func(i int, attrs map[string]string) error {
			service := services[i]
			if attrs["service_id"] != strconv.Itoa(service.ID) || attrs["description"] != service.Description {
				return fmt.Errorf("%s.services.%d = id:%q description:%q, want API id:%d description:%q", address, i, attrs["service_id"], attrs["description"], service.ID, service.Description)
			}
			return nil
		})(s)
	}
}

func testAccCheckIPTransitServiceDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		serviceID, err := strconv.Atoi(rs.Primary.Attributes["service.0.service_id"])
		if err != nil {
			return fmt.Errorf("%s service id is invalid: %w", address, err)
		}
		service, err := testAccClients().V2.GetIPTransitService(serviceID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["service.#"]; got != "1" {
			return fmt.Errorf("%s state service count = %q, want 1", address, got)
		}
		if got := rs.Primary.Attributes["service.0.description"]; got != service.Description {
			return fmt.Errorf("%s state service.0.description = %q, want API description %q", address, got, service.Description)
		}
		return nil
	}
}

func testAccCheckIPTransitIPsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		serviceID, err := strconv.Atoi(rs.Primary.Attributes["rows.0.service_id"])
		if err != nil {
			return fmt.Errorf("%s rows.0.service_id is invalid: %w", address, err)
		}
		addresses, err := testAccClients().V2.GetIPTransitIPAddresses(&serviceID)
		if err != nil {
			return err
		}
		if len(addresses) == 0 {
			return fmt.Errorf("IP transit IP API returned no addresses for service %d", serviceID)
		}
		return testAccCheckServiceRows(address, "rows", len(addresses), func(i int, attrs map[string]string) error {
			row := addresses[i]
			if attrs["row_id"] != strconv.Itoa(row.ID) || attrs["service_id"] != strconv.Itoa(row.ServiceIPTransitID) || attrs["ip"] != row.IP {
				return fmt.Errorf("%s.rows.%d = id:%q service:%q ip:%q, want API id:%d service:%d ip:%q", address, i, attrs["row_id"], attrs["service_id"], attrs["ip"], row.ID, row.ServiceIPTransitID, row.IP)
			}
			return nil
		})(s)
	}
}

func testAccCheckIPTransitPortsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		serviceID, err := strconv.Atoi(rs.Primary.Attributes["rows.0.service_id"])
		if err != nil {
			return fmt.Errorf("%s rows.0.service_id is invalid: %w", address, err)
		}
		ports, err := testAccClients().V2.GetIPTransitPorts(&serviceID)
		if err != nil {
			return err
		}
		if len(ports) == 0 {
			return fmt.Errorf("IP transit ports API returned no ports for service %d", serviceID)
		}
		return testAccCheckServiceRows(address, "rows", len(ports), func(i int, attrs map[string]string) error {
			port := ports[i]
			if attrs["row_id"] != strconv.Itoa(port.ID) || attrs["service_id"] != strconv.Itoa(port.ServiceIPTransitID) || attrs["name"] != port.Name {
				return fmt.Errorf("%s.rows.%d = id:%q service:%q name:%q, want API id:%d service:%d name:%q", address, i, attrs["row_id"], attrs["service_id"], attrs["name"], port.ID, port.ServiceIPTransitID, port.Name)
			}
			return nil
		})(s)
	}
}

func testAccCheckTransportServicesDataSourceFromAPI(address string, filter *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		services, err := testAccClients().V2.GetTransportServices(gona.ServiceListOptions{ServiceID: filter})
		if err != nil {
			return err
		}
		return testAccCheckServiceRows(address, "services", len(services), func(i int, attrs map[string]string) error {
			service := services[i]
			if attrs["service_id"] != strconv.Itoa(service.ID) || attrs["description"] != service.Description {
				return fmt.Errorf("%s.services.%d = id:%q description:%q, want API id:%d description:%q", address, i, attrs["service_id"], attrs["description"], service.ID, service.Description)
			}
			return nil
		})(s)
	}
}

func testAccCheckTransportServiceDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		serviceID, err := strconv.Atoi(rs.Primary.Attributes["service.0.service_id"])
		if err != nil {
			return fmt.Errorf("%s service id is invalid: %w", address, err)
		}
		service, err := testAccClients().V2.GetTransportService(serviceID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["service.#"]; got != "1" {
			return fmt.Errorf("%s state service count = %q, want 1", address, got)
		}
		if got := rs.Primary.Attributes["service.0.description"]; got != service.Description {
			return fmt.Errorf("%s state service.0.description = %q, want API description %q", address, got, service.Description)
		}
		return nil
	}
}

func testAccCheckTransportPortsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		serviceID, err := strconv.Atoi(rs.Primary.Attributes["rows.0.service_id"])
		if err != nil {
			return fmt.Errorf("%s rows.0.service_id is invalid: %w", address, err)
		}
		ports, err := testAccClients().V2.GetTransportPorts(&serviceID)
		if err != nil {
			return err
		}
		if len(ports) == 0 {
			return fmt.Errorf("transport ports API returned no ports for service %d", serviceID)
		}
		return testAccCheckServiceRows(address, "rows", len(ports), func(i int, attrs map[string]string) error {
			port := ports[i]
			if attrs["row_id"] != strconv.Itoa(port.ID) || attrs["service_id"] != strconv.Itoa(port.ServiceTransportID) || attrs["name"] != port.Name {
				return fmt.Errorf("%s.rows.%d = id:%q service:%q name:%q, want API id:%d service:%d name:%q", address, i, attrs["row_id"], attrs["service_id"], attrs["name"], port.ID, port.ServiceTransportID, port.Name)
			}
			return nil
		})(s)
	}
}

func testAccCheckServiceRows(address, listAttr string, wantCount int, check func(int, map[string]string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes[listAttr+".#"]; got != strconv.Itoa(wantCount) {
			return fmt.Errorf("%s state %s count = %q, want API count %d", address, listAttr, got, wantCount)
		}
		for i := 0; i < wantCount; i++ {
			attrs := map[string]string{}
			for _, field := range []string{"row_id", "service_id", "name", "ip"} {
				attrs[field] = rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, field)]
			}
			if err := check(i, attrs); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccResourceState(s *terraform.State, address string) (*terraform.ResourceState, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok || rs.Primary == nil {
		return nil, fmt.Errorf("not found: %s", address)
	}
	return rs, nil
}

func testAccOptionalEnvInt(t *testing.T, name string) (int, bool) {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("%s must be an integer for this acceptance test: %v", name, err)
	}
	return parsed, true
}

func testAccIPTransitServicesDataSourceConfig() string {
	return testAccProviderConfig() + `
data "netactuate_iptransit_services" "all" {}
`
}

func testAccIPTransitServiceDataSourcesConfig(serviceID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_iptransit_service" "test" {
  service_id = %d
}

data "netactuate_iptransit_ips" "test" {
  service_iptransit_id = %d
}

data "netactuate_iptransit_ports" "test" {
  service_iptransit_id = %d
}
`, serviceID, serviceID, serviceID)
}

func testAccTransportServicesDataSourceConfig() string {
	return testAccProviderConfig() + `
data "netactuate_transport_services" "all" {}
`
}

func testAccTransportServiceDataSourcesConfig(serviceID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_transport_service" "test" {
  service_id = %d
}

data "netactuate_transport_ports" "test" {
  service_transport_id = %d
}
`, serviceID, serviceID)
}
