package netactuate

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/netactuate/gona/gona"
)

// deleteStorageWithRetry runs a storage delete, waiting out the provisioning window.
//
// WHY. A storage object that is still provisioning refuses deletion with
//
//	400 "Your storage is not ready yet"
//
// A block volume answers this way for a short window after create. It is transient: the
// object becomes ready and the delete then succeeds. Failing immediately leaves a billable
// object behind.
//
// It is the same not-ready window that makes storage credentials appear only after about
// twenty seconds, and the same shape as a VPC refusing deletion while its VMs detach.
//
// Retried only on that specific message, so any other 400 still fails at once, and a
// not-found is treated as success because a deleted thing being gone is the goal.
func deleteStorageWithRetry(what string, del func() error) error {
	const retries = 18
	const interval = 10 * time.Second
	for attempt := 0; ; attempt++ {
		err := del()
		if err == nil {
			return nil
		}
		if gona.IsV3NotFound(err) || gona.IsNotFound(err) {
			log.Printf("[WARN] %s already deleted", what)
			return nil
		}
		// Two transient refusals, both of which resolve on their own:
		//
		//   "not ready yet"            the object is still provisioning
		//   "attached to Kubernetes"   an NKE storage addon still references this
		//                              namespace. Terraform destroys the addon first,
		//                              but the platform detaches asynchronously, so the
		//                              namespace delete can arrive before the detach has
		//                              landed.
		//
		// Anything else is a real error and fails at once.
		msg := err.Error()
		if !strings.Contains(msg, "not ready yet") && !strings.Contains(msg, "attached to Kubernetes") {
			return err
		}
		if attempt >= retries-1 {
			return err
		}
		log.Printf("[DEBUG] %s is still provisioning, retry %d/%d", what, attempt+1, retries)
		time.Sleep(interval)
	}
}

// waitForStorageCapacity waits until the API reports at least the capacity that was asked
// for.
//
// WHY, and why waiting for "ready" is not enough. A capacity change is accepted with a 200
// and applied asynchronously, and the object reports ready:true before the new capacity is
// visible: a block volume updated from 1 GB to 2 GB reports ready with totalGB still 1, so an
// update that waits only for ready writes the OLD capacity into state.
//
// `ready` on this platform means "the object exists and is usable", not "every field you
// just changed has landed". Credentials populate after ready too. So the only reliable test
// is the observable end state: poll the field that was changed until it is what was asked
// for.
func waitForStorageCapacity(what string, want int, get func() (int, error)) error {
	if want <= 0 {
		return nil
	}
	const retries = 18
	const interval = 10 * time.Second
	for attempt := 0; ; attempt++ {
		have, err := get()
		if err != nil {
			return err
		}
		if have >= want {
			return nil
		}
		if attempt >= retries-1 {
			return fmt.Errorf("%s capacity is still %d GB, wanted %d, after %s",
				what, have, want, time.Duration(retries)*interval)
		}
		log.Printf("[DEBUG] %s capacity %d GB, waiting for %d", what, have, want)
		time.Sleep(interval)
	}
}
