package netactuate

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func applyDocumentation(p *schema.Provider) {
	for name, field := range p.Schema {
		if field.Description == "" {
			field.Description = providerFieldDescription(name)
		}
	}
	for name, resource := range p.ResourcesMap {
		short := strings.TrimPrefix(name, "netactuate_")
		if resource.Description == "" {
			resource.Description = resourceDescription(short)
		}
		fillSchemaDescriptions(resource.Schema)
	}
	for name, dataSource := range p.DataSourcesMap {
		short := strings.TrimPrefix(name, "netactuate_")
		if dataSource.Description == "" {
			dataSource.Description = dataSourceDescription(short)
		}
		fillSchemaDescriptions(dataSource.Schema)
	}
}

func fillSchemaDescriptions(fields map[string]*schema.Schema) {
	for name, field := range fields {
		if field.Description == "" {
			field.Description = attributeDescription(name, field)
		}
		switch elem := field.Elem.(type) {
		case *schema.Resource:
			fillSchemaDescriptions(elem.Schema)
		case *schema.Schema:
			if elem.Description == "" {
				elem.Description = listElementDescription(name)
			}
		}
	}
}

func providerFieldDescription(name string) string {
	descriptions := map[string]string{
		"api_key": "NetActuate API key. This may also be set with the NETACTUATE_API_KEY environment variable.",
		"api_url": "Base URL for the vAPI2 endpoint. Leave unset to use the NetActuate production API.",
	}
	if description, ok := descriptions[name]; ok {
		return description
	}
	return "NetActuate provider setting."
}

func resourceDescription(name string) string {
	special := map[string]string{
		"bgp_group":                   "Manages a BGP group. NetActuate does not expose a delete endpoint for BGP groups, so deleting the resource removes it from Terraform state only.",
		"bgp_prefix_purchase":         "Purchases an anycast BGP prefix. NetActuate does not expose a release endpoint for purchased prefixes, so deleting the resource removes it from Terraform state only.",
		"firewall_set_detach_all_vms": "Detaches all VMs from a firewall set as an explicit action resource.",
		"firewall_set_rule_order":     "Moves a firewall rule within a firewall set rule order.",
		"firewall_set_vm_detach":      "Detaches a VM relation from a firewall set as an explicit action resource.",
	}
	if description, ok := special[name]; ok {
		return description
	}
	return "Manages " + article(subjectName(name)) + " " + subjectName(name) + "."
}

func dataSourceDescription(name string) string {
	special := map[string]string{
		"cloud_plan_id":          "Looks up a cloud plan ID by plan name.",
		"current_server":         "Reads metadata for the server associated with the current execution environment.",
		"nke_kubeconfig":         "Reads kubeconfig content for an NKE cluster.",
		"platform_looking_glass": "Runs a platform looking glass query and returns the API response.",
		"resource_tags":          "Reads tags assigned to a supported NetActuate resource.",
	}
	if description, ok := special[name]; ok {
		return description
	}
	subject := subjectName(name)
	if strings.HasSuffix(name, "s") {
		return "Reads " + subject + " visible to the account."
	}
	return "Reads " + article(subject) + " " + subject + "."
}

func attributeDescription(name string, field *schema.Schema) string {
	descriptions := map[string]string{
		"address_id": "Address ID assigned by the API.", "agreement_id": "Accepted legal agreement ID required for the order.", "bgp_group_id": "BGP group ID assigned by the API.",
		"certificate": "PEM encoded certificate body.", "cluster_id": "NKE cluster ID assigned by the API.", "content": "Record value or response content returned by the API.",
		"description": "Human readable description stored with the object.", "device_id": "Dedicated device ID accepted by the API.", "direction": "Traffic direction matched by the rule.",
		"enabled": "Whether the feature or rule is enabled.", "firewall_set_id": "Firewall set ID assigned by the API.", "group_id": "BGP group ID assigned by the API.",
		"hostname": "Hostname assigned to the server.", "interface_id": "Router interface ID assigned by the API.", "ip_version": "IP version for this object. Accepted values are 4 and 6 unless the field description states otherwise.",
		"label": "Display label for the object.", "local_asn": "BGP local ASN. Between 1 and 4294967294.", "match_network": "CIDR network matched by the rule.",
		"mb_id": "Package billing contract ID assigned by the API.", "mbpkgid": "Package ID assigned by the API.", "name": "Display name for the object.",
		"network": "CIDR network accepted by the routing API.", "oidc_client_id": "OIDC client ID assigned by the API.", "plan": "Plan name accepted by the API.",
		"private_key": "PEM encoded private key for the certificate.", "profile": "Dedicated server operating system profile ID accepted by the API.", "protocol": "Protocol matched by the rule. Common values are TCP, UDP and ICMP.",
		"psk_secret": "Pre shared key used by the IPsec peer.", "public_key": "Public key material accepted by the API.", "remote_asn": "BGP remote ASN. Between 1 and 4294967294.",
		"resource_name": "Resource type name accepted by the tag API.", "reverse": "PTR record value for reverse DNS.", "router_id": "Cloud router ID assigned by the API.",
		"secret_key": "Secret key name within the secret list.", "secret_list_id": "Secret list ID assigned by the API.", "secret_value": "Sensitive secret value stored under the key.",
		"service_id": "Service ID assigned by the API.", "ssh_key_id": "SSH key ID assigned by the API.", "subnet": "CIDR subnet accepted by the API.",
		"tag_id": "Tag ID assigned by the API.", "translation_address": "Address used as the NAT translation target.", "translation_network": "CIDR network used as the NAT translation target.",
		"type": "Object type accepted by the API.", "version": "Version string accepted by the API.", "vrf_id": "Cloud router VRF ID assigned by the API.", "zone_id": "DNS zone ID assigned by the API.",
	}
	if description, ok := descriptions[name]; ok {
		return description
	}
	if field.Sensitive {
		return "Sensitive value sent to the NetActuate API."
	}
	if field.Computed && !field.Optional {
		return "Value returned by the NetActuate API for this field."
	}
	return "Value sent to the NetActuate API for this field."
}

func listElementDescription(name string) string {
	descriptions := map[string]string{"domains": "Domain name covered by the certificate.", "endpoints": "Service endpoint URL.", "ipv4": "IPv4 resolver address.", "ipv6": "IPv6 resolver address.", "nameservers": "Nameserver hostname.", "source_net": "Source CIDR network matched by the rule.", "destination_net": "Destination CIDR network matched by the rule.", "tag_ids": "Tag ID assigned to the resource."}
	if description, ok := descriptions[name]; ok {
		return description
	}
	return "List element value returned by the NetActuate API."
}

func subjectName(name string) string {
	replacements := map[string]string{"asn": "ASN", "bgp": "BGP", "ddos": "DDoS", "dhcp": "DHCP", "dns": "DNS", "dnat": "DNAT", "http": "HTTP", "ipsec": "IPsec", "ipv4": "IPv4", "ipv6": "IPv6", "nke": "NKE", "ntp": "NTP", "oidc": "OIDC", "snat": "SNAT", "ssh": "SSH", "ssl": "SSL", "vm": "VM", "vms": "VMs", "vpc": "VPC", "vrf": "VRF"}
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if replacement, ok := replacements[part]; ok {
			parts[i] = replacement
		} else if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func article(subject string) string {
	if strings.Contains("AEIOUaeiou", subject[:1]) {
		return "an"
	}
	return "a"
}
