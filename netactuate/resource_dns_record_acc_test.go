//go:build acctest

package netactuate

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateDNSRecord_importUpdateAndOutOfBandDelete(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	configOne := testAccDNSRecordConfig(zoneName, "192.0.2.10", 3600)
	configTwo := testAccDNSRecordConfig(zoneName, "192.0.2.11", 7200)
	var recordID int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSRecordAndZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: configOne,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
					testAccTrackFromState("netactuate_dns_record", "netactuate_dns_record.test"),
					testAccRememberID("netactuate_dns_record.test", &recordID),
					testAccCheckDNSRecordFromAPI("netactuate_dns_record.test", zoneName, "www", "A", "192.0.2.10", 3600, 0),
					testAccCheckDNSRecordListedFromAPI("netactuate_dns_record.test", "A", "192.0.2.10", 3600),
				),
			},
			{
				ResourceName:      "netactuate_dns_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   configOne,
				PlanOnly: true,
			},
			{
				// SDKv2 TestStep does not expose plan actions here, so replacement is
				// rejected by applying the update, checking the ID, then requiring no diff.
				Config: configTwo,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSameID("netactuate_dns_record.test", &recordID),
					testAccCheckDNSRecordFromAPI("netactuate_dns_record.test", zoneName, "www", "A", "192.0.2.11", 7200, 0),
					testAccCheckDNSRecordListedFromAPI("netactuate_dns_record.test", "A", "192.0.2.11", 7200),
				),
			},
			{
				Config:   configTwo,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_dns_record"] {
						_ = clients.V2.DeleteRecord(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_dns_record.test"),
			},
		},
	})
}

func TestAccNetactuateDNSRecord_allSupportedTypesRoundTripFromAPI(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	config := testAccDNSRecordAllTypesConfig(zoneName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSRecordAndZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.ComposeTestCheckFunc(testAccDNSRecordAllTypesStepChecks(zoneName)...),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func testAccDNSRecordAllTypesStepChecks(zoneName string) []resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{
		testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
	}
	return append(checks, testAccDNSRecordAllTypesChecks(zoneName)...)
}

func testAccCheckDNSRecordAndZoneDestroy(s *terraform.State) error {
	if err := testAccCheckDNSRecordDestroy(s); err != nil {
		return err
	}
	return testAccCheckDNSZoneDestroy(s)
}

func testAccCheckDNSRecordDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_dns_record" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V2.GetRecord(id); err == nil {
			return fmt.Errorf("DNS record still exists: %d", id)
		} else if !gona.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccRememberID(address string, target *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		*target = id
		return nil
	}
}

func testAccCheckSameID(address string, target *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		if id != *target {
			return fmt.Errorf("%s ID = %d after update, want unchanged ID %d", address, id, *target)
		}
		return nil
	}
}

func testAccCheckDNSRecordFromAPI(address, zoneName, recordName, recordType, content string, ttl, priority int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		record, err := testAccClients().V2.GetRecord(id)
		if err != nil {
			return err
		}
		return testAccCheckDNSRecordAPIValue(address, *record, zoneName, recordName, recordType, content, ttl, priority)
	}
}

func testAccCheckDNSRecordListedFromAPI(address, recordType, content string, ttl int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zoneID, err := testAccStateInt(s, address, "zone_id")
		if err != nil {
			return err
		}
		records, err := testAccClients().V2.ListRecords(zoneID)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.ID != id {
				continue
			}
			if record.Type != recordType {
				return fmt.Errorf("%s API list type = %q, want %q", address, record.Type, recordType)
			}
			if record.Content != content {
				return fmt.Errorf("%s API list content = %q, want %q", address, record.Content, content)
			}
			if record.TTL.Int() != ttl {
				return fmt.Errorf("%s API list TTL = %q, want %d", address, string(record.TTL), ttl)
			}
			return nil
		}
		return fmt.Errorf("%s not found in API record list for zone %d", address, zoneID)
	}
}

func testAccCheckDNSRecordAPIValue(address string, record gona.DNSRecord, zoneName, recordName, recordType, content string, ttl, priority int) error {
	expectedFQDN := testAccDNSRecordFQDN(zoneName, recordName)
	if record.Name != expectedFQDN {
		return fmt.Errorf("%s API name = %q, want %q", address, record.Name, expectedFQDN)
	}
	if record.Type != recordType {
		return fmt.Errorf("%s API type = %q, want %q", address, record.Type, recordType)
	}
	if record.Content != content {
		return fmt.Errorf("%s API content = %q, want %q", address, record.Content, content)
	}
	if record.TTL.Int() != ttl {
		return fmt.Errorf("%s API TTL = %q, want %d", address, string(record.TTL), ttl)
	}
	if record.Priority != priority {
		return fmt.Errorf("%s API priority = %d, want %d", address, record.Priority, priority)
	}
	return nil
}

func testAccStateInt(s *terraform.State, address, key string) (int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, fmt.Errorf("not found: %s", address)
	}
	v, ok := rs.Primary.Attributes[key]
	if !ok {
		return 0, fmt.Errorf("%s missing state attribute %s", address, key)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid %s %q: %w", address, key, v, err)
	}
	return i, nil
}

func testAccDNSRecordConfig(zoneName, content string, ttl int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_dns_record" "test" {
  zone_id = netactuate_dns_zone.test.id
  name    = "www"
  type    = "A"
  content = %q
  ttl     = %d
}
`, zoneName, content, ttl)
}

type testAccDNSRecordCase struct {
	key      string
	name     string
	typ      string
	content  string
	ttl      int
	priority int
}

func testAccDNSRecordCases(zoneName string) []testAccDNSRecordCase {
	return []testAccDNSRecordCase{
		{key: "a", name: "a", typ: "A", content: "192.0.2.10", ttl: 300},
		{key: "aaaa", name: "aaaa", typ: "AAAA", content: "2001:db8::10", ttl: 301},
		{key: "cname", name: "alias", typ: "CNAME", content: "target." + zoneName, ttl: 302},
		{key: "mx", name: "@", typ: "MX", content: "mail." + zoneName, ttl: 303, priority: 10},
		{key: "ns", name: "delegated", typ: "NS", content: "ns1." + zoneName, ttl: 304},
		{key: "ptr", name: "ptr", typ: "PTR", content: "host." + zoneName, ttl: 305},
		{key: "spf", name: "spf", typ: "SPF", content: "v=spf1 -all", ttl: 306},
		// SRV with a plain name, because the RFC 2782 form cannot be created at all.
		// See TestAccNetactuateDNSRecord_srvRFC2782NameIsRejected below, which pins that
		// defect so this suite notices if the platform ever fixes it.
		{key: "srv", name: "sipplain", typ: "SRV", content: "0 5 5060 sip." + zoneName, ttl: 307},
		{key: "txt", name: "txt", typ: "TXT", content: "terraform-acceptance", ttl: 308},
	}
}

func testAccDNSRecordAllTypesChecks(zoneName string) []resource.TestCheckFunc {
	var checks []resource.TestCheckFunc
	for _, tc := range testAccDNSRecordCases(zoneName) {
		address := "netactuate_dns_record." + tc.key
		checks = append(checks,
			testAccTrackFromState("netactuate_dns_record", address),
			testAccCheckDNSRecordFromAPI(address, zoneName, tc.name, tc.typ, tc.content, tc.ttl, tc.priority),
		)
		// NS records are INVISIBLE to GET /dns/records/{zoneId}. An NS record creates
		// with 200, GET /dns/record/{id} returns it, and
		// the zone's record list omits it entirely while listing the A and MX records made
		// in the same breath. That is a platform defect, not a provider one, and it is
		// pinned by TestAccNetactuateDNSRecord_nsRecordIsMissingFromZoneList below.
		// Asserting the list here would just re-fail on the known defect every run.
		if tc.typ != "NS" {
			checks = append(checks, testAccCheckDNSRecordListedFromAPI(address, tc.typ, tc.content, tc.ttl))
		}
	}
	return checks
}

func testAccDNSRecordAllTypesConfig(zoneName string) string {
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}
`, zoneName)

	for _, tc := range testAccDNSRecordCases(zoneName) {
		// A bare label ("a"), a full FQDN ("a.zone") and the apex (the zone name
		// itself) are ALL accepted. The 422 this
		// test first hit was not about bare names at all, it was the SRV record below.
		config += fmt.Sprintf(`
resource "netactuate_dns_record" "%s" {
  zone_id = netactuate_dns_zone.test.id
  name    = %q
  type    = %q
  content = %q
  ttl     = %d
`, tc.key, tc.name, tc.typ, tc.content, tc.ttl)
		if tc.priority != 0 {
			config += fmt.Sprintf("  priority = %d\n", tc.priority)
		}
		config += "}\n"
	}
	return config
}

func testAccDNSRecordFQDN(zoneName, recordName string) string {
	if recordName == "@" {
		return zoneName
	}
	return recordName + "." + zoneName
}

// TestAccNetactuateDNSRecord_srvRFC2782NameIsRejected pins a PLATFORM defect, deliberately.
//
// Every real SRV record is named _service._proto.name per RFC 2782. The vAPI2 record
// validator rejects any name containing an underscore label for type SRV and type A with
// 422 validation_failed "The name must be a valid FQDN", while ACCEPTING the same shape for
// TXT (so _acme-challenge, and therefore ACME DNS-01, works).
//
// So SRV is advertised as a supported type by both the provider schema and its docs, and
// cannot hold a conformant SRV name. The API returns:
//
//	_sip._tcp   SRV -> 422        sipplain        SRV -> 200
//	_test       A   -> 422        _acme-challenge TXT -> 200
//
// This test asserts the CURRENT broken behaviour. It is expected to fail, loudly, on the day
// the platform fixes it, which is the point: that is when the suite should be updated and
// the guide can start recommending SRV.
func TestAccNetactuateDNSRecord_srvRFC2782NameIsRejected(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "srv" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_dns_record" "srv" {
  zone_id = netactuate_dns_zone.srv.id
  name    = "_sip._tcp"
  type    = "SRV"
  content = "0 5 5060 sip.%s"
  ttl     = 300
}
`, zoneName, zoneName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`must be a valid FQDN`),
			},
		},
	})
}

// TestAccNetactuateDNSRecord_nsRecordIsMissingFromZoneList pins a PLATFORM defect.
//
// An NS record can be created and fetched by id, but does not appear in the zone's record
// list. With three records created in one zone:
//
//	POST /dns/record  NS delegated -> 200 id=3212119
//	POST /dns/record  A  a         -> 200 id=3212122
//	POST /dns/record  MX @         -> 200 id=3212125
//	GET  /dns/records/{zone}       -> 2 records, the A and the MX. The NS is absent.
//	GET  /dns/record/3212119       -> 200, type NS. It exists.
//
// The Terraform consequence is not the resource itself, whose Read uses GetRecord by id and
// is therefore correct. It is that ANY list-driven path cannot see NS records: drift
// detection, import discovery, and any future records data source. resourceDNSZoneDelete
// also walks ListRecords to clear a zone before deleting it, and only gets away with
// skipping the NS record because the zone delete cascades server side.
//
// This test asserts the CURRENT broken behaviour and is expected to fail on the day the
// platform fixes it, which is exactly when this suite should be updated.
func TestAccNetactuateDNSRecord_nsRecordIsMissingFromZoneList(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "nslist" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_dns_record" "nslist" {
  zone_id = netactuate_dns_zone.nslist.id
  name    = "delegated"
  type    = "NS"
  content = "ns1.%s"
  ttl     = 300
}
`, zoneName, zoneName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					// It exists when fetched by id.
					testAccCheckDNSRecordFromAPI("netactuate_dns_record.nslist", zoneName, "delegated", "NS", "ns1."+zoneName, 300, 0),
					// And it is absent from the zone's list. If this ever fails, the
					// platform has fixed the defect: delete this test and restore the
					// NS case to the all-types list assertion.
					func(s *terraform.State) error {
						id, err := testAccID(s, "netactuate_dns_record.nslist")
						if err != nil {
							return err
						}
						zoneID, err := testAccStateInt(s, "netactuate_dns_record.nslist", "zone_id")
						if err != nil {
							return err
						}
						records, err := testAccClients().V2.ListRecords(zoneID)
						if err != nil {
							return err
						}
						for _, r := range records {
							if r.ID == id {
								return fmt.Errorf("NS record %d IS now listed for zone %d: the platform defect is fixed, update this suite", id, zoneID)
							}
						}
						return nil
					},
				),
			},
		},
	})
}
