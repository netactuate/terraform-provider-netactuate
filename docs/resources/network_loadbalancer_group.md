# netactuate_network_loadbalancer_group

Manages a group on a NetActuate network (TCP/UDP) load balancer. A group defines the listening address, load balancing algorithm, forwarding rules, backends, and health check settings.

The network load balancer ID is available from `netactuate_vpc.main.network_loadbalancer_id` after VPC creation.

## Example Usage

```hcl
resource "netactuate_network_loadbalancer_group" "tcp80" {
  network_loadbalancer_id = netactuate_vpc.main.network_loadbalancer_id
  name                    = "web-tcp"
  description             = "HTTP traffic"
  ip_version              = 4
  algorithm               = "least-connections"
  match_address           = netactuate_vpc_floating_ip.pub.address

  health_check {
    enabled  = true
    method   = "Ping"
    interval = 10
    retries  = 3
    delay    = 5
    timeout  = 5
  }

  rule {
    protocol      = "TCP"
    port_match    = 80
    port_internal = 80
  }

  backend {
    name             = "backend1"
    internal_address = "10.10.0.10"
  }

  backend {
    name             = "backend2"
    internal_address = "10.10.0.11"
  }
}
```

## Argument Reference

### Required

- `network_loadbalancer_id` (Number) — The ID of the network load balancer. Forces recreation.
- `name` (String) — Name of the load balancer group.
- `ip_version` (Number) — IP version: `4` or `6`. Forces recreation.
- `algorithm` (String) — Load balancing algorithm (e.g. `"round-robin"`, `"least-connections"`).
- `match_address` (String) — Public IP address to match incoming traffic.
- `health_check` (Block, required) — Health check configuration (see below).
- `rule` (Block List, min 1) — Forwarding rules (see below).
- `backend` (Block List, min 1) — Backend servers (see below).

### Optional

- `description` (String) — Description of the group.

### Computed

- `network_group_id` (Number) — The group ID assigned by the API.
- `is_online` (Boolean) — Whether the group is online.

### `health_check` block

- `enabled` (Boolean, required) — Whether health checks are enabled.
- `method` (String, required) — Health check method (e.g. `"Ping"`, `"TCP"`).
- `interval` (Number, required) — Interval between checks in seconds.
- `retries` (Number, required) — Number of failures before marking unhealthy.
- `delay` (Number, required) — Delay between retries in seconds.
- `timeout` (Number, required) — Timeout per check in seconds.

### `rule` blocks

- `protocol` (String, required) — Protocol: `"TCP"` or `"UDP"`.
- `port_match` (Number, required) — External port to listen on.
- `port_internal` (Number, required) — Internal port to forward to on backends.
- `network_rule_id` (Number, computed) — The rule ID assigned by the API.

### `backend` blocks

- `name` (String, required) — Name of the backend.
- `internal_address` (String, required) — Internal IP address of the backend server.
- `network_backend_id` (Number, computed) — The backend ID assigned by the API.
- `is_online` (Boolean, computed) — Whether this backend is passing health checks.

## Import

```
terraform import netactuate_network_loadbalancer_group.tcp80 <network_loadbalancer_id>/<network_group_id>
```
