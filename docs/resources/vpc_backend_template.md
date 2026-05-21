# netactuate_vpc_backend_template

Manages a named backend pool template within a VPC. Backend templates group a set of server IP addresses that can be referenced by VPC load balancer groups, allowing the backend pool to be managed independently of the load balancer configuration.

## Example Usage

```hcl
resource "netactuate_vpc_backend_template" "app" {
  vpc_id      = netactuate_vpc.main.vpc_id
  name        = "app-backends"
  description = "Application server pool"

  backend_host {
    name    = "backend1"
    address = "10.10.0.10"
  }

  backend_host {
    name    = "backend2"
    address = "10.10.0.11"
  }
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the parent VPC. Forces recreation.
- `name` (String) — Name of the backend template.

### Optional

- `description` (String) — Description of the backend template.
- `backend_host` (Block List) — Backend hosts in this template (see below).

### Computed

- `backend_template_id` (Number) — The backend template ID assigned by the API.

### `backend_host` blocks

- `name` (String, required) — Name of the backend host.
- `address` (String, required) — IP address of the backend host (typically a private VPC IP).
- `backend_host_id` (Number, computed) — The backend host ID assigned by the API.

## Notes

- Updates to `name`, `description`, or the `backend_host` list are applied with a full replace operation — the entire backend template is rewritten atomically.

## Import

```
terraform import netactuate_vpc_backend_template.app <vpc_id>/<backend_template_id>
```
