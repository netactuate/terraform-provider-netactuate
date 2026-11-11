package netactuate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

// workerIPsStep is one answer from the fake API: the cluster's node counters and the workers
// (name -> address, "" for a worker with no reservation yet).
type workerIPsStep struct {
	minimum, total, deleting, outdated int
	maximum                            *int
	vpcID                              int
	workers                            []string // "name=ip" or "name="
	status                             int      // non-zero: every call answers with this HTTP status
}

// fakeWorkerIPsAPI answers GET /nke/clusters/512, its worker-nodes and the VPC's reservations
// from steps, advancing one step per cluster GET and repeating the last.
func fakeWorkerIPsAPI(t *testing.T, steps []workerIPsStep) (*ProviderClients, *int) {
	t.Helper()
	var mu sync.Mutex
	gets, cur := 0, steps[0]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/nke/clusters/512" {
			if gets < len(steps) {
				cur = steps[gets]
			}
			gets++
		}
		if cur.status != 0 {
			w.WriteHeader(cur.status)
			fmt.Fprintf(w, `{"code":%d,"message":"Forbidden"}`, cur.status)
			return
		}
		write := func(data interface{}) {
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": data})
		}
		switch {
		case r.URL.Path == "/nke/clusters/512":
			nodes := map[string]interface{}{"minimum": cur.minimum, "total": cur.total, "deleting": cur.deleting, "outdated": cur.outdated, "maximum": cur.maximum}
			write(map[string]interface{}{"clusterId": 512, "vpcId": cur.vpcID, "nodes": nodes})
		case r.URL.Path == "/nke/clusters/512/worker-nodes":
			rows := []map[string]interface{}{}
			for i, wk := range cur.workers {
				name := strings.SplitN(wk, "=", 2)[0]
				rows = append(rows, map[string]interface{}{"workerNodeId": i, "name": name, "mbpkgid": 990000 + i})
			}
			write(map[string]interface{}{"data": rows, "meta": map[string]int{"total": len(rows)}})
		case r.URL.Path == fmt.Sprintf("/vpcs/%d/ip-reservations", cur.vpcID):
			rows := []map[string]interface{}{{"network": "10.0.0.1/32", "type": "VPC Gateway", "vm": map[string]interface{}{"mbpkgid": nil}}}
			for i, wk := range cur.workers {
				if ip := strings.SplitN(wk, "=", 2)[1]; ip != "" {
					rows = append(rows, map[string]interface{}{"network": ip + "/32", "type": "Primary Interface", "vm": map[string]interface{}{"mbpkgid": 990000 + i}})
				}
			}
			write(map[string]interface{}{"reservations": map[string]interface{}{"data": rows, "meta": map[string]int{"total": len(rows), "limit": 1000}}})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return &ProviderClients{V3: gona.NewV3Client("k", srv.URL)}, &gets
}

func readWorkerIPs(t *testing.T, clients *ProviderClients, timeout time.Duration) (map[string]interface{}, error) {
	t.Helper()
	old := nkeWorkerPrivateIPsInterval
	nkeWorkerPrivateIPsInterval = time.Millisecond
	t.Cleanup(func() { nkeWorkerPrivateIPsInterval = old })
	d := schema.TestResourceDataRaw(t, dataSourceNKEWorkerPrivateIPs().Schema, map[string]interface{}{"cluster_id": 512})
	ctx := context.Background()
	if timeout > 0 { // the SDK bounds the read's context by timeouts { read }
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if diags := dataSourceNKEWorkerPrivateIPsRead(ctx, d, clients); diags.HasError() {
		return nil, fmt.Errorf("%s", diags[0].Summary)
	}
	return d.Get("private_ips").(map[string]interface{}), nil
}

func intp(i int) *int { return &i }

func TestNKEWorkerPrivateIPsSteadyState(t *testing.T) {
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 2, total: 2, maximum: intp(2), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5"}},
	})
	ips, err := readWorkerIPs(t, clients, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 || ips["nke-tcp-1"] != "10.0.0.4" || ips["nke-tcp-2"] != "10.0.0.5" || *gets != 1 {
		t.Fatalf("ips %v after %d cluster reads", ips, *gets)
	}
}

// A third worker that failed and has no address must not hold a healthy two-worker cluster.
func TestNKEWorkerPrivateIPsSurplusWorkerWithoutIPDoesNotBlock(t *testing.T) {
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 2, total: 3, maximum: intp(3), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3="}},
	})
	ips, err := readWorkerIPs(t, clients, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 || *gets != 1 {
		t.Fatalf("ips %v after %d cluster reads, want the 2 with addresses at once", ips, *gets)
	}
}

// After minimum_nodes goes 2 -> 3, the read waits for the third worker's address.
func TestNKEWorkerPrivateIPsWaitsForMinimum(t *testing.T) {
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 3, total: 3, maximum: intp(3), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3="}},
		{minimum: 3, total: 3, maximum: intp(3), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3="}},
		{minimum: 3, total: 3, maximum: intp(3), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3=10.0.0.6"}},
	})
	ips, err := readWorkerIPs(t, clients, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 3 || ips["nke-tcp-3"] != "10.0.0.6" || *gets != 3 {
		t.Fatalf("ips %v after %d cluster reads", ips, *gets)
	}
}

// After maximum_nodes goes 3 -> 2, the read waits until the platform has removed the extra worker,
// so the load balancer never gets the old three.
func TestNKEWorkerPrivateIPsWaitsForScaleDown(t *testing.T) {
	three := []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3=10.0.0.6"}
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 2, total: 3, maximum: intp(2), vpcID: 778, workers: three},
		{minimum: 2, total: 3, maximum: intp(2), deleting: 1, vpcID: 778, workers: three},
		{minimum: 2, total: 2, maximum: intp(2), vpcID: 778, workers: []string{"nke-tcp-2=10.0.0.5", "nke-tcp-3=10.0.0.6"}},
	})
	ips, err := readWorkerIPs(t, clients, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 || ips["nke-tcp-1"] != nil || *gets != 3 {
		t.Fatalf("ips %v after %d cluster reads", ips, *gets)
	}
}

// A rolling version upgrade runs one worker above the maximum while replacing old ones; that is
// not a scale-down and must not hold the read.
func TestNKEWorkerPrivateIPsRollingUpgradeSurgeDoesNotBlock(t *testing.T) {
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 2, total: 3, maximum: intp(2), deleting: 1, outdated: 1, vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3=10.0.0.6"}},
	})
	ips, err := readWorkerIPs(t, clients, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 3 || *gets != 1 {
		t.Fatalf("ips %v after %d cluster reads", ips, *gets)
	}
}

func TestNKEWorkerPrivateIPsNeedsVPC(t *testing.T) {
	clients, _ := fakeWorkerIPsAPI(t, []workerIPsStep{{minimum: 2, total: 2, vpcID: 0}})
	if _, err := readWorkerIPs(t, clients, 0); err == nil || !strings.Contains(err.Error(), "not in a VPC") {
		t.Fatalf("want a not-in-a-VPC error, got %v", err)
	}
}

// A permission or other API error fails at once instead of waiting.
func TestNKEWorkerPrivateIPsFailsFastOnAPIError(t *testing.T) {
	clients, gets := fakeWorkerIPsAPI(t, []workerIPsStep{{status: http.StatusForbidden}})
	if _, err := readWorkerIPs(t, clients, 0); err == nil || *gets != 1 {
		t.Fatalf("want an immediate error, got %v after %d reads", err, *gets)
	}
}

func TestNKEWorkerPrivateIPsTimesOut(t *testing.T) {
	clients, _ := fakeWorkerIPsAPI(t, []workerIPsStep{
		{minimum: 3, total: 3, maximum: intp(3), vpcID: 778, workers: []string{"nke-tcp-1=10.0.0.4", "nke-tcp-2=10.0.0.5", "nke-tcp-3="}},
	})
	_, err := readWorkerIPs(t, clients, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "2 of 3 worker nodes have a private IP at the read timeout") {
		t.Fatalf("want a timeout naming what it waited for, got %v", err)
	}
}

func TestNKEWorkerPrivateIPsReadTimeoutDefault(t *testing.T) {
	if got := dataSourceNKEWorkerPrivateIPs().Timeouts.Read; got == nil || *got != 20*time.Minute {
		t.Fatalf("read timeout: %v, want 20m", got)
	}
}
