//go:build acctest

package netactuate

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateStorageBucketLifecycle(t *testing.T) {
	name := testAccName("storage-bucket-lifecycle")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	createConfig := testAccStorageBucketConfigID(name, locationID, 1, false)
	modifiedConfig := testAccStorageBucketConfigID(updatedName, locationID, 2, true)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBucketDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_storage_bucket.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_bucket", "netactuate_storage_bucket.test"),
			testAccCheckStorageBucketFromAPI("netactuate_storage_bucket.test", name, locationID, 1, false, false),
			testAccCheckStorageCredentialsPreserved("netactuate_storage_bucket.test", "access_key", "secret_key"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_bucket", "netactuate_storage_bucket.test"),
			testAccCheckStorageBucketFromAPI("netactuate_storage_bucket.test", updatedName, locationID, 2, false, true),
			testAccCheckStorageCredentialsPreserved("netactuate_storage_bucket.test", "access_key", "secret_key"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccStorageLifecycleImportStateID("netactuate_storage_bucket.test"),
			ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
		}),
	})
}

func TestAccNetactuateStorageBucketOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("storage-bucket-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBucketDestroy,
		Steps: []resource.TestStep{{
			Config: testAccStorageBucketConfigID(name, locationID, 1, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_storage_bucket", "netactuate_storage_bucket.test"),
				testAccCheckStorageBucketFromAPI("netactuate_storage_bucket.test", name, locationID, 1, false, false),
				testAccDeleteStorageBucketOutOfBand("netactuate_storage_bucket.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateStorageObjectStoreLifecycle(t *testing.T) {
	name := testAccName("storage-object-store-lifecycle")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	createConfig := testAccStorageObjectStoreConfigID(name, locationID, 1)
	modifiedConfig := testAccStorageObjectStoreConfigID(updatedName, locationID, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageObjectStoreDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_storage_object_store.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_object_store", "netactuate_storage_object_store.test"),
			testAccCheckStorageObjectStoreFromAPI("netactuate_storage_object_store.test", name, locationID, 1, false),
			testAccCheckStorageCredentialsPreserved("netactuate_storage_object_store.test", "access_key", "secret_key"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_object_store", "netactuate_storage_object_store.test"),
			testAccCheckStorageObjectStoreFromAPI("netactuate_storage_object_store.test", updatedName, locationID, 2, false),
			testAccCheckStorageCredentialsPreserved("netactuate_storage_object_store.test", "access_key", "secret_key"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccStorageLifecycleImportStateID("netactuate_storage_object_store.test"),
			ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
		}),
	})
}

func TestAccNetactuateStorageObjectStoreOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("storage-object-store-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageObjectStoreDestroy,
		Steps: []resource.TestStep{{
			Config: testAccStorageObjectStoreConfigID(name, locationID, 1),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_storage_object_store", "netactuate_storage_object_store.test"),
				testAccCheckStorageObjectStoreFromAPI("netactuate_storage_object_store.test", name, locationID, 1, false),
				testAccCheckStorageCredentialsPreserved("netactuate_storage_object_store.test", "access_key", "secret_key"),
				testAccDeleteStorageObjectStoreOutOfBand("netactuate_storage_object_store.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateStorageBlockNamespaceLifecycle(t *testing.T) {
	name := testAccName("storage-block-namespace-lifecycle")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	createConfig := testAccStorageBlockNamespaceConfigID(name, locationID, 1)
	modifiedConfig := testAccStorageBlockNamespaceConfigID(updatedName, locationID, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockNamespaceDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_storage_block_namespace.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
			testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", name, locationID, 1, false),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
			testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", updatedName, locationID, 2, false),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccStorageLifecycleImportStateID("netactuate_storage_block_namespace.test"),
			ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
		}),
	})
}

func TestAccNetactuateStorageBlockNamespaceOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("storage-block-namespace-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockNamespaceDestroy,
		Steps: []resource.TestStep{{
			Config: testAccStorageBlockNamespaceConfigID(name, locationID, 1),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
				testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", name, locationID, 1, false),
				testAccDeleteStorageBlockNamespaceOutOfBand("netactuate_storage_block_namespace.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateStorageBlockVolumeLifecycle(t *testing.T) {
	name := testAccName("storage-block-volume-lifecycle")
	nsName := name + "-ns"
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	createConfig := testAccStorageBlockVolumeWithNamespaceConfigID(nsName, name, locationID, 1)
	modifiedConfig := testAccStorageBlockVolumeWithNamespaceConfigID(nsName, updatedName, locationID, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockVolumeAndNamespaceDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_storage_block_volume.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
			testAccTrackFromState("netactuate_storage_block_volume", "netactuate_storage_block_volume.test"),
			testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", nsName, locationID, 1, false),
			testAccCheckStorageBlockVolumeFromAPI("netactuate_storage_block_volume.test", name, locationID, 1),
			testAccCheckStorageBlockVolumeAttachedToNamespaceAPI("netactuate_storage_block_volume.test", "netactuate_storage_block_namespace.test"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
			testAccTrackFromState("netactuate_storage_block_volume", "netactuate_storage_block_volume.test"),
			testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", nsName, locationID, 1, false),
			testAccCheckStorageBlockVolumeFromAPI("netactuate_storage_block_volume.test", updatedName, locationID, 2),
			testAccCheckStorageBlockVolumeAttachedToNamespaceAPI("netactuate_storage_block_volume.test", "netactuate_storage_block_namespace.test"),
		), testAccLifecycleOptions{
			ImportStateIdFunc: testAccStorageLifecycleImportStateID("netactuate_storage_block_volume.test"),
		}),
	})
}

func TestAccNetactuateStorageBlockVolumeOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("storage-block-volume-oob")
	nsName := name + "-ns"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockVolumeAndNamespaceDestroy,
		Steps: []resource.TestStep{{
			Config: testAccStorageBlockVolumeWithNamespaceConfigID(nsName, name, locationID, 1),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
				testAccTrackFromState("netactuate_storage_block_volume", "netactuate_storage_block_volume.test"),
				testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", nsName, locationID, 1, false),
				testAccCheckStorageBlockVolumeFromAPI("netactuate_storage_block_volume.test", name, locationID, 1),
				testAccCheckStorageBlockVolumeAttachedToNamespaceAPI("netactuate_storage_block_volume.test", "netactuate_storage_block_namespace.test"),
				testAccDeleteStorageBlockVolumeOutOfBand("netactuate_storage_block_volume.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccStorageLifecycleImportStateID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccDeleteStorageBucketOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteStorageBucket(id)
	}
}

func testAccDeleteStorageObjectStoreOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteStorageObjectStore(id)
	}
}

func testAccDeleteStorageBlockNamespaceOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteStorageBlockNamespace(id)
	}
}

func testAccDeleteStorageBlockVolumeOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteStorageBlockVolume(id)
	}
}
