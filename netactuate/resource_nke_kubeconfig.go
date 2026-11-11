package netactuate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
	"gopkg.in/yaml.v3"
)

const nkeKubeconfigDescription = "Issues a kubeconfig for an NKE cluster once and keeps it in state; the next apply after it expires replaces it. " +
	"The kubeconfig holds a cluster-admin token, stored in plain text in the Terraform state: protect the state. " +
	"Destroying the resource does not revoke the token; it stays valid until it expires."

// nkeKubeconfigExpirationSchema is expiration_seconds for the kubeconfig resource and data source.
func nkeKubeconfigExpirationSchema(forceNew bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		ForceNew:    forceNew,
		Default:     604800,
		Description: "Requested token lifetime in seconds. The minimum lifetime is 7 days (604800), which is also the default",
	}
}

func resourceNKEKubeconfig() *schema.Resource {
	return &schema.Resource{
		Description:   nkeKubeconfigDescription,
		CreateContext: resourceNKEKubeconfigCreate,
		ReadContext:   resourceNKEKubeconfigRead,
		DeleteContext: resourceNKEKubeconfigDelete,
		CustomizeDiff: resourceNKEKubeconfigRenewWhenExpired,
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The NKE cluster ID",
			},
			"expiration_seconds": nkeKubeconfigExpirationSchema(true),
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The kubeconfig YAML for accessing the cluster",
			},
			"expires_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the token expires (RFC 3339), from its exp claim. The next apply after that replaces the kubeconfig",
			},
		},
	}
}

func resourceNKEKubeconfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	kubeconfig, err := c.GenerateNKEKubeconfig(clusterID, d.Get("expiration_seconds").(int))
	if err != nil {
		return diag.FromErr(err)
	}
	expiresOn, err := kubeconfigTokenExpiry(kubeconfig)
	if err != nil {
		return diag.Errorf("kubeconfig for NKE cluster %d: %s", clusterID, err)
	}
	if !expiresOn.After(time.Now()) {
		return diag.Errorf("kubeconfig for NKE cluster %d: the token is already expired", clusterID)
	}

	var diags diag.Diagnostics
	d.SetId(fmt.Sprintf("nke-kubeconfig-%d", clusterID))
	setValue("kubeconfig", kubeconfig, d, &diags)
	setValue("expires_on", expiresOn.UTC().Format(time.RFC3339), d, &diags)
	return diags
}

func resourceNKEKubeconfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3
	clusterID := d.Get("cluster_id").(int)

	if _, err := c.GetNKECluster(clusterID); err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] NKE cluster %d not found, removing its kubeconfig from state", clusterID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceNKEKubeconfigDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

func resourceNKEKubeconfigRenewWhenExpired(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	if d.Id() == "" {
		return nil
	}
	expiresOn, err := time.Parse(time.RFC3339, d.Get("expires_on").(string))
	if err == nil && time.Now().Before(expiresOn) {
		return nil
	}
	for _, k := range []string{"kubeconfig", "expires_on"} {
		if err := d.SetNewComputed(k); err != nil {
			return err
		}
	}
	return d.ForceNew("kubeconfig")
}

// kubeconfigTokenExpiry returns the exp claim of the current context's token.
func kubeconfigTokenExpiry(kubeconfig string) (time.Time, error) {
	var cfg struct {
		CurrentContext string `yaml:"current-context"`
		Contexts       []struct {
			Name    string `yaml:"name"`
			Context struct {
				User string `yaml:"user"`
			} `yaml:"context"`
		} `yaml:"contexts"`
		Users []struct {
			Name string `yaml:"name"`
			User struct {
				Token string `yaml:"token"`
			} `yaml:"user"`
		} `yaml:"users"`
	}
	if err := yaml.Unmarshal([]byte(kubeconfig), &cfg); err != nil {
		return time.Time{}, fmt.Errorf("the kubeconfig is not valid YAML")
	}
	userName := ""
	for _, c := range cfg.Contexts {
		if c.Name == cfg.CurrentContext && cfg.CurrentContext != "" {
			userName = c.Context.User
		}
	}
	if userName == "" {
		return time.Time{}, fmt.Errorf("the kubeconfig's current-context does not name a user")
	}
	token, found := "", 0
	for _, u := range cfg.Users {
		if u.Name == userName {
			token, found = u.User.Token, found+1
		}
	}
	if found != 1 || token == "" {
		return time.Time{}, fmt.Errorf("the kubeconfig has no single token for its current-context user")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[2] == "" {
		return time.Time{}, fmt.Errorf("the token is not a JWT")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerJSON, &header) != nil || header.Alg == "" {
		return time.Time{}, fmt.Errorf("the token's header is not a JWT header")
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[2]); err != nil {
		return time.Time{}, fmt.Errorf("the token's signature is not valid base64url")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("the token's claims are not valid base64url")
	}
	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("the token's claims are not valid JSON")
	}
	raw, ok := claims["exp"]
	if !ok {
		return time.Time{}, fmt.Errorf("the token has no exp claim")
	}
	if s := strings.TrimSpace(string(raw)); s == "" || s[0] == '"' {
		return time.Time{}, fmt.Errorf("the token's exp claim is not a number")
	}
	var exp json.Number
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&exp); err != nil {
		return time.Time{}, fmt.Errorf("the token's exp claim is not a number")
	}
	secs, err := exp.Int64()
	if err != nil || secs <= 0 {
		return time.Time{}, fmt.Errorf("the token's exp claim is not a whole number of seconds")
	}
	return time.Unix(secs, 0), nil
}
