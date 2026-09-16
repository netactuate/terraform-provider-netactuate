//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateFirewallSetRule_VM_criteria1_2_3_4_6_7(t *testing.T) {
	name := testAccName("fw")
	initial := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	alphaMoved := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	betaMoved := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	gammaMoved := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	reordered := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	removed := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccCheckFirewallSetAPI("netactuate_firewall_set.test", name, true),
					testAccCheckListContainsResourceAttr("data.netactuate_firewall_sets.test", "firewall_sets", "id", "netactuate_firewall_set.test", "id"),
					// Matched on admin_comment, NOT on rule_id, and that is not a weaker
					// assertion, it is the correct one. Publishing a draft firewall set
					// replaces every rule id on the set, which resourceFirewallRuleRead
					// already works around with a priority fallback. Creating three rules
					// publishes three times, so alpha's and beta's recorded rule_ids are
					// stale by the time gamma is done. Asserting on them fails against a
					// listing that is entirely correct.
					testAccCheckListContainsResourceAttr("data.netactuate_firewall_rules.test", "rules", "admin_comment", "netactuate_firewall_rule.alpha", "admin_comment"),
					testAccCheckListContainsResourceAttr("data.netactuate_firewall_rules.test", "rules", "admin_comment", "netactuate_firewall_rule.beta", "admin_comment"),
					testAccCheckListContainsResourceAttr("data.netactuate_firewall_rules.test", "rules", "admin_comment", "netactuate_firewall_rule.gamma", "admin_comment"),
					testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
						{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
						{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
						{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					}, []string{"alpha", "beta", "gamma"}),
				),
			},
			{
				Config: alphaMoved,
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
					{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"beta", "gamma", "alpha"}),
			},
			{
				Config: betaMoved,
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
					{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"beta", "gamma", "alpha"}),
			},
			{
				Config: gammaMoved,
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
					{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					{Name: "alpha", Priority: 40, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"beta", "gamma", "alpha"}),
			},
			{
				Config: reordered,
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "beta", Priority: 10, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
					{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"beta", "gamma", "alpha"}),
			},
			{
				Config: removed,
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "gamma", Priority: 20, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"gamma", "alpha"}),
			},
			{
				ResourceName:      "netactuate_firewall_set.test",
				ImportState:       true,
				ImportStateVerify: true,
				// sync_after_publish is a write only instruction telling the API to push
				// the set to its VMs after publishing. It is not resource state, the API
				// never returns it, and import therefore cannot reproduce it. Same shape as
				// allow_downsize_reboot on the server resource.
				ImportStateVerifyIgnore: []string{"sync_after_publish"},
			},
			{
				ResourceName:      "netactuate_firewall_rule.alpha",
				ImportState:       true,
				ImportStateVerify: true,
				// sync_after_publish is a write only instruction telling the API to push
				// the set to its VMs after publishing. It is not resource state, the API
				// never returns it, and import therefore cannot reproduce it. Same shape as
				// allow_downsize_reboot on the server resource.
				ImportStateVerifyIgnore: []string{"sync_after_publish"},
			},
			{
				Config:   removed,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					_ = testAccDeleteFirewallRuleByAdminComment(name + " gamma")
					for id := range testAccCreated["netactuate_firewall_set"] {
						_ = clients.V2.SyncFirewallSetRules(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_firewall_rule.gamma"),
			},
			{
				Config: testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
					{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}),
				Check: testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "alpha", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"alpha"}),
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_firewall_set"] {
						_ = clients.V2.DeleteFirewallSet(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_firewall_set.test"),
					testAccCheckGone("netactuate_firewall_rule.alpha"),
				),
			},
		},
	})
}

func TestAccNetactuateFirewallSetVM_VM_criterion5(t *testing.T) {
	name := testAccName("fw-vm")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	withAttachment := testAccFirewallSetVMConfig(name, locationID, imageID, plan, contractID, password, true)
	withoutAttachment := testAccFirewallSetVMConfig(name, locationID, imageID, plan, contractID, password, false)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetVMAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAttachment,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckFirewallSetVMAPI("netactuate_firewall_set.test", "netactuate_server.test", true),
					testAccCheckListContainsResourceAttr("data.netactuate_firewall_set_vms.test", "vms", "mbpkgid", "netactuate_server.test", "id"),
					testAccCheckPluralListShape("data.netactuate_server_ips.test", "ipv4", map[string]testAccPluralFieldCheck{
						"id": positiveIntField,
						"ip": nonEmptyStringField,
					}),
				),
			},
			{
				Config: withoutAttachment,
				Check:  testAccCheckFirewallSetVMAPI("netactuate_firewall_set.test", "netactuate_server.test", false),
			},
		},
	})
}

type testAccFirewallRule struct {
	Name                 string
	Priority             int
	Action               string
	Protocol             string
	SourceNet            []string
	DestinationPortStart int
	DestinationPortEnd   int
}

func testAccCheckFirewallSetDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_firewall_set" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V2.GetFirewallSet(id); err == nil {
			return fmt.Errorf("firewall set still exists: %d", id)
		} else if !gona.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckFirewallSetVMAttachmentDestroy(s *terraform.State) error {
	if err := testAccCheckFirewallSetDestroy(s); err != nil {
		return err
	}
	return testAccCheckServerDestroy(s)
}

func testAccCheckFirewallSetAPI(address, name string, enabled bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		setID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		set, err := testAccClients().V2.GetFirewallSet(setID)
		if err != nil {
			return err
		}
		if set.Name != name {
			return fmt.Errorf("firewall set API name = %q, want %q", set.Name, name)
		}
		if set.Description != name {
			return fmt.Errorf("firewall set API description = %q, want %q", set.Description, name)
		}
		if set.Enabled != enabled {
			return fmt.Errorf("firewall set API enabled = %t, want %t", set.Enabled, enabled)
		}
		if set.DraftFirewallSetID != nil {
			return fmt.Errorf("firewall set %d still has draft %d after publish", setID, *set.DraftFirewallSetID)
		}
		return nil
	}
}

func testAccCheckFirewallRulesAPI(address string, want []testAccFirewallRule, wantOrder []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		setID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V2.GetFirewallRules(setID)
		if err != nil {
			return err
		}
		byName := map[string]gona.FirewallRule{}
		order := make([]string, 0, len(wantOrder))
		wantNames := map[string]struct{}{}
		for _, w := range want {
			wantNames[w.Name] = struct{}{}
		}
		for _, r := range rules {
			parts := strings.Split(r.AdminComment, " ")
			ruleName := parts[len(parts)-1]
			if _, ok := wantNames[ruleName]; !ok {
				continue
			}
			byName[ruleName] = r
			order = append(order, ruleName)
		}
		for _, w := range want {
			r, ok := byName[w.Name]
			if !ok {
				return fmt.Errorf("firewall rule %q not found on API in set %d", w.Name, setID)
			}
			if err := testAccFirewallRuleMatches(w, r); err != nil {
				return err
			}
		}
		if strings.Join(order, ",") != strings.Join(wantOrder, ",") {
			return fmt.Errorf("firewall API order = %v, want %v", order, wantOrder)
		}
		return nil
	}
}

func testAccFirewallRuleMatches(w testAccFirewallRule, r gona.FirewallRule) error {
	if r.IPVersion != "IPv4" || r.Direction != "IN" || r.Action != w.Action || !r.Enabled {
		return fmt.Errorf("firewall rule %q API base fields = ip_version:%q direction:%q action:%q enabled:%t", w.Name, r.IPVersion, r.Direction, r.Action, r.Enabled)
	}
	if r.RulePriority != w.Priority {
		return fmt.Errorf("firewall rule %q API priority = %d, want %d", w.Name, r.RulePriority, w.Priority)
	}
	if r.MatchCriteria == nil {
		return fmt.Errorf("firewall rule %q API match_criteria is nil", w.Name)
	}
	if r.MatchCriteria.Protocol != w.Protocol {
		return fmt.Errorf("firewall rule %q API protocol = %q, want %q", w.Name, r.MatchCriteria.Protocol, w.Protocol)
	}
	if !stringSliceEqual(r.MatchCriteria.SourceNet, w.SourceNet) {
		return fmt.Errorf("firewall rule %q API source_net = %v, want %v", w.Name, r.MatchCriteria.SourceNet, w.SourceNet)
	}
	if !intPtrValueEqual(r.MatchCriteria.DestinationPortStart, w.DestinationPortStart) || !intPtrValueEqual(r.MatchCriteria.DestinationPortEnd, w.DestinationPortEnd) {
		return fmt.Errorf("firewall rule %q API destination ports = %s-%s, want %d-%d", w.Name, ptrString(r.MatchCriteria.DestinationPortStart), ptrString(r.MatchCriteria.DestinationPortEnd), w.DestinationPortStart, w.DestinationPortEnd)
	}
	return nil
}

func testAccCheckFirewallSetVMAPI(setAddress, serverAddress string, wantAttached bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		setID, err := testAccID(s, setAddress)
		if err != nil {
			return err
		}
		mbpkgid, err := testAccID(s, serverAddress)
		if err != nil {
			return err
		}
		vms, err := testAccClients().V2.GetFirewallSetVMs(setID)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			if vm.Mbpkgid != mbpkgid {
				continue
			}
			if !wantAttached {
				return fmt.Errorf("VM %d is still attached to firewall set %d on API", mbpkgid, setID)
			}
			if vm.InterfaceID != 0 || vm.SetPriority != 7 {
				return fmt.Errorf("firewall VM API attachment = interface_id:%d set_priority:%d, want 0 and 7", vm.InterfaceID, vm.SetPriority)
			}
			return nil
		}
		if wantAttached {
			return fmt.Errorf("VM %d is not attached to firewall set %d on API", mbpkgid, setID)
		}
		return nil
	}
}

func testAccDeleteFirewallRuleByAdminComment(adminComment string) error {
	clients := testAccClients()
	for setID := range testAccCreated["netactuate_firewall_set"] {
		draft, err := clients.V2.CreateDraftFirewallSet(setID)
		if err != nil {
			return err
		}
		rules, err := clients.V2.GetFirewallRules(draft.ID)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if r.AdminComment != adminComment {
				continue
			}
			if err := clients.V2.DeleteFirewallRule(draft.ID, r.ID); err != nil {
				return err
			}
			if _, err := clients.V2.PublishDraftFirewallSet(draft.ID); err != nil {
				return err
			}
			return clients.V2.SyncFirewallSetRules(setID)
		}
		_ = clients.V2.DeleteDraftFirewallSet(draft.ID)
	}
	return nil
}

func testAccFirewallSetRulesConfig(name string, rules []testAccFirewallRule) string {
	var b strings.Builder
	b.WriteString(testAccProviderConfig())
	b.WriteString(fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = true
}
`, name, name))
	for _, r := range rules {
		b.WriteString(fmt.Sprintf(`
resource "netactuate_firewall_rule" %q {
  firewall_set_id        = netactuate_firewall_set.test.id
  ip_version             = "IPv4"
  direction              = "IN"
  action                 = %q
  protocol               = %q
  source_net             = %s
  destination_port_start = %d
  destination_port_end   = %d
  admin_comment          = %q
  enabled                = true
  rule_priority          = %d
  sync_after_publish     = true
}
`, r.Name, r.Action, r.Protocol, testAccStringList(r.SourceNet), r.DestinationPortStart, r.DestinationPortEnd, name+" "+r.Name, r.Priority))
	}
	b.WriteString(`
data "netactuate_firewall_sets" "test" {
  depends_on = [netactuate_firewall_set.test]
}
`)
	dependencies := []string{"netactuate_firewall_set.test"}
	for _, r := range rules {
		dependencies = append(dependencies, "netactuate_firewall_rule."+r.Name)
	}
	b.WriteString(fmt.Sprintf(`
data "netactuate_firewall_rules" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  depends_on      = [%s]
}
`, strings.Join(dependencies, ", ")))
	return b.String()
}

func testAccFirewallSetVMConfig(name string, locationID, imageID int, plan, contractID, password string, attach bool) string {
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = true
}

resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q

  lifecycle {
    ignore_changes = [password]
  }
}
`, name, name, name+".example.invalid", plan, locationID, imageID, password, contractID)
	if attach {
		config += `
resource "netactuate_firewall_set_vm" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  mbpkgid         = netactuate_server.test.id
  interface_id    = 0
  set_priority    = 7
}

data "netactuate_firewall_set_vms" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  depends_on      = [netactuate_firewall_set_vm.test]
}

data "netactuate_server_ips" "test" {
  mbpkgid    = netactuate_server.test.id
  depends_on = [netactuate_server.test]
}
`
	}
	return config
}

func testAccCheckListContainsResourceAttr(dataAddress, listAttr, field, resourceAddress, resourceAttr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		data, ok := s.RootModule().Resources[dataAddress]
		if !ok {
			return fmt.Errorf("not found: %s", dataAddress)
		}
		res, ok := s.RootModule().Resources[resourceAddress]
		if !ok {
			return fmt.Errorf("not found: %s", resourceAddress)
		}
		want := res.Primary.Attributes[resourceAttr]
		if want == "" {
			return fmt.Errorf("%s attribute %s is empty", resourceAddress, resourceAttr)
		}
		count, err := strconv.Atoi(data.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("%s state %s count is not an integer: %w", dataAddress, listAttr, err)
		}
		for i := 0; i < count; i++ {
			if data.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, field)] == want {
				return nil
			}
		}
		return fmt.Errorf("%s state %s has no item with %s = %s", dataAddress, listAttr, field, want)
	}
}

func testAccStringList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func intPtrValueEqual(p *int, want int) bool {
	return p != nil && *p == want
}

func ptrString(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return strconv.Itoa(*p)
}
