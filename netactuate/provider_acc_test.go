//go:build acctest

package netactuate

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"netactuate": func() (*schema.Provider, error) {
		return Provider(), nil
	},
}

var testAccCreated = map[string]map[int]struct{}{
	"netactuate_vpc":                         {},
	"netactuate_server":                      {},
	"netactuate_nke_cluster":                 {},
	"netactuate_router":                      {},
	"netactuate_image":                       {},
	"netactuate_dns_zone":                    {},
	"netactuate_dns_record":                  {},
	"netactuate_storage_bucket":              {},
	"netactuate_storage_object_store":        {},
	"netactuate_storage_block_namespace":     {},
	"netactuate_storage_block_volume":        {},
	"netactuate_magic_mesh":                  {},
	"netactuate_oidc_client":                 {},
	"netactuate_access_control_subnet":       {},
	"netactuate_cloud_floating_ipv4":         {},
	"netactuate_dedicated_server_buy":        {},
	"netactuate_dedicated_server_buy_build":  {},
	"netactuate_dedicated_server_deployment": {},
}

func init() {
	resource.AddTestSweepers("netactuate_vpc", &resource.Sweeper{
		Name: "netactuate_vpc",
		F:    testSweepTracked("netactuate_vpc", func(c *ProviderClients, id int) error { return c.V3.DeleteVPC(id) }),
	})
	resource.AddTestSweepers("netactuate_server", &resource.Sweeper{
		Name: "netactuate_server",
		F:    testSweepTracked("netactuate_server", func(c *ProviderClients, id int) error { _, err := c.V2.DeleteServer(id, true); return err }),
	})
	resource.AddTestSweepers("netactuate_nke_cluster", &resource.Sweeper{
		Name: "netactuate_nke_cluster",
		F:    testSweepTracked("netactuate_nke_cluster", func(c *ProviderClients, id int) error { return c.V3.DeleteNKECluster(id) }),
	})
	resource.AddTestSweepers("netactuate_router", &resource.Sweeper{
		Name: "netactuate_router",
		F:    testSweepTracked("netactuate_router", func(c *ProviderClients, id int) error { return c.V3.DeleteRouter(id) }),
	})
	resource.AddTestSweepers("netactuate_image", &resource.Sweeper{
		Name: "netactuate_image",
		F:    testSweepTracked("netactuate_image", func(c *ProviderClients, id int) error { _, err := c.V2.DeleteImage(id); return err }),
	})
	resource.AddTestSweepers("netactuate_dns_record", &resource.Sweeper{
		Name: "netactuate_dns_record",
		F:    testSweepTracked("netactuate_dns_record", func(c *ProviderClients, id int) error { return c.V2.DeleteRecord(id) }),
	})
	resource.AddTestSweepers("netactuate_dns_zone", &resource.Sweeper{
		Name: "netactuate_dns_zone",
		F: testSweepTracked("netactuate_dns_zone", func(c *ProviderClients, id int) error {
			records, err := c.V2.ListRecords(id)
			if err != nil && !gona.IsNotFound(err) {
				return err
			}
			for _, record := range records {
				if err := c.V2.DeleteRecord(record.ID); err != nil && !gona.IsNotFound(err) {
					return err
				}
			}
			return c.V2.DeleteZone(id)
		}),
	})
	resource.AddTestSweepers("netactuate_storage_bucket", &resource.Sweeper{
		Name: "netactuate_storage_bucket",
		F:    testSweepTracked("netactuate_storage_bucket", func(c *ProviderClients, id int) error { return c.V3.DeleteStorageBucket(id) }),
	})
	resource.AddTestSweepers("netactuate_storage_object_store", &resource.Sweeper{
		Name: "netactuate_storage_object_store",
		F:    testSweepTracked("netactuate_storage_object_store", func(c *ProviderClients, id int) error { return c.V3.DeleteStorageObjectStore(id) }),
	})
	resource.AddTestSweepers("netactuate_storage_block_namespace", &resource.Sweeper{
		Name: "netactuate_storage_block_namespace",
		F:    testSweepTracked("netactuate_storage_block_namespace", func(c *ProviderClients, id int) error { return c.V3.DeleteStorageBlockNamespace(id) }),
	})
	resource.AddTestSweepers("netactuate_storage_block_volume", &resource.Sweeper{
		Name: "netactuate_storage_block_volume",
		F:    testSweepTracked("netactuate_storage_block_volume", func(c *ProviderClients, id int) error { return c.V3.DeleteStorageBlockVolume(id) }),
	})
	resource.AddTestSweepers("netactuate_magic_mesh", &resource.Sweeper{
		Name: "netactuate_magic_mesh",
		F:    testSweepTracked("netactuate_magic_mesh", func(c *ProviderClients, id int) error { return c.V3.DeleteMagicMesh(id) }),
	})
	resource.AddTestSweepers("netactuate_oidc_client", &resource.Sweeper{
		Name: "netactuate_oidc_client",
		F:    testSweepTracked("netactuate_oidc_client", func(c *ProviderClients, id int) error { return c.V3.DeleteOIDCClient(id) }),
	})
	resource.AddTestSweepers("netactuate_access_control_subnet", &resource.Sweeper{
		Name: "netactuate_access_control_subnet",
		F: testSweepTracked("netactuate_access_control_subnet", func(c *ProviderClients, id int) error {
			return c.V2.DeleteAccessControlSubnet(strconv.Itoa(id))
		}),
	})
	resource.AddTestSweepers("netactuate_cloud_floating_ipv4", &resource.Sweeper{
		Name: "netactuate_cloud_floating_ipv4",
		F:    testSweepTracked("netactuate_cloud_floating_ipv4", func(c *ProviderClients, id int) error { return c.V3.DeleteCloudFloatingIPv4(id) }),
	})
	resource.AddTestSweepers("netactuate_dedicated_server_buy", &resource.Sweeper{
		Name: "netactuate_dedicated_server_buy",
		F:    testSweepTracked("netactuate_dedicated_server_buy", func(c *ProviderClients, id int) error { return c.V2.DeleteDedicatedServer(id, nil) }),
	})
	resource.AddTestSweepers("netactuate_dedicated_server_buy_build", &resource.Sweeper{
		Name: "netactuate_dedicated_server_buy_build",
		F:    testSweepTracked("netactuate_dedicated_server_buy_build", func(c *ProviderClients, id int) error { return c.V2.DeleteDedicatedServer(id, nil) }),
	})
	resource.AddTestSweepers("netactuate_dedicated_server_deployment", &resource.Sweeper{
		Name: "netactuate_dedicated_server_deployment",
		F:    testSweepTracked("netactuate_dedicated_server_deployment", func(c *ProviderClients, id int) error { return c.V2.DeleteDedicatedServer(id, nil) }),
	})
}

func testAccPreCheck(t *testing.T, names ...string) {
	t.Helper()
	required := append([]string{"NETACTUATE_API_KEY"}, names...)
	for _, name := range required {
		if os.Getenv(name) == "" {
			t.Skipf("%s must be set for NetActuate acceptance tests", name)
		}
	}
}

func testAccName(testName string) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("random suffix: %v", err))
	}
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return fmt.Sprintf("nah-%s-%s", string(out), testName)
}

func testAccEnvString(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		if os.Getenv(resource.EnvTfAcc) == "" {
			return ""
		}
		t.Skipf("%s must be set for this acceptance test", name)
	}
	return v
}

func testAccEnvInt(t *testing.T, name string) int {
	t.Helper()
	v := testAccEnvString(t, name)
	i, err := strconv.Atoi(v)
	if err != nil {
		t.Skipf("%s must be an integer for this acceptance test: %v", name, err)
	}
	return i
}

func testAccClients() *ProviderClients {
	key := os.Getenv("NETACTUATE_API_KEY")
	return &ProviderClients{
		V2: gona.NewClient(key),
		V3: gona.NewV3Client(key, os.Getenv("NETACTUATE_API_URL_V3")),
	}
}

func testAccTrack(resourceType string, id int) {
	if id == 0 {
		return
	}
	if _, ok := testAccCreated[resourceType]; !ok {
		testAccCreated[resourceType] = map[int]struct{}{}
	}
	testAccCreated[resourceType][id] = struct{}{}
}

func testAccTrackFromState(resourceType, address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		testAccTrack(resourceType, id)
		return nil
	}
}

func testAccCheckGone(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rs, ok := s.RootModule().Resources[address]; ok && rs.Primary != nil && rs.Primary.ID != "" {
			return fmt.Errorf("%s still in state with ID %s", address, rs.Primary.ID)
		}
		return nil
	}
}

func testAccID(s *terraform.State, address string) (int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, fmt.Errorf("not found: %s", address)
	}
	id, err := strconv.Atoi(rs.Primary.ID)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid ID %q: %w", address, rs.Primary.ID, err)
	}
	return id, nil
}

func testSweepTracked(resourceType string, del func(*ProviderClients, int) error) resource.SweeperFunc {
	return func(region string) error {
		if os.Getenv("NETACTUATE_API_KEY") == "" {
			return nil
		}
		clients := testAccClients()
		for id := range testAccCreated[resourceType] {
			if err := del(clients, id); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccProviderConfig() string {
	if v := os.Getenv("NETACTUATE_API_URL"); v != "" {
		return fmt.Sprintf(`
provider "netactuate" {
  api_url = %q
  api_url_v3 = %q
}
`, v, os.Getenv("NETACTUATE_API_URL_V3"))
	}
	return fmt.Sprintf(`
provider "netactuate" {
  api_url_v3 = %q
}
`, os.Getenv("NETACTUATE_API_URL_V3"))
}
