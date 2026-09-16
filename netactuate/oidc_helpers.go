package netactuate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func oidcClientSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"oidc_client_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "OIDC client ID assigned by the API",
		},
		"tenant": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Tenant value used with third party OIDC services",
		},
		"label": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Display label for the OIDC client",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "Description for the OIDC client",
		},
		"jwks_uri": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "HTTPS URL where the API fetches public JWKs for this client. Mutually exclusive with netactuate_oidc_client_key: a client with a JWKS URI set cannot also have public keys pushed to it, and attempting both fails the apply with HTTP 400.",
		},
		"account_default": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "If true, this client becomes the account default OIDC client and can affect resources Terraform does not manage. Leave false unless you intend to change the account default.",
		},
		"enforce_allow_list": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Require keyless requests to come from allowed resources",
		},
		"ttl": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     300,
			Description: "JWT lifetime in seconds",
		},
		"default_audience": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "Default audience used when a request does not specify one",
		},
		"created_on": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Timestamp when the client was created",
		},
		"last_used_on": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Timestamp when the client was last used, or empty when unused",
		},
	}
}

func oidcClientDataSourceSchema() map[string]*schema.Schema {
	s := oidcClientSchema()
	for _, v := range s {
		v.Required = false
		v.Optional = false
		v.Computed = true
		v.Default = nil
	}
	s["oidc_client_id"].Optional = true
	s["oidc_client_id"].Computed = false
	s["oidc_client_id"].AtLeastOneOf = []string{"oidc_client_id", "label"}
	s["label"].Optional = true
	s["label"].Computed = false
	s["label"].AtLeastOneOf = []string{"oidc_client_id", "label"}
	return s
}

func oidcClientListElementSchema() map[string]*schema.Schema {
	s := oidcClientDataSourceSchema()
	s["oidc_client_id"].Optional = false
	s["oidc_client_id"].Computed = true
	s["oidc_client_id"].AtLeastOneOf = nil
	s["label"].Optional = false
	s["label"].Computed = true
	s["label"].AtLeastOneOf = nil
	return s
}

func flattenOIDCClient(client gona.OIDCClient) map[string]interface{} {
	result := map[string]interface{}{
		"oidc_client_id":     client.ClientID,
		"tenant":             client.Tenant.String(),
		"label":              client.Label,
		"description":        client.Description,
		"account_default":    client.AccountDefault,
		"enforce_allow_list": client.EnforceAllowList.Bool(),
		"ttl":                client.TTL,
		"default_audience":   client.DefaultAudience,
		"created_on":         client.CreatedOn,
		"last_used_on":       "",
		"jwks_uri":           "",
	}
	if client.LastUsedOn != nil {
		result["last_used_on"] = *client.LastUsedOn
	}
	if client.JWKSURI != nil {
		result["jwks_uri"] = *client.JWKSURI
	}
	return result
}

func parseTwoPartID(id, nameA, nameB string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid ID format %q, expected {%s}/{%s}", id, nameA, nameB)
	}
	a, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s %q: %s", nameA, parts[0], err)
	}
	b, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s %q: %s", nameB, parts[1], err)
	}
	return a, b, nil
}

func flattenOIDCKey(key gona.OIDCClientKey) map[string]interface{} {
	publicKey := key.PublicKey
	if publicKey == "" {
		publicKey = key.Value
	}
	return map[string]interface{}{
		"key_id":      key.KeyID,
		"label":       key.Label,
		"description": key.Description,
		"provided_on": key.ProvidedOn,
		"type":        key.Type,
		"public_key":  publicKey,
	}
}

func oidcKeyElementSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "OIDC client key ID assigned by the API",
		},
		"label": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Display label for the key",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Description for the key",
		},
		"provided_on": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Timestamp when the key was provided",
		},
		"type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Key type",
		},
		"public_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "Public JWK value",
		},
	}
}

func flattenOIDCAuthLog(log gona.OIDCClientAuthLog) map[string]interface{} {
	return map[string]interface{}{
		"log_id":     log.LogID,
		"issued_on":  log.IssuedOn,
		"expires_on": log.ExpiresOn,
		"jti":        log.JTI,
	}
}

func flattenOIDCChangeLog(log gona.OIDCClientChangeLog) map[string]interface{} {
	return map[string]interface{}{
		"key_id":      log.KeyID,
		"recorded_on": log.RecordedOn,
		"type":        log.Type,
	}
}

func oidcVMRawJSON(vm gona.OIDCClientVM) string {
	if len(vm.Raw) == 0 {
		return ""
	}
	var compact json.RawMessage
	if err := json.Unmarshal(vm.Raw, &compact); err != nil {
		return string(vm.Raw)
	}
	out, err := json.Marshal(compact)
	if err != nil {
		return string(vm.Raw)
	}
	return string(out)
}
