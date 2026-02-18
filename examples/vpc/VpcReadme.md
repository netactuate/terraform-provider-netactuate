# VPC Example

## Setup

```bash
cd examples/vpc
terraform init
```

## Apply everything

```bash
terraform plan
terraform apply
```

## Check state

```bash
# Show all resources in local state
terraform state list

# Show details of a specific resource
terraform state show netactuate_vpc.example
terraform state show netactuate_vpc_gateway_snat_rule.outbound

# Show the full raw state file
terraform state pull
```

## Destroy everything at once

```bash
terraform destroy
```

## Destroy step by step

Destroy in dependency order — children before parents.

```bash
# 1. Load balancer groups (depend on LBs which are part of the VPC)
terraform destroy -target=netactuate_http_loadbalancer_group.web_https
terraform destroy -target=netactuate_network_loadbalancer_group.web_lb

# 2. Gateway rules
terraform destroy -target=netactuate_vpc_gateway_dnat_rule.web
terraform destroy -target=netactuate_vpc_gateway_dnat_rule.web_v6
terraform destroy -target=netactuate_vpc_gateway_snat_rule.outbound
terraform destroy -target=netactuate_vpc_gateway_snat_rule.outbound_v6
terraform destroy -target=netactuate_vpc_gateway_snat_rule.outbound_secondary
terraform destroy -target=netactuate_vpc_gateway_firewall_rule.allow_http_ipv4
terraform destroy -target=netactuate_vpc_gateway_firewall_rule.allow_http_ipv6
terraform destroy -target=netactuate_vpc_gateway_firewall_rule.allow_ssh_ipv6

# 3. Floating IPs
terraform destroy -target=netactuate_vpc_floating_ip.extra_ipv4
terraform destroy -target=netactuate_vpc_floating_ip.extra_ipv6

# 4. Backend templates
terraform destroy -target=netactuate_vpc_backend_template.web_backends

# 5. SSL certificates
terraform destroy -target=netactuate_ssl_certificate.example

# 6. VPC-level SSH key
terraform destroy -target=netactuate_vpc_ssh_key.bastion_key

# 7. Server (remove from state first if you want to keep it, otherwise destroy)
terraform state rm netactuate_server.vpc_server
# or: terraform destroy -target=netactuate_server.vpc_server

# 8. VPC itself
terraform destroy -target=netactuate_vpc.example

# 9. Account-level SSH key
terraform destroy -target=netactuate_sshkey.sshkey
```
