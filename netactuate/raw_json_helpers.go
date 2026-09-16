package netactuate

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/netactuate/gona/gona"
)

func rawMapString(raw map[string]json.RawMessage) (string, error) {
	out, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func rawMapInt(raw map[string]json.RawMessage, key string) int {
	value, ok := raw[key]
	if !ok {
		return 0
	}
	var number json.Number
	if err := json.Unmarshal(value, &number); err == nil {
		if i, err := strconv.Atoi(number.String()); err == nil {
			return i
		}
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		if i, err := strconv.Atoi(text); err == nil {
			return i
		}
	}
	return 0
}

func rawMapStringField(raw map[string]json.RawMessage, key string) string {
	value, ok := raw[key]
	if !ok {
		return ""
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		return text
	}
	return string(value)
}

func flattenRawMaps[T ~map[string]json.RawMessage](items []T, idKey string) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		raw := map[string]json.RawMessage(item)
		rawJSON, err := rawMapString(raw)
		if err != nil {
			return nil, fmt.Errorf("encode row %d: %w", i, err)
		}
		// Rows name their identifier after themselves: a dedicated device carries device_id.
		id := rawMapInt(raw, idKey)
		if id == 0 {
			id = rawMapInt(raw, "id")
		}
		result[i] = map[string]interface{}{
			idKey:      id,
			"name":     rawMapStringField(raw, "name"),
			"raw_json": rawJSON,
		}
	}
	return result, nil
}

func flattenDedicatedServers(servers []gona.SingleMetal) []map[string]interface{} {
	result := make([]map[string]interface{}, len(servers))
	for i, server := range servers {
		result[i] = flattenDedicatedServer(server)
	}
	return result
}

func flattenDedicatedServer(server gona.SingleMetal) map[string]interface{} {
	return map[string]interface{}{
		"dedicated_server_id": server.ID,
		"mbpkgid":             server.MBPKGID,
		"hostname":            server.Hostname,
		"datacenter_id":       server.DatacenterID,
		"location":            server.Location,
		"primary_ipv4":        server.PrimaryIP,
		"primary_ipv6":        stringPtrOrEmpty(server.PrimaryIPv6),
		"package_status":      server.PackageStatus,
		"canceling":           server.Canceling,
		"nps_installed":       server.NPSInstalled,
		"nps_os":              server.NPSOS,
		"locked":              server.Locked,
		"locked_msg":          stringPtrOrEmpty(server.LockedMsg),
		"ipmi_status":         server.IPMIStatus,
		"ipmi_pubip":          server.IPMIPubIP,
		"ipmi_status_time":    server.IPMIStatusTime,
	}
}
