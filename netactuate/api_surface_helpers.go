package netactuate

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func rawJSONDataSourceSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: description,
	}
}

func compactSurfaceJSON(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func stringsToNameservers(values []interface{}) []gona.VPCNameserver {
	nameservers := make([]gona.VPCNameserver, len(values))
	for i, value := range values {
		nameservers[i] = gona.VPCNameserver{Server: value.(string)}
	}
	return nameservers
}

func flattenNameservers(values []gona.VPCNameserver) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = normalizeNameserver(value.Server)
	}
	return out
}

// normalizeNameserver strips the default host mask the platform appends to a bare
// nameserver address (/32 for IPv4, /128 for IPv6). Without this a bare-IP
// configuration reads back as CIDR and the plan never settles.
func normalizeNameserver(server string) string {
	server = strings.TrimSuffix(server, "/32")
	server = strings.TrimSuffix(server, "/128")
	return server
}

func parseSurfaceTwoPartID(id, want string) (int, int, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, 0, strconv.ErrSyntax
	}
	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	_ = want
	return first, second, nil
}
