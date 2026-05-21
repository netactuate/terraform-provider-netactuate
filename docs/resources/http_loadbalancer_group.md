# netactuate_http_loadbalancer_group

Manages a group on a NetActuate HTTP load balancer. An HTTP LB group handles Layer 7 routing, SSL termination, sticky sessions, domain/path-based routing rules, and backend pools.

The HTTP load balancer ID is available from `netactuate_vpc.main.http_loadbalancer_id` after VPC creation.

## Example Usage

```hcl
resource "netactuate_ssl_certificate" "tls" {
  name        = "prod-cert"
  certificate = file("cert.pem")
  private_key = file("key.pem")
}

resource "netactuate_http_loadbalancer_group" "https" {
  http_loadbalancer_id = netactuate_vpc.main.http_loadbalancer_id
  name                 = "web-https"
  algorithm            = "least-connections"
  internal_port        = 80
  match_address        = netactuate_vpc_floating_ip.pub.address
  match_ports          = "443"

  sticky_sessions_enabled = true

  health_check_active_enabled  = true
  health_check_active_interval = 10
  health_check_active_retries  = 3
  health_check_active_delay    = 5
  health_check_active_path     = "/health"

  rule {
    match_domain            = "example.com"
    match_path              = "/"
    ssl_enabled             = true
    ssl_certificate_id      = netactuate_ssl_certificate.tls.ssl_certificate_id
    https_redirect_enabled  = true
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

- `http_loadbalancer_id` (Number) — The ID of the HTTP load balancer. Forces recreation.
- `name` (String) — Name of the HTTP LB group.
- `algorithm` (String) — Load balancing algorithm: `"round-robin"` or `"least-connections"`.
- `internal_port` (Number) — The port to connect to on backend servers.
- `match_address` (String) — Public IP address to match incoming traffic.
- `match_ports` (String) — Ports to listen on: `"80"`, `"443"`, or `"80+443"`.
- `rule` (Block List, min 1) — Domain/path routing rules (see below).
- `backend` (Block List, min 1) — Backend servers (see below).

### Optional

- `description` (String) — Description of the group.
- `sticky_sessions_enabled` (Boolean) — Enable sticky sessions. Defaults to `false`.
- `ssl_to_backend_enabled` (Boolean) — Use SSL when connecting to backends. Defaults to `false`.
- `health_check_active_enabled` (Boolean) — Enable active health checks. Defaults to `false`.
- `health_check_active_interval` (Number) — Active health check interval in seconds.
- `health_check_active_retries` (Number) — Number of failures before marking unhealthy.
- `health_check_active_delay` (Number) — Delay between retries in seconds.
- `health_check_active_timeout` (Number) — Timeout per check in seconds.
- `health_check_active_path` (String) — HTTP path for active health checks (e.g. `"/health"`).
- `health_check_passive_enabled` (Boolean) — Enable passive health checks. Defaults to `false`.

### Computed

- `http_group_id` (Number) — The group ID assigned by the API.
- `is_online` (Boolean) — Whether the group is online.

### `rule` blocks

- `match_domain` (String, required) — Domain name to match (e.g. `"example.com"`).
- `match_path` (String, required) — Path prefix to match (e.g. `"/"`).
- `ssl_enabled` (Boolean, optional) — Whether SSL termination is enabled for this rule. Defaults to `false`.
- `ssl_certificate_id` (Number, optional) — SSL certificate ID for this rule. Omit to use auto SSL.
- `https_redirect_enabled` (Boolean, optional) — Redirect HTTP to HTTPS. Defaults to `false`.
- `http_rule_id` (Number, computed) — The rule ID assigned by the API.

### `backend` blocks

- `name` (String, required) — Name of the backend.
- `internal_address` (String, required) — Internal IP address of the backend server.
- `http_backend_id` (Number, computed) — The backend ID assigned by the API.
- `is_online` (Boolean, computed) — Whether this backend is passing health checks.

## Import

```
terraform import netactuate_http_loadbalancer_group.https <http_loadbalancer_id>/<http_group_id>
```
