//go:build acctest

package netactuate

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type testAccLifecycleOptions struct {
	ImportStateIdFunc       resource.ImportStateIdFunc
	ImportStateVerifyIgnore []string
}

func testAccLifecycleSteps(createConfig, modifiedConfig, resourceAddress string, checkCreate, checkModified resource.TestCheckFunc, opts testAccLifecycleOptions) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: createConfig,
			Check:  checkCreate,
		},
		{
			Config:             createConfig,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			Config: modifiedConfig,
			Check:  checkModified,
		},
		{
			Config:             modifiedConfig,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			ResourceName:            resourceAddress,
			ImportState:             true,
			ImportStateIdFunc:       opts.ImportStateIdFunc,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: opts.ImportStateVerifyIgnore,
		},
		{
			Config:             modifiedConfig,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	}
}

func testAccStateID(s *terraform.State, address string) (string, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return "", fmt.Errorf("not found: %s", address)
	}
	if rs.Primary == nil || rs.Primary.ID == "" {
		return "", fmt.Errorf("%s has no primary ID", address)
	}
	return rs.Primary.ID, nil
}
