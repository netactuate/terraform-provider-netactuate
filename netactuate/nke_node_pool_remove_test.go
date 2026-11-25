package netactuate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/netactuate/gona/gona"
)

// PUT size only records a target on the platform, so scale-down and pool
// delete must go through the per-node remove route (found live on eng1,
// Q22). This checks which nodes get a remove call.
func TestRemoveNKEPoolNodes(t *testing.T) {
	var mu sync.Mutex
	var removed []int
	workers := `[
		{"workerNodeId":257,"poolId":71,"deleteScheduled":false},
		{"workerNodeId":258,"poolId":72,"deleteScheduled":false},
		{"workerNodeId":260,"poolId":72,"deleteScheduled":false},
		{"workerNodeId":261,"poolId":72,"deleteScheduled":true}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/nke/clusters/122/worker-nodes":
			w.Write([]byte(`{"code":200,"data":{"meta":{"total":4,"offset":0,"limit":100},"data":` + workers + `}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/nke/clusters/122/worker-nodes/"):
			var id int
			fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/nke/clusters/122/worker-nodes/"), "%d", &id)
			w.Write([]byte(fmt.Sprintf(`{"code":200,"data":{"workerNodeId":%d,"poolId":72}}`, id)))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remove"):
			var id int
			fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/nke/clusters/122/worker-nodes/"), "%d/remove", &id)
			mu.Lock()
			removed = append(removed, id)
			mu.Unlock()
			w.Write([]byte(`{"code":200,"data":{}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	c := gona.NewV3Client("k", srv.URL+"/")

	if err := removeNKEPoolNodes(c, 122, 72, 1); err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != 260 {
		t.Fatalf("scale down by one: want [260], got %v", removed)
	}

	removed = nil
	if err := removeNKEPoolNodes(c, 122, 72, -1); err != nil {
		t.Fatal(err)
	}
	sort.Ints(removed)
	if len(removed) != 2 || removed[0] != 258 || removed[1] != 260 {
		t.Fatalf("drain: want [258 260] (not the default pool node, not the scheduled one), got %v", removed)
	}
}
