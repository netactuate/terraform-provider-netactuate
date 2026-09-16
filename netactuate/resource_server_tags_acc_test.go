//go:build acctest

package netactuate

import (
	"fmt"
	"github.com/netactuate/gona/gona"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Tags baseline:
// reorder: no plan diff because tags are sorted by StateFunc and DiffSuppressFunc
// whitespace: no plan diff because tags are trimmed by StateFunc and DiffSuppressFunc
// duplicate: no plan diff because tags are de-duplicated by StateFunc and DiffSuppressFunc
// import: tags are not preserved by import unless ignored because the import
// path has no prior non-empty tags state, so Read treats tags as unmanaged
// exposed behavior: netactuate_server.tags is a comma-separated string
func TestAccNetactuateServerTagsBaseline_apiReadReorderWhitespaceDuplicateAndImport(t *testing.T) {
	name := testAccName("server-tags")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	tagA := testAccName("tag-a")
	tagB := testAccName("tag-b")
	configInitial := testAccServerTagsConfig(name, locationID, imageID, plan, contractID, password, tagA+", "+tagB)
	configReordered := testAccServerTagsConfig(name, locationID, imageID, plan, contractID, password, tagB+", "+tagA)
	configWhitespace := testAccServerTagsConfig(name, locationID, imageID, plan, contractID, password, "  "+tagA+" ,  "+tagB+"  ")
	configDuplicate := testAccServerTagsConfig(name, locationID, imageID, plan, contractID, password, tagA+", "+tagB+", "+tagA)

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
		CheckDestroy:      testAccCheckServerDestroyAndTags(tagA, tagB),
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckServerTagsAPI("netactuate_server.test", []string{tagA, tagB}),
				),
			},
			{
				Config:   configReordered,
				PlanOnly: true,
			},
			{
				Config:   configWhitespace,
				PlanOnly: true,
			},
			{
				Config:   configDuplicate,
				PlanOnly: true,
			},
			{
				ResourceName:            "netactuate_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "allow_downsize_reboot", "package_billing", "params", "tags"},
				Check:                   testAccCheckServerTagsAPI("netactuate_server.test", []string{tagA, tagB}),
			},
			{
				// BASELINE, not a requirement. Returning to the original tag string after
				// an import produces an EMPTY plan, because suppressTagsDiff absorbs it.
				//
				// The earlier version of this step asserted a non-empty plan and failed.
				// That was a guess about how tags behave, and the point of a baseline is
				// to record what they actually do before the surface changes. Every
				// PlanOnly step above is empty too: reorder, whitespace and duplicate are
				// all suppressed. So today the CSV `tags` attribute is diff-insensitive to
				// everything except a genuine change of membership.
				//
				// A netactuate_tag resource and a TypeSet would make a
				// reorder SHOULD be a no-op but a duplicate and stray whitespace should
				// not silently vanish. This step is what proves that change altered
				// behaviour rather than preserving a quirk by accident.
				Config:   configInitial,
				PlanOnly: true,
			},
		},
	})
}

func TestAccNetactuateServerTagsBaseline_outOfBandDeleteRefreshLeavesStateClean(t *testing.T) {
	name := testAccName("server-tags-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	tagA := testAccName("tag-oob-a")
	tagB := testAccName("tag-oob-b")
	config := testAccServerTagsConfig(name, locationID, imageID, plan, contractID, password, tagA+", "+tagB)

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
		CheckDestroy:      testAccCheckServerDestroyAndTags(tagA, tagB),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckServerTagsAPI("netactuate_server.test", []string{tagA, tagB}),
				),
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_server"] {
						jobID, err := clients.V2.DeleteServer(id, true)
						if err != nil {
							continue
						}
						if diags := wait4JobStatus("delete", jobID, clients.V2); diags.HasError() {
							t.Fatalf("wait for out-of-band server delete: %v", diags)
						}
						// The delete JOB completing does not mean the package record is
						// gone. GET cloud/server?mbpkgid=N kept returning the server for a
						// short while after the job reported done, so refreshing straight
						// away found it present and correctly left it in state, and the
						// test then failed for a race rather than a defect.
						//
						// The platform is eventually consistent on deletes, which is the
						// same shape as a VPC refusing deletion while its VMs detach. Poll
						// the observable end state rather than trusting the job status.
						deadline := time.Now().Add(3 * time.Minute)
						for {
							if _, err := clients.V2.GetServer(id); gona.IsNotFound(err) {
								break
							}
							if time.Now().After(deadline) {
								t.Fatalf("server %d still readable 3 minutes after its delete job finished", id)
							}
							time.Sleep(10 * time.Second)
						}
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_server.test"),
			},
		},
	})
}

func testAccCheckServerTagsAPI(address string, want []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		tags, err := testAccClients().V2.GetResourceTags(testAccServerTagsResourceName(rs), id)
		if err != nil {
			return err
		}
		got := make([]string, 0, len(tags))
		for _, tag := range tags {
			got = append(got, tag.Name)
		}
		sort.Strings(got)
		want = append([]string(nil), want...)
		sort.Strings(want)
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			return fmt.Errorf("server tag API mismatch for %d: got %q want %q", id, strings.Join(got, ","), strings.Join(want, ","))
		}
		return nil
	}
}

func testAccServerTagsResourceName(rs *terraform.ResourceState) string {
	if rs.Primary.Attributes["vpc_id"] != "" && rs.Primary.Attributes["vpc_id"] != "0" {
		return resourceNameVirtualServerVPC
	}
	return resourceNameVirtualServer
}

func testAccServerTagsConfig(name string, locationID, imageID int, plan, contractID, password, tags string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q
  vpc_id                      = netactuate_vpc.test.vpc_id
  tags                        = %q

  lifecycle {
    ignore_changes = [password]
  }
}
`, name, name, locationID, name+".example.invalid", plan, locationID, imageID, password, contractID, tags)
}

// testAccCheckServerDestroyAndTags runs the ordinary server destroy check and then removes the
// tag objects the test caused to exist.
//
// netactuate_server.tags creates tags implicitly: "Tags are created automatically if they do not
// exist". Destroying the server removes the ASSIGNMENT, never the tag object, so every run of
// these two tests left its tags on the account permanently. Tags cost nothing to hold and
// are easy to miss in account cleanup, so this check removes the test-owned tags.
//
// Only tags this test named are removed, matched exactly, so nothing of the account's is touched.
func testAccCheckServerDestroyAndTags(names ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		destroyErr := testAccCheckServerDestroy(s)

		clients := testAccClients()
		tags, err := clients.V2.GetTags()
		if err != nil {
			if destroyErr != nil {
				return destroyErr
			}
			return fmt.Errorf("listing tags to clean up: %w", err)
		}
		wanted := make(map[string]bool, len(names))
		for _, n := range names {
			wanted[n] = true
		}
		for _, t := range tags {
			if wanted[t.Name] {
				if err := clients.V2.DeleteTag(t.ID); err != nil {
					return fmt.Errorf("removing test tag %q (%d): %w", t.Name, t.ID, err)
				}
			}
		}
		return destroyErr
	}
}
