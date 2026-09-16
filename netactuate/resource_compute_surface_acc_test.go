//go:build acctest

package netactuate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

const testAccSSHPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEoVhgyuxnVaZ7tqk8Xlk4m6G3XlZ0w4u5OskUrXnPks nah-acceptance"

func TestAccNetactuateServer_APIReadImportPlanVPCAndOutOfBandDelete(t *testing.T) {
	name := testAccName("server")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	imageName := testAccEnvString(t, "NETACTUATE_ACC_IMAGE_NAME")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	configID := testAccServerWithDataSourceConfigID(name, locationID, imageID, plan, contractID, password)
	configName := testAccServerConfigName(name, locationName, imageName, plan, contractID, password)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_LOCATION_NAME",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_IMAGE_NAME",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckServerAPI("netactuate_server.test", "netactuate_vpc.test", name+".example.invalid", plan, locationID, imageID),
					testAccCheckServerDataSourceAPI("data.netactuate_server.test", name+".example.invalid", plan, locationID, imageID),
				),
			},
			{
				ResourceName:            "netactuate_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "allow_downsize_reboot", "package_billing", "params"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_server"] {
						jobID, err := clients.V2.DeleteServer(id, true)
						if err != nil {
							continue
						}
						if d := wait4JobStatus("delete", jobID, clients.V2); d != nil && d.HasError() {
							continue
						}
						_ = clients.V2.UnlinkServer(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_server.test"),
			},
		},
	})
}

func TestAccNetactuateSSHKey_APIReadImportPlanDataSourceAndOutOfBandDelete(t *testing.T) {
	name := testAccName("sshkey")
	config := testAccSSHKeyConfig(name)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
					testAccCheckSSHKeyAPI("netactuate_sshkey.test", name, testAccSSHPublicKey),
					testAccCheckSSHKeyDataSourceAPI("data.netactuate_sshkey.test", name, testAccSSHPublicKey),
				),
			},
			{
				ResourceName:      "netactuate_sshkey.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_sshkey"] {
						_ = clients.V2.DeleteSSHKey(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_sshkey.test"),
			},
		},
	})
}

func TestAccNetactuateSSLCertificate_APIReadImportPlanAndOutOfBandDelete(t *testing.T) {
	name := testAccName("ssl")
	config := testAccSSLCertificateConfig(name)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSLCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_ssl_certificate", "netactuate_ssl_certificate.test"),
					testAccCheckSSLCertificateAPI("netactuate_ssl_certificate.test", name, "nah acceptance certificate"),
				),
			},
			{
				ResourceName:            "netactuate_ssl_certificate.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate", "private_key"},
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_ssl_certificate"] {
						_ = clients.V3.DeleteSSLCertificate(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_ssl_certificate.test"),
			},
		},
	})
}

func testAccCheckServerAPI(address, vpcAddress, hostname, plan string, locationID, imageID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		server, err := testAccClients().V2.GetServer(id)
		if err != nil {
			return err
		}
		if server.Name != hostname {
			return fmt.Errorf("API server hostname = %q, want %q", server.Name, hostname)
		}
		if !strings.EqualFold(server.Package, plan) {
			return fmt.Errorf("API server plan = %q, want %q", server.Package, plan)
		}
		if server.LocationID != locationID {
			return fmt.Errorf("API server location_id = %d, want %d", server.LocationID, locationID)
		}
		if server.OSID != imageID {
			return fmt.Errorf("API server image_id = %d, want %d", server.OSID, imageID)
		}
		if server.VpcID == nil || *server.VpcID != vpcID {
			if server.VpcID == nil {
				return fmt.Errorf("API server vpc_id is nil, want %d", vpcID)
			}
			return fmt.Errorf("API server vpc_id = %d, want %d", *server.VpcID, vpcID)
		}
		return nil
	}
}

func testAccCheckSSHKeyAPI(address, name, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		sshKey, err := testAccClients().V2.GetSSHKey(id)
		if err != nil {
			return err
		}
		if sshKey.Name != name {
			return fmt.Errorf("API SSH key name = %q, want %q", sshKey.Name, name)
		}
		if strings.TrimSpace(sshKey.Key) != key {
			return fmt.Errorf("API SSH key material did not match configured key")
		}
		return nil
	}
}

func testAccCheckServerDataSourceAPI(address, hostname, plan string, locationID, imageID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		server, err := testAccClients().V2.GetServer(id)
		if err != nil {
			return err
		}
		if server.Name != hostname {
			return fmt.Errorf("API server data source hostname = %q, want %q", server.Name, hostname)
		}
		if !strings.EqualFold(server.Package, plan) {
			return fmt.Errorf("API server data source plan = %q, want %q", server.Package, plan)
		}
		if server.LocationID != locationID {
			return fmt.Errorf("API server data source location_id = %d, want %d", server.LocationID, locationID)
		}
		if server.OSID != imageID {
			return fmt.Errorf("API server data source image_id = %d, want %d", server.OSID, imageID)
		}
		if rs.Primary.Attributes["hostname"] != server.Name {
			return fmt.Errorf("server data source hostname = %q, API has %q", rs.Primary.Attributes["hostname"], server.Name)
		}
		if rs.Primary.Attributes["package"] != server.Package {
			return fmt.Errorf("server data source package = %q, API has %q", rs.Primary.Attributes["package"], server.Package)
		}
		if rs.Primary.Attributes["location_id"] != strconv.Itoa(server.LocationID) {
			return fmt.Errorf("server data source location_id = %q, API has %d", rs.Primary.Attributes["location_id"], server.LocationID)
		}
		if rs.Primary.Attributes["image_id"] != strconv.Itoa(server.OSID) {
			return fmt.Errorf("server data source image_id = %q, API has %d", rs.Primary.Attributes["image_id"], server.OSID)
		}
		return nil
	}
}

func testAccCheckSSHKeyDataSourceAPI(address, name, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		sshKey, err := testAccClients().V2.GetSSHKey(id)
		if err != nil {
			return err
		}
		if sshKey.Name != name {
			return fmt.Errorf("API SSH key data source name = %q, want %q", sshKey.Name, name)
		}
		if strings.TrimSpace(sshKey.Key) != key {
			return fmt.Errorf("API SSH key data source material did not match configured key")
		}
		if rs.Primary.Attributes["name"] != sshKey.Name {
			return fmt.Errorf("SSH key data source name = %q, API has %q", rs.Primary.Attributes["name"], sshKey.Name)
		}
		if strings.TrimSpace(rs.Primary.Attributes["key"]) != strings.TrimSpace(sshKey.Key) {
			return fmt.Errorf("SSH key data source material did not match API")
		}
		return nil
	}
}

func testAccCheckSSLCertificateAPI(address, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		cert, err := testAccClients().V3.GetSSLCertificate(id)
		if err != nil {
			return err
		}
		if cert.Name != name {
			return fmt.Errorf("API SSL certificate name = %q, want %q", cert.Name, name)
		}
		if cert.Description != description {
			return fmt.Errorf("API SSL certificate description = %q, want %q", cert.Description, description)
		}
		if cert.SSLCertificateID != id {
			return fmt.Errorf("API SSL certificate id = %d, want %d", cert.SSLCertificateID, id)
		}
		return nil
	}
}

func testAccCheckSSHKeyDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_sshkey" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V2.GetSSHKey(id); err == nil {
			return fmt.Errorf("SSH key still exists: %d", id)
		} else if !gona.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckSSLCertificateDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_ssl_certificate" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetSSLCertificate(id); err == nil {
			return fmt.Errorf("SSL certificate still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccServerWithDataSourceConfigID(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccServerConfigID(name, locationID, imageID, plan, contractID, password) + `
data "netactuate_server" "test" {
  id = netactuate_server.test.id
}
`
}

func testAccSSHKeyConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_sshkey" "test" {
  name = %q
  key  = %q
}

data "netactuate_sshkey" "test" {
  id = netactuate_sshkey.test.id
}
`, name, testAccSSHPublicKey)
}

func testAccSSLCertificateConfig(name string) string {
	certificate, privateKey := testAccSelfSignedCertificate()
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_ssl_certificate" "test" {
  name        = %q
  description = "acceptance certificate"
  certificate = <<EOT
%sEOT
  private_key = <<EOT
%sEOT
}
`, name, certificate, privateKey)
}

// testAccSelfSignedCertificate returns a throwaway certificate and its key, generated for the
// run rather than embedded in the source. A committed private key is indistinguishable from a
// real one to anyone reading the repository, and to a secret scanner.
func testAccSelfSignedCertificate() (string, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("generate test key: %v", err))
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "acceptance.example.invalid"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(fmt.Sprintf("create test certificate: %v", err))
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(fmt.Sprintf("marshal test key: %v", err))
	}

	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certificate), string(privateKey)
}
