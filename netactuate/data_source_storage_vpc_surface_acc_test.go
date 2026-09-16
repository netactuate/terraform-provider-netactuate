//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateStorageSurfaceDataSources_hydrateFromAPI(t *testing.T) {
	name := testAccName("storage-surface")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	config := testAccStorageSurfaceDataSourcesConfig(name, locationID)
	relabelled := testAccStorageSurfaceDataSourcesConfigLabelled(name, locationID, "-b")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageSurfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_storage_bucket", "netactuate_storage_bucket.test"),
					testAccTrackFromState("netactuate_storage_object_store", "netactuate_storage_object_store.test"),
					testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
					testAccCheckStorageTypesDataSourceFromAPI("data.netactuate_storage_types.test"),
				),
			},
			{
				// Relabelling changes what the data sources depend on, so Terraform reads them
				// again rather than reusing the values recorded during the first apply.
				Config: relabelled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketsDataSourceFromAPI("data.netactuate_storage_buckets.test", "netactuate_storage_bucket.test"),
					testAccCheckStorageObjectStoresDataSourceFromAPI("data.netactuate_storage_object_stores.test", "netactuate_storage_object_store.test"),
					testAccCheckStorageBlockNamespacesDataSourceFromAPI("data.netactuate_storage_block_namespaces.test", "netactuate_storage_block_namespace.test"),
				),
			},
		},
	})
}

func TestAccNetactuateVPCSurfaceDataSources_hydrateFromAPI(t *testing.T) {
	name := testAccName("vpc-surface-data")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccVPCSurfaceDataSourcesConfig(name, locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCLocationsDataSourceFromAPI("data.netactuate_vpc_locations.test"),
					testAccCheckVPCIPReservationsDataSourceFromAPI("data.netactuate_vpc_ip_reservations.test", "netactuate_vpc.test"),
					testAccCheckVPCSSHSettingsDataSourceFromAPI("data.netactuate_vpc_ssh.test", "netactuate_vpc.test"),
				),
			},
		},
	})
}

func testAccCheckStorageSurfaceDestroy(s *terraform.State) error {
	if err := testAccCheckStorageBucketDestroy(s); err != nil {
		return err
	}
	if err := testAccCheckStorageObjectStoreDestroy(s); err != nil {
		return err
	}
	return testAccCheckStorageBlockNamespaceDestroy(s)
}

func testAccCheckStorageTypesDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		types, err := testAccClients().V3.ListStorageTypes()
		if err != nil {
			return err
		}
		if len(types) == 0 {
			return fmt.Errorf("storage types API returned no types")
		}
		// Terraform reads a data source during plan, before this test's own storage fixtures
		// exist, so the state count is a snapshot taken earlier than any count read here.
		// Comparing the two asserts that nothing was created in between, which is the opposite
		// of what this test does. Assert the state is populated and well formed instead.
		stateCount, err := strconv.Atoi(rs.Primary.Attributes["types.#"])
		if err != nil {
			return fmt.Errorf("%s state types.# is not an integer: %w", address, err)
		}
		if stateCount == 0 {
			return fmt.Errorf("%s state holds no storage types while the API returned %d", address, len(types))
		}
		known := map[string]bool{}
		for _, storageType := range types {
			known[storageType.Type] = true
		}
		for i := 0; i < stateCount; i++ {
			prefix := fmt.Sprintf("types.%d", i)
			got := rs.Primary.Attributes[prefix+".type"]
			if got == "" || !known[got] {
				return fmt.Errorf("%s state %s.type = %q, which the API does not report", address, prefix, got)
			}
		}
		return nil
	}
}

func testAccCheckStorageBucketsDataSourceFromAPI(dataAddress, bucketAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		bucketID, err := testAccID(s, bucketAddress)
		if err != nil {
			return err
		}
		buckets, err := testAccClients().V3.ListStorageBuckets()
		if err != nil {
			return err
		}
		return testAccCheckStorageListContains(s, dataAddress, "buckets", "bucket_id", bucketID, len(buckets))
	}
}

func testAccCheckStorageObjectStoresDataSourceFromAPI(dataAddress, storeAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		storeID, err := testAccID(s, storeAddress)
		if err != nil {
			return err
		}
		stores, err := testAccClients().V3.ListStorageObjectStores()
		if err != nil {
			return err
		}
		return testAccCheckStorageListContains(s, dataAddress, "object_stores", "object_store_id", storeID, len(stores))
	}
}

func testAccCheckStorageBlockNamespacesDataSourceFromAPI(dataAddress, nsAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		nsID, err := testAccID(s, nsAddress)
		if err != nil {
			return err
		}
		namespaces, err := testAccClients().V3.ListStorageBlockNamespaces()
		if err != nil {
			return err
		}
		return testAccCheckStorageListContains(s, dataAddress, "block_namespaces", "block_namespace_id", nsID, len(namespaces))
	}
}

func testAccCheckStorageListContains(s *terraform.State, dataAddress, listAttr, idAttr string, wantID, wantCount int) error {
	rs, err := testAccResourceState(s, dataAddress)
	if err != nil {
		return err
	}
	if got := rs.Primary.Attributes[listAttr+".#"]; got != strconv.Itoa(wantCount) {
		return fmt.Errorf("%s state %s count = %q, want API count %d", dataAddress, listAttr, got, wantCount)
	}
	for i := 0; i < wantCount; i++ {
		if rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, idAttr)] == strconv.Itoa(wantID) {
			return nil
		}
	}
	return fmt.Errorf("%s state %s has no item with %s = %d", dataAddress, listAttr, idAttr, wantID)
}

func testAccCheckVPCLocationsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		locations, err := testAccClients().V3.ListVPCLocations()
		if err != nil {
			return err
		}
		// The endpoint returns an empty list on this account, which is a successful read. What
		// this proves either way is that the data source reports exactly what the API returned.
		if got := rs.Primary.Attributes["locations.#"]; got != strconv.Itoa(len(locations)) {
			return fmt.Errorf("%s state location count = %q, want API count %d", address, got, len(locations))
		}
		for i, location := range locations {
			prefix := fmt.Sprintf("locations.%d", i)
			if rs.Primary.Attributes[prefix+".id"] != strconv.Itoa(location.ID) || rs.Primary.Attributes[prefix+".name"] != location.Name {
				return fmt.Errorf("%s state %s = id:%q name:%q, want API id:%d name:%q", address, prefix, rs.Primary.Attributes[prefix+".id"], rs.Primary.Attributes[prefix+".name"], location.ID, location.Name)
			}
		}
		return nil
	}
}

func testAccCheckVPCIPReservationsDataSourceFromAPI(dataAddress, vpcAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, dataAddress)
		if err != nil {
			return err
		}
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		reservations, err := testAccClients().V3.GetVPCIPReservations(vpcID)
		if err != nil {
			return err
		}
		for key, raw := range map[string][]byte{"gateways": reservations.Gateways, "interfaces": reservations.Interfaces, "vms": reservations.VMs} {
			want, err := compactSurfaceJSON(raw)
			if err != nil {
				return err
			}
			if got := rs.Primary.Attributes[key]; got != want {
				return fmt.Errorf("%s state %s = %q, want API JSON %q", dataAddress, key, got, want)
			}
		}
		return nil
	}
}

func testAccCheckVPCSSHSettingsDataSourceFromAPI(dataAddress, vpcAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, dataAddress)
		if err != nil {
			return err
		}
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		settings, err := testAccClients().V3.GetVPCSSHSettings(vpcID)
		if err != nil {
			return err
		}
		port := 0
		if settings.Port != nil {
			port = *settings.Port
		}
		if rs.Primary.Attributes["enabled"] != strconv.FormatBool(settings.Enabled) || rs.Primary.Attributes["port"] != strconv.Itoa(port) {
			return fmt.Errorf("%s state SSH settings = enabled:%q port:%q, want API enabled:%t port:%d", dataAddress, rs.Primary.Attributes["enabled"], rs.Primary.Attributes["port"], settings.Enabled, port)
		}
		return nil
	}
}

func testAccStorageSurfaceDataSourcesConfig(name string, locationID int) string {
	return testAccStorageSurfaceDataSourcesConfigLabelled(name, locationID, "")
}

func testAccStorageSurfaceDataSourcesConfigLabelled(name string, locationID int, suffix string) string {
	name += suffix
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_bucket" "test" {
  label       = %q
  location_id = %d
  capacity    = 1
}

resource "netactuate_storage_object_store" "test" {
  label       = %q
  location_id = %d
  capacity    = 1
}

resource "netactuate_storage_block_namespace" "test" {
  label       = %q
  location_id = %d
  capacity    = 1
}

data "netactuate_storage_types" "test" {}

data "netactuate_storage_buckets" "test" {
  depends_on = [netactuate_storage_bucket.test]
}

data "netactuate_storage_object_stores" "test" {
  depends_on = [netactuate_storage_object_store.test]
}

data "netactuate_storage_block_namespaces" "test" {
  depends_on = [netactuate_storage_block_namespace.test]
}
`, name+"-bucket", locationID, name+"-object-store", locationID, name+"-block-namespace", locationID)
}

func testAccVPCSurfaceDataSourcesConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label        = %q
  description  = %q
  location_id  = %d
  bastion_port = 2201
}

data "netactuate_vpc_locations" "test" {}

data "netactuate_vpc_ip_reservations" "test" {
  vpc_id = netactuate_vpc.test.vpc_id
}

data "netactuate_vpc_ssh" "test" {
  vpc_id = netactuate_vpc.test.vpc_id
}
`, name, name, locationID)
}
