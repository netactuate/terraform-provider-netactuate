package netactuate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

const testTokenSignature = "c2lnbmF0dXJlLW5vdC1jaGVja2Vk"

// testKubeconfig is a kubeconfig shaped like the API's, with a JWT whose claims are claimsJSON.
func testKubeconfig(claimsJSON string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"k"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(claimsJSON))
	return "apiVersion: v1\nkind: Config\nclusters:\n  - name: tcp-1\n    cluster:\n      server: https://example.invalid\n" +
		"users:\n  - name: cluster-admin\n    user:\n      token: " + header + "." + payload + "." + testTokenSignature + "\n" +
		"contexts:\n  - name: cluster-admin@tcp-1\n    context:\n      cluster: tcp-1\n      user: cluster-admin\ncurrent-context: cluster-admin@tcp-1\n"
}

// fakeKubeconfigAPI counts kubeconfig POSTs, answering with kubeconfig, and answers the cluster GET
// (404 once gone).
type fakeKubeconfigAPI struct {
	mu          sync.Mutex
	kubeconfig  string
	posts, gets int
	clusterGone bool
}

func (f *fakeKubeconfigAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/nke/clusters/512/kubeconfig", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.posts++
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "data": f.kubeconfig})
	})
	mux.HandleFunc("/nke/clusters/512", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.gets++
		if f.clusterGone {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":404,"data":{"message":"Not found"}}`))
			return
		}
		w.Write([]byte(`{"code":200,"data":{"clusterId":512}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// expires_on is the token's exp, deliberately different from the requested lifetime; Read never
// issues another token.
func TestNKEKubeconfigExpiryFromTokenAndIssuedOnce(t *testing.T) {
	exp := time.Now().Add(9 * 24 * time.Hour).Truncate(time.Second) // requested 7 days, granted 9
	f := &fakeKubeconfigAPI{kubeconfig: testKubeconfig(fmt.Sprintf(`{"sub":"system:serviceaccount:kube-system:admin-user","iat":%d,"exp":%d}`, time.Now().Unix(), exp.Unix()))}
	clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}

	d := schema.TestResourceDataRaw(t, resourceNKEKubeconfig().Schema, map[string]interface{}{
		"cluster_id": 512, "expiration_seconds": 604800,
	})
	if diags := resourceNKEKubeconfigCreate(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if got, want := d.Get("expires_on").(string), exp.UTC().Format(time.RFC3339); got != want {
		t.Fatalf("expires_on %q, want the token's exp %q", got, want)
	}
	if d.Id() != "nke-kubeconfig-512" || d.Get("kubeconfig").(string) != f.kubeconfig {
		t.Fatalf("state after create: id=%q", d.Id())
	}
	for i := 0; i < 3; i++ {
		if diags := resourceNKEKubeconfigRead(context.Background(), d, clients); diags.HasError() {
			t.Fatalf("read: %v", diags)
		}
	}
	if f.posts != 1 || f.gets != 3 {
		t.Fatalf("kubeconfig POSTs=%d cluster GETs=%d, want 1 and 3", f.posts, f.gets)
	}

	f.clusterGone = true
	if diags := resourceNKEKubeconfigRead(context.Background(), d, clients); diags.HasError() || d.Id() != "" {
		t.Fatalf("read after the cluster is gone: id=%q %v", d.Id(), diags)
	}
}

// Bad tokens are refused, and no error message carries any part of the token.
func TestNKEKubeconfigRejectsBadTokensWithoutLeakingThem(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	cases := map[string]string{
		"no exp":        testKubeconfig(`{"iat":1}`),
		"string exp":    testKubeconfig(fmt.Sprintf(`{"exp":"%d"}`, future)),
		"fraction exp":  testKubeconfig(`{"exp":1.5}`),
		"object exp":    testKubeconfig(`{"exp":{"v":1}}`),
		"expired":       testKubeconfig(fmt.Sprintf(`{"exp":%d}`, time.Now().Add(-time.Hour).Unix())),
		"claims not js": strings.Replace(testKubeconfig(`{"exp":1}`), base64.RawURLEncoding.EncodeToString([]byte(`{"exp":1}`)), base64.RawURLEncoding.EncodeToString([]byte(`not json`)), 1),
		"not a jwt":     strings.Replace(testKubeconfig(`{"exp":1}`), "token: ", "token: opaque-", 1),
		"bad header":    strings.Replace(testKubeconfig(fmt.Sprintf(`{"exp":%d}`, future)), base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"k"}`)), base64.RawURLEncoding.EncodeToString([]byte(`{"kid":"k"}`)), 1),
		"no signature":  strings.Replace(testKubeconfig(fmt.Sprintf(`{"exp":%d}`, future)), "."+testTokenSignature, ".", 1),
		"bad signature": strings.Replace(testKubeconfig(fmt.Sprintf(`{"exp":%d}`, future)), "."+testTokenSignature, ".not*base64url", 1),
		"no token":      strings.Replace(testKubeconfig(`{"exp":1}`), "      token: ", "      client-key-data: ", 1),
		"no context":    strings.Replace(testKubeconfig(fmt.Sprintf(`{"exp":%d}`, future)), "current-context: cluster-admin@tcp-1", "current-context: other", 1),
		"two users":     strings.Replace(testKubeconfig(fmt.Sprintf(`{"exp":%d}`, future)), "contexts:", "  - name: cluster-admin\n    user:\n      token: x.y.z\ncontexts:", 1),
		"not yaml":      "users: [\n" + testTokenSignature,
	}
	for name, kc := range cases {
		f := &fakeKubeconfigAPI{kubeconfig: kc}
		clients := &ProviderClients{V3: gona.NewV3Client("k", f.server(t).URL)}
		d := schema.TestResourceDataRaw(t, resourceNKEKubeconfig().Schema, map[string]interface{}{"cluster_id": 512})
		diags := resourceNKEKubeconfigCreate(context.Background(), d, clients)
		if !diags.HasError() {
			t.Fatalf("%s: want an error", name)
		}
		if msg := diags[0].Summary + diags[0].Detail; strings.Contains(msg, testTokenSignature) || strings.Contains(msg, "eyJ") {
			t.Fatalf("%s: error leaks the token: %s", name, msg)
		}
		if d.Id() != "" {
			t.Fatalf("%s: state written for a refused kubeconfig", name)
		}
	}
}

// A valid kubeconfig plans no change; an expired one, or one whose expires_on cannot be read,
// plans a replacement.
func TestNKEKubeconfigRenewsWhenExpiredOrUnreadable(t *testing.T) {
	r := resourceNKEKubeconfig()
	cfg := terraform.NewResourceConfigRaw(map[string]interface{}{"cluster_id": 512, "expiration_seconds": 604800})
	state := func(expiresOn string) *terraform.InstanceState {
		return &terraform.InstanceState{ID: "nke-kubeconfig-512", Attributes: map[string]string{
			"id": "nke-kubeconfig-512", "cluster_id": "512", "expiration_seconds": "604800",
			"kubeconfig": "x", "expires_on": expiresOn,
		}}
	}
	plan := func(expiresOn string) *terraform.InstanceDiff {
		t.Helper()
		diff, err := r.SimpleDiff(context.Background(), state(expiresOn), cfg, nil)
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		return diff
	}

	if diff := plan(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)); diff != nil && len(diff.Attributes) > 0 {
		t.Fatalf("valid kubeconfig: want no diff, got %+v", diff.Attributes)
	}
	for _, expiresOn := range []string{time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), "not-a-time", ""} {
		if diff := plan(expiresOn); diff == nil || !diff.RequiresNew() {
			t.Fatalf("expires_on %q: want a replacement, got %+v", expiresOn, diff)
		}
	}
}

func TestNKEKubeconfigExpirationAtLeastSevenDays(t *testing.T) {
	v := resourceNKEKubeconfig().Schema["expiration_seconds"].ValidateFunc
	if _, errs := v(604799, "expiration_seconds"); len(errs) == 0 {
		t.Fatal("604799: want an error")
	}
	if _, errs := v(604800, "expiration_seconds"); len(errs) != 0 {
		t.Fatalf("604800: %v", errs)
	}
	// The maximum fits a 32-bit int: the provider is built for 386 and 32-bit ARM.
	if _, errs := v(2147483647, "expiration_seconds"); len(errs) != 0 {
		t.Fatalf("2147483647: %v", errs)
	}
}
