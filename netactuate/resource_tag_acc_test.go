//go:build acctest

package netactuate

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateTag_createUpdateImportAndOutOfBandDelete(t *testing.T) {
	name := testAccName("tag")
	updatedName := testAccName("tag-updated")
	description := "created by Terraform acceptance test"
	updatedDescription := "updated by Terraform acceptance test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTagsDestroyed(name, updatedName),
		Steps: []resource.TestStep{
			{
				Config: testAccTagConfig(name, description),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_tag.test", "name", name),
					resource.TestCheckResourceAttr("netactuate_tag.test", "description", description),
					resource.TestCheckResourceAttr("netactuate_tag.test", "is_default", "0"),
					resource.TestCheckResourceAttr("netactuate_tag.test", "is_favorite", "0"),
					resource.TestCheckResourceAttr("netactuate_tag.test", "is_locked", "0"),
					resource.TestCheckResourceAttr("netactuate_tag.test", "show_dashboard", "0"),
					resource.TestCheckResourceAttrSet("netactuate_tag.test", "icon"),
					testAccCheckTagAPI("netactuate_tag.test", name, description),
				),
			},
			{
				Config: testAccTagConfig(updatedName, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_tag.test", "name", updatedName),
					resource.TestCheckResourceAttr("netactuate_tag.test", "description", updatedDescription),
					testAccCheckTagAPI("netactuate_tag.test", updatedName, updatedDescription),
				),
			},
			{
				ResourceName:      "netactuate_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					id := testAccTagIDByExactName(t, updatedName)
					if err := testAccClients().V2.DeleteTag(id); err != nil {
						t.Fatalf("delete tag out of band: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_tag.test"),
			},
		},
	})
}

func TestAccNetactuateTagAssignment_assignImportAndRemove(t *testing.T) {
	tagName := testAccName("tag-assignment")
	serverName := testAccName("tag-assign") + ".natest.io"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")

	// The test builds its own server to tag rather than taking an id from the environment.
	// An earlier draft read NETACTUATE_ACC_TAGGABLE_SERVER_ID and tagged whatever it named,
	// which on this account would mean mutating one of 81 pre-existing production servers. The
	// rule is that we create and destroy only what we created, so the server is ours, carries
	// the nah- prefix and is torn down with the tag.
	//
	// No VPC, deliberately: a server outside one uses the virtual-server resource_name token,
	// and a VPC would add about seven minutes and real money for nothing this test asserts.

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
		CheckDestroy:      testAccCheckTagsDestroyed(tagName),
		Steps: []resource.TestStep{
			{
				Config: testAccTagAssignmentConfig(tagName, serverName, locationID, imageID, plan, contractID, password, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.netactuate_resource_tags.test", "resource_name", resourceNameVirtualServer),
					resource.TestCheckResourceAttrPair("data.netactuate_resource_tags.test", "identifier", "netactuate_server.test", "id"),
					testAccCheckResourceHasTag("netactuate_server.test", tagName, true),
				),
			},
			{
				ResourceName:      "netactuate_tag_assignment.test",
				ImportState:       true,
				ImportStateIdFunc: testAccTagAssignmentImportID("netactuate_tag_assignment.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccTagAssignmentConfig(tagName, serverName, locationID, imageID, plan, contractID, password, false),
				Check:  testAccCheckResourceHasTag("netactuate_server.test", tagName, false),
			},
		},
	})
}

func testAccTagConfig(name, description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_tag" "test" {
  name        = %q
  description = %q
}
`, name, description)
}

func testAccTagAssignmentConfig(name, serverName string, locationID, imageID int, plan, contractID, password string, assigned bool) string {
	assignment := ""
	dependsOn := ""
	if assigned {
		assignment = `
resource "netactuate_tag_assignment" "test" {
  tag_id        = netactuate_tag.test.id
  resource_name = "` + resourceNameVirtualServer + `"
  identifier    = tonumber(netactuate_server.test.id)
}
`
		dependsOn = `
  depends_on = [netactuate_tag_assignment.test]
`
	}

	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q
}

resource "netactuate_tag" "test" {
  name = %q
}
%s
data "netactuate_resource_tags" "test" {
  resource_name = %q
  identifier    = tonumber(netactuate_server.test.id)
%s}
`, serverName, plan, locationID, imageID, password, contractID, name, assignment, resourceNameVirtualServer, dependsOn)
}

func testAccCheckTagAPI(address, wantName, wantDescription string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		tag, err := testAccClients().V2.GetTag(id)
		if err != nil {
			return err
		}
		if tag.Name != wantName {
			return fmt.Errorf("tag %d name is %q, want %q", id, tag.Name, wantName)
		}
		if tag.Description != wantDescription {
			return fmt.Errorf("tag %d description is %q, want %q", id, tag.Description, wantDescription)
		}
		return nil
	}
}

// testAccCheckResourceHasTag reads the server id out of state rather than taking a literal,
// because the server is created by this test and its id is not known until apply.
func testAccCheckResourceHasTag(serverAddress, tagName string, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, err := testAccID(s, serverAddress)
		if err != nil {
			return err
		}
		tags, err := testAccClients().V2.GetResourceTags(resourceNameVirtualServer, resourceID)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			if tag.Name == tagName {
				if want {
					return nil
				}
				return fmt.Errorf("server %d still has tag %q", resourceID, tagName)
			}
		}
		if want {
			return fmt.Errorf("server %d does not have tag %q", resourceID, tagName)
		}
		return nil
	}
}

func testAccTagAssignmentImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return "", fmt.Errorf("not found: %s", address)
		}
		return rs.Primary.ID, nil
	}
}

func testAccTagIDByExactName(t *testing.T, name string) int {
	t.Helper()
	tags, err := testAccClients().V2.GetTags()
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	for _, tag := range tags {
		if tag.Name == name {
			return tag.ID
		}
	}
	t.Fatalf("tag %q not found", name)
	return 0
}

func testAccCheckTagsDestroyed(names ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tags, err := testAccClients().V2.GetTags()
		if err != nil {
			return fmt.Errorf("listing tags to clean up: %w", err)
		}

		wanted := make(map[string]bool, len(names))
		for _, name := range names {
			wanted[name] = true
		}

		var remaining []string
		for _, tag := range tags {
			if !wanted[tag.Name] {
				continue
			}
			if err := testAccClients().V2.DeleteTag(tag.ID); err != nil {
				remaining = append(remaining, fmt.Sprintf("%s (%d): %v", tag.Name, tag.ID, err))
			}
		}
		if len(remaining) > 0 {
			sort.Strings(remaining)
			return fmt.Errorf("test tag cleanup failed: %s", strings.Join(remaining, ", "))
		}
		return nil
	}
}
