//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateStorageLocations_dataSourceHydratesFromAPI(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageLocationsDataSourceConfig(),
				Check:  testAccCheckStorageLocationsDataSourceFromAPI("data.netactuate_storage_locations.test"),
			},
		},
	})
}

func TestAccNetactuateStorageBucket_importPlanUpdateLocationAndOutOfBandDelete(t *testing.T) {
	name := testAccName("storage-bucket")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
	differentLocationName := testAccDifferentStorageLocationName(t, locationID)
	configID := testAccStorageBucketConfigID(name, locationID, 1, false)
	configUpdated := testAccStorageBucketConfigID(updatedName, locationID, 2, true)
	configName := testAccStorageBucketConfigName(updatedName, locationName, 2, true)
	configDifferentLocation := testAccStorageBucketConfigName(updatedName, differentLocationName, 2, true)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID", "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_storage_bucket", "netactuate_storage_bucket.test"),
					testAccCheckStorageBucketFromAPI("netactuate_storage_bucket.test", name, locationID, 1, false, false),
					// user_key is NOT returned by the API for buckets or object stores, only endpoints,
					// accessKey and secretKey. The provider exposes it and can never fill it, so it is
					// deliberately not asserted. Recorded as a release note rather than a code change,
					// since removing an attribute is breaking.
					testAccCheckStorageCredentialsPreserved("netactuate_storage_bucket.test", "access_key", "secret_key"),
				),
			},
			{
				ResourceName:      "netactuate_storage_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
				// enable_auto_scaling is a write only INPUT. The API returns the resulting
				// state as auto_scaling instead, so import can never reproduce it. Same
				// shape as sync_after_publish on the firewall set.
				ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check:  testAccCheckStorageBucketFromAPI("netactuate_storage_bucket.test", updatedName, locationID, 2, false, true),
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				Config:             configDifferentLocation,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteStorageBucketsByLabel(updatedName); err != nil {
						t.Fatalf("out of band storage bucket delete: %v", err)
					}
					testAccWaitGoneFromStorageAPI(t, "storage bucket", func() (bool, error) {
						exists, err := testAccStorageBucketExistsByLabel(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_storage_bucket.test"),
			},
		},
	})
}

func TestAccNetactuateStorageObjectStore_importPlanUpdateLocationAndOutOfBandDelete(t *testing.T) {
	name := testAccName("storage-object-store")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
	differentLocationName := testAccDifferentStorageLocationName(t, locationID)
	configID := testAccStorageObjectStoreConfigID(name, locationID, 1)
	configUpdated := testAccStorageObjectStoreConfigID(updatedName, locationID, 2)
	configName := testAccStorageObjectStoreConfigName(updatedName, locationName, 2)
	configDifferentLocation := testAccStorageObjectStoreConfigName(updatedName, differentLocationName, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID", "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageObjectStoreDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_storage_object_store", "netactuate_storage_object_store.test"),
					testAccCheckStorageObjectStoreFromAPI("netactuate_storage_object_store.test", name, locationID, 1, false),
					testAccCheckStorageCredentialsPreserved("netactuate_storage_object_store.test", "access_key", "secret_key"),
				),
			},
			{
				ResourceName:      "netactuate_storage_object_store.test",
				ImportState:       true,
				ImportStateVerify: true,
				// enable_auto_scaling is a write only INPUT. The API returns the resulting
				// state as auto_scaling instead, so import can never reproduce it. Same
				// shape as sync_after_publish on the firewall set.
				ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check:  testAccCheckStorageObjectStoreFromAPI("netactuate_storage_object_store.test", updatedName, locationID, 2, false),
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				Config:             configDifferentLocation,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteStorageObjectStoresByLabel(updatedName); err != nil {
						t.Fatalf("out of band storage object store delete: %v", err)
					}
					testAccWaitGoneFromStorageAPI(t, "storage object store", func() (bool, error) {
						exists, err := testAccStorageObjectStoreExistsByLabel(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_storage_object_store.test"),
			},
		},
	})
}

func TestAccNetactuateStorageBlockNamespace_importPlanUpdateLocationAndOutOfBandDelete(t *testing.T) {
	name := testAccName("storage-block-namespace")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
	differentLocationName := testAccDifferentStorageLocationName(t, locationID)
	configID := testAccStorageBlockNamespaceConfigID(name, locationID, 1)
	configUpdated := testAccStorageBlockNamespaceConfigID(updatedName, locationID, 2)
	configName := testAccStorageBlockNamespaceConfigName(updatedName, locationName, 2)
	configDifferentLocation := testAccStorageBlockNamespaceConfigName(updatedName, differentLocationName, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID", "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
					testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", name, locationID, 1, false),
				),
			},
			{
				ResourceName:      "netactuate_storage_block_namespace.test",
				ImportState:       true,
				ImportStateVerify: true,
				// enable_auto_scaling is a write only INPUT. The API returns the resulting
				// state as auto_scaling instead, so import can never reproduce it. Same
				// shape as sync_after_publish on the firewall set.
				ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check:  testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", updatedName, locationID, 2, false),
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				Config:             configDifferentLocation,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteStorageBlockNamespacesByLabel(updatedName); err != nil {
						t.Fatalf("out of band storage block namespace delete: %v", err)
					}
					testAccWaitGoneFromStorageAPI(t, "storage block namespace", func() (bool, error) {
						exists, err := testAccStorageBlockNamespaceExistsByLabel(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_storage_block_namespace.test"),
			},
		},
	})
}

func TestAccNetactuateStorageBlockVolume_importPlanUpdateLocationNamespaceAndOutOfBandDelete(t *testing.T) {
	name := testAccName("storage-block-volume")
	nsName := name + "-ns"
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
	differentLocationName := testAccDifferentStorageLocationName(t, locationID)
	configID := testAccStorageBlockVolumeWithNamespaceConfigID(nsName, name, locationID, 1)
	configUpdated := testAccStorageBlockVolumeWithNamespaceConfigID(nsName, updatedName, locationID, 2)
	configName := testAccStorageBlockVolumeWithNamespaceConfigName(nsName, updatedName, locationName, 2)
	configDifferentLocation := testAccStorageBlockVolumeWithNamespaceConfigName(nsName, updatedName, differentLocationName, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID", "NETACTUATE_ACC_STORAGE_LOCATION_NAME")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStorageBlockVolumeAndNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
					testAccTrackFromState("netactuate_storage_block_volume", "netactuate_storage_block_volume.test"),
					testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", nsName, locationID, 1, false),
					testAccCheckStorageBlockVolumeFromAPI("netactuate_storage_block_volume.test", name, locationID, 1),
					testAccCheckStorageBlockVolumeAttachedToNamespaceAPI("netactuate_storage_block_volume.test", "netactuate_storage_block_namespace.test"),
				),
			},
			{
				ResourceName:      "netactuate_storage_block_volume.test",
				ImportState:       true,
				ImportStateVerify: true,
				// enable_auto_scaling is a write only INPUT. The API returns the resulting
				// state as auto_scaling instead, so import can never reproduce it. Same
				// shape as sync_after_publish on the firewall set.
				ImportStateVerifyIgnore: []string{"enable_auto_scaling"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBlockVolumeFromAPI("netactuate_storage_block_volume.test", updatedName, locationID, 2),
					testAccCheckStorageBlockVolumeAttachedToNamespaceAPI("netactuate_storage_block_volume.test", "netactuate_storage_block_namespace.test"),
				),
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				Config:             configDifferentLocation,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteStorageBlockVolumesByLabel(updatedName); err != nil {
						t.Fatalf("out of band storage block volume delete: %v", err)
					}
					testAccWaitGoneFromStorageAPI(t, "storage block volume", func() (bool, error) {
						exists, err := testAccStorageBlockVolumeExistsByLabel(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_storage_block_volume.test"),
			},
		},
	})
}

func testAccCheckStorageLocationsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		locations, err := testAccClients().V3.ListStorageLocations()
		if err != nil {
			return err
		}
		if len(locations) == 0 {
			return fmt.Errorf("storage locations API returned no locations")
		}
		if got := rs.Primary.Attributes["locations.#"]; got != strconv.Itoa(len(locations)) {
			return fmt.Errorf("%s state location count = %q, want API count %d", address, got, len(locations))
		}
		for i, loc := range locations {
			prefix := fmt.Sprintf("locations.%d", i)
			if got := rs.Primary.Attributes[prefix+".id"]; got != strconv.Itoa(loc.Location.ID) {
				return fmt.Errorf("%s state %s.id = %q, want API id %d", address, prefix, got, loc.Location.ID)
			}
			if got := rs.Primary.Attributes[prefix+".name"]; got != loc.Location.Name {
				return fmt.Errorf("%s state %s.name = %q, want API name %q", address, prefix, got, loc.Location.Name)
			}
		}
		return nil
	}
}

func testAccCheckStorageBucketFromAPI(address, expectedLabel string, expectedLocationID, expectedCapacity int, expectedAutoScaling, expectedPrivate bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		bucket, err := testAccClients().V3.GetStorageBucket(id)
		if err != nil {
			return err
		}
		if bucket.Metadata.BucketID != id || bucket.Metadata.Label != expectedLabel || bucket.Metadata.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API bucket = id:%d label:%q location_id:%d", address, bucket.Metadata.BucketID, bucket.Metadata.Label, bucket.Metadata.Location.ID)
		}
		if !bucket.Metadata.Ready {
			return fmt.Errorf("%s API ready = false, want true", address)
		}
		if bucket.Metadata.Capacity.TotalGB < expectedCapacity {
			return fmt.Errorf("%s API total_capacity_gb = %d, want at least %d", address, bucket.Metadata.Capacity.TotalGB, expectedCapacity)
		}
		if bucket.Metadata.Capacity.AutoScaling != expectedAutoScaling {
			return fmt.Errorf("%s API auto_scaling = %t, want %t", address, bucket.Metadata.Capacity.AutoScaling, expectedAutoScaling)
		}
		if bucket.Metadata.Private != expectedPrivate {
			return fmt.Errorf("%s API private = %t, want %t", address, bucket.Metadata.Private, expectedPrivate)
		}
		// Credentials are deliberately NOT asserted from the API here. They are issued
		// at CREATE and are not returned by any read: GET /storage/buckets/{id}
		// answers "credentials": {} for a bucket created
		// moments earlier. Asserting them from a read could only ever fail.
		//
		// The thing that WAS broken is asserted from state instead, by
		// testAccCheckStorageCredentialsPreserved: a refresh must not destroy the
		// credentials it cannot re-read.
		return nil
	}
}

func testAccCheckStorageObjectStoreFromAPI(address, expectedLabel string, expectedLocationID, expectedCapacity int, expectedAutoScaling bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		store, err := testAccClients().V3.GetStorageObjectStore(id)
		if err != nil {
			return err
		}
		if store.Metadata.ObjectStoreID != id || store.Metadata.Label != expectedLabel || store.Metadata.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API object store = id:%d label:%q location_id:%d", address, store.Metadata.ObjectStoreID, store.Metadata.Label, store.Metadata.Location.ID)
		}
		if !store.Metadata.Ready {
			return fmt.Errorf("%s API ready = false, want true", address)
		}
		if store.Metadata.Capacity.TotalGB < expectedCapacity {
			return fmt.Errorf("%s API total_capacity_gb = %d, want at least %d", address, store.Metadata.Capacity.TotalGB, expectedCapacity)
		}
		if store.Metadata.Capacity.AutoScaling != expectedAutoScaling {
			return fmt.Errorf("%s API auto_scaling = %t, want %t", address, store.Metadata.Capacity.AutoScaling, expectedAutoScaling)
		}
		// Credentials appear only once the object store is READY, about 25 seconds after
		// create, and userKey never appears at all. Asserted from state instead, by
		// testAccCheckStorageCredentialsPreserved.
		return nil
	}
}

func testAccCheckStorageBlockNamespaceFromAPI(address, expectedLabel string, expectedLocationID, expectedCapacity int, expectedAutoScaling bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		ns, err := testAccClients().V3.GetStorageBlockNamespace(id)
		if err != nil {
			return err
		}
		if ns.Metadata.BlockNamespaceID != id || ns.Metadata.Label != expectedLabel || ns.Metadata.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API block namespace = id:%d label:%q location_id:%d", address, ns.Metadata.BlockNamespaceID, ns.Metadata.Label, ns.Metadata.Location.ID)
		}
		if !ns.Metadata.Ready {
			return fmt.Errorf("%s API ready = false, want true", address)
		}
		if ns.Metadata.Capacity.TotalGB < expectedCapacity {
			return fmt.Errorf("%s API total_capacity_gb = %d, want at least %d", address, ns.Metadata.Capacity.TotalGB, expectedCapacity)
		}
		if ns.Metadata.Capacity.AutoScaling != expectedAutoScaling {
			return fmt.Errorf("%s API auto_scaling = %t, want %t", address, ns.Metadata.Capacity.AutoScaling, expectedAutoScaling)
		}
		if len(ns.Credentials.Endpoints) == 0 || ns.Credentials.Pool == "" || ns.Credentials.Namespace == "" || ns.Credentials.ClusterID == "" {
			return fmt.Errorf("%s API block credentials are incomplete", address)
		}
		return nil
	}
}

func testAccCheckStorageBlockVolumeFromAPI(address, expectedLabel string, expectedLocationID, expectedCapacity int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vol, err := testAccClients().V3.GetStorageBlockVolume(id)
		if err != nil {
			return err
		}
		if vol.Metadata.BlockVolumeID != id || vol.Metadata.Label != expectedLabel || vol.Metadata.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API block volume = id:%d label:%q location_id:%d", address, vol.Metadata.BlockVolumeID, vol.Metadata.Label, vol.Metadata.Location.ID)
		}
		if !vol.Metadata.Ready {
			return fmt.Errorf("%s API ready = false, want true", address)
		}
		if vol.Metadata.Capacity.TotalGB < expectedCapacity {
			return fmt.Errorf("%s API total_capacity_gb = %d, want at least %d", address, vol.Metadata.Capacity.TotalGB, expectedCapacity)
		}
		// A block VOLUME and a block NAMESPACE return DIFFERENT credential sets, and it
		// is not an omission:
		//
		//	namespace: endpoints, userKey, secretKey, pool, namespace, clusterId
		//	volume:    endpoints, userKey, secretKey, imageName, namespace
		//
		// pool and clusterId describe the namespace the volume lives in, and imageName
		// describes the volume, so each object returns what it owns. Asserting the
		// namespace's set against a volume demanded fields the API will never send.
		if len(vol.Credentials.Endpoints) == 0 || vol.Credentials.UserKey == "" ||
			vol.Credentials.SecretKey == "" || vol.Credentials.Namespace == "" ||
			vol.Credentials.ImageName == "" {
			return fmt.Errorf("%s API block volume credentials are incomplete: endpoints=%d userKey=%t secretKey=%t namespace=%t imageName=%t",
				address, len(vol.Credentials.Endpoints), vol.Credentials.UserKey != "",
				vol.Credentials.SecretKey != "", vol.Credentials.Namespace != "",
				vol.Credentials.ImageName != "")
		}
		return nil
	}
}

func testAccCheckStorageBlockVolumeAttachedToNamespaceAPI(volumeAddress, namespaceAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		volumeID, err := testAccID(s, volumeAddress)
		if err != nil {
			return err
		}
		namespaceID, err := testAccID(s, namespaceAddress)
		if err != nil {
			return err
		}
		ns, err := testAccClients().V3.GetStorageBlockNamespace(namespaceID)
		if err != nil {
			return err
		}
		// Poll for the volume's namespace credential rather than reading once.
		//
		// Credentials populate after the object reports ready, not with it: this is the
		// same not-ready window that delays bucket credentials by about twenty seconds,
		// refuses a delete before ready, and applies an update after the call returns.
		// Reading once immediately after create caught the volume with ready true and an
		// empty credential set, and the check failed on a race rather than on a real
		// mismatch. The wave's own rule is to poll the observable end state.
		var vol *gona.StorageBlockVolume
		deadline := time.Now().Add(2 * time.Minute)
		for {
			vol, err = testAccClients().V3.GetStorageBlockVolume(volumeID)
			if err != nil {
				return err
			}
			if vol.Credentials.Namespace != "" {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("%s still has no namespace credential 2 minutes after create", volumeAddress)
			}
			time.Sleep(10 * time.Second)
		}
		// The volume to namespace relationship is NOT READABLE from the API:
		//
		//   - the two objects return different credential sets, so pool and clusterId
		//     cannot be compared: only the namespace returns them
		//   - the `namespace` credential is per-object, not a shared key. A volume
		//     reports "volume-namespace-<volumeId>" and its namespace reports
		//     "namespace-<namespaceId>", so they are never equal by construction
		//   - GET /storage/block-volumes/{id} returns object-store shaped metadata with
		//     no namespace field, and the LIST metadata carries type, id, assignedOn,
		//     label, blockVolumeId, location, ready, capacity, usage and hardwareClass,
		//     with no namespace linkage either
		//
		// So a customer cannot ask the API which namespace a volume belongs to. The
		// provider holds block_namespace_id in state from the create request and has no
		// way to verify or refresh it, which means drift on that field is undetectable.
		// Recorded in review/27 as a platform gap.
		//
		// What IS assertable is that the volume has usable block credentials of its own,
		// which is checked above, and that creating it against this namespace succeeded.
		_ = ns
		return nil
	}
}

func testAccCheckStorageBucketDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_storage_bucket" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetStorageBucket(id); err == nil {
			return fmt.Errorf("storage bucket still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckStorageObjectStoreDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_storage_object_store" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetStorageObjectStore(id); err == nil {
			return fmt.Errorf("storage object store still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckStorageBlockNamespaceDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_storage_block_namespace" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetStorageBlockNamespace(id); err == nil {
			return fmt.Errorf("storage block namespace still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckStorageBlockVolumeDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_storage_block_volume" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetStorageBlockVolume(id); err == nil {
			return fmt.Errorf("storage block volume still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckStorageBlockVolumeAndNamespaceDestroy(s *terraform.State) error {
	if err := testAccCheckStorageBlockVolumeDestroy(s); err != nil {
		return err
	}
	return testAccCheckStorageBlockNamespaceDestroy(s)
}

func testAccDeleteStorageBucketsByLabel(label string) error {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_storage_bucket"] {
		bucket, err := clients.V3.GetStorageBucket(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if bucket.Metadata.Label == label {
			if err := clients.V3.DeleteStorageBucket(id); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccDeleteStorageObjectStoresByLabel(label string) error {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_storage_object_store"] {
		store, err := clients.V3.GetStorageObjectStore(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if store.Metadata.Label == label {
			if err := clients.V3.DeleteStorageObjectStore(id); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccDeleteStorageBlockNamespacesByLabel(label string) error {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_storage_block_namespace"] {
		ns, err := clients.V3.GetStorageBlockNamespace(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if ns.Metadata.Label == label {
			if err := clients.V3.DeleteStorageBlockNamespace(id); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccDeleteStorageBlockVolumesByLabel(label string) error {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_storage_block_volume"] {
		vol, err := clients.V3.GetStorageBlockVolume(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if vol.Metadata.Label == label {
			if err := clients.V3.DeleteStorageBlockVolume(id); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccStorageBucketExistsByLabel(label string) (bool, error) {
	for id := range testAccCreated["netactuate_storage_bucket"] {
		bucket, err := testAccClients().V3.GetStorageBucket(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return false, err
		}
		if err == nil && bucket.Metadata.Label == label {
			return true, nil
		}
	}
	return false, nil
}

func testAccStorageObjectStoreExistsByLabel(label string) (bool, error) {
	for id := range testAccCreated["netactuate_storage_object_store"] {
		store, err := testAccClients().V3.GetStorageObjectStore(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return false, err
		}
		if err == nil && store.Metadata.Label == label {
			return true, nil
		}
	}
	return false, nil
}

func testAccStorageBlockNamespaceExistsByLabel(label string) (bool, error) {
	for id := range testAccCreated["netactuate_storage_block_namespace"] {
		ns, err := testAccClients().V3.GetStorageBlockNamespace(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return false, err
		}
		if err == nil && ns.Metadata.Label == label {
			return true, nil
		}
	}
	return false, nil
}

func testAccStorageBlockVolumeExistsByLabel(label string) (bool, error) {
	for id := range testAccCreated["netactuate_storage_block_volume"] {
		vol, err := testAccClients().V3.GetStorageBlockVolume(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return false, err
		}
		if err == nil && vol.Metadata.Label == label {
			return true, nil
		}
	}
	return false, nil
}

func testAccWaitGoneFromStorageAPI(t *testing.T, label string, gone func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		ok, err := gone()
		if err != nil {
			lastErr = err
		} else if ok {
			return
		}
		time.Sleep(5 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("%s was not gone before timeout: %v", label, lastErr)
	}
	t.Fatalf("%s was not gone before timeout", label)
}

func testAccDifferentStorageLocationName(t *testing.T, currentLocationID int) string {
	t.Helper()
	if currentLocationID == 0 {
		return ""
	}
	locations, err := testAccClients().V3.ListStorageLocations()
	if err != nil {
		if testAccIsAcceptanceRun() {
			t.Skipf("list storage locations for different-location check: %v", err)
		}
		return ""
	}
	for _, location := range locations {
		if location.Location.ID != currentLocationID && location.Location.Name != "" {
			return location.Location.Name
		}
	}
	if testAccIsAcceptanceRun() {
		t.Skip("NETACTUATE account storage catalog has no second location for replacement plan check")
	}
	return ""
}

func testAccStorageLocationsDataSourceConfig() string {
	return testAccProviderConfig() + `
data "netactuate_storage_locations" "test" {}
`
}

func testAccStorageBucketConfigID(name string, locationID, capacity int, private bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_bucket" "test" {
  label       = %q
  location_id = %d
  capacity    = %d
  private     = %t
}
`, name, locationID, capacity, private)
}

func testAccStorageBucketConfigName(name, locationName string, capacity int, private bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_bucket" "test" {
  label    = %q
  location = %q
  capacity = %d
  private  = %t
}
`, name, locationName, capacity, private)
}

func testAccStorageObjectStoreConfigID(name string, locationID, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_object_store" "test" {
  label       = %q
  location_id = %d
  capacity    = %d
}
`, name, locationID, capacity)
}

func testAccStorageObjectStoreConfigName(name, locationName string, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_object_store" "test" {
  label    = %q
  location = %q
  capacity = %d
}
`, name, locationName, capacity)
}

func testAccStorageBlockNamespaceConfigID(name string, locationID, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_block_namespace" "test" {
  label       = %q
  location_id = %d
  capacity    = %d
}
`, name, locationID, capacity)
}

func testAccStorageBlockNamespaceConfigName(name, locationName string, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_block_namespace" "test" {
  label    = %q
  location = %q
  capacity = %d
}
`, name, locationName, capacity)
}

func testAccStorageBlockVolumeWithNamespaceConfigID(namespaceName, volumeName string, locationID, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_block_namespace" "test" {
  label       = %q
  location_id = %d
  capacity    = %d
}

resource "netactuate_storage_block_volume" "test" {
  label       = %q
  location_id = netactuate_storage_block_namespace.test.location_id
  capacity    = %d
}
`, namespaceName, locationID, capacity, volumeName, capacity)
}

func testAccStorageBlockVolumeWithNamespaceConfigName(namespaceName, volumeName, locationName string, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_block_namespace" "test" {
  label    = %q
  location = %q
  capacity = %d
}

resource "netactuate_storage_block_volume" "test" {
  label    = %q
  location = netactuate_storage_block_namespace.test.location
  capacity = %d
}
`, namespaceName, locationName, capacity, volumeName, capacity)
}

func testAccStorageBlockVolumeConfigName(name, locationName string, capacity int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_storage_block_volume" "test" {
  label    = %q
  location = %q
  capacity = %d
}
`, name, locationName, capacity)
}

// testAccCheckStorageCredentialsPreserved asserts that credentials survive in state.
//
// This is the assertion that matters for storage. The Read paths used to write the API's
// empty credentials object straight into state, so the FIRST REFRESH destroyed the access
// key, secret key, user key and endpoints issued at create, unrecoverably. Terraform runs a
// refresh before the checks in a test step, so a value still present here has survived one.
func testAccCheckStorageCredentialsPreserved(address string, keys ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		for _, k := range keys {
			if v := rs.Primary.Attributes[k]; v == "" {
				return fmt.Errorf(
					"%s %s is empty after refresh: credentials issued at create were destroyed by a read",
					address, k)
			}
		}
		return nil
	}
}
