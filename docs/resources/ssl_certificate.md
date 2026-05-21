# netactuate_ssl_certificate

Manages a TLS/SSL certificate for use with HTTP load balancer groups. Certificates are stored centrally and referenced by ID in `netactuate_http_loadbalancer_group` rule blocks.

## Example Usage

```hcl
resource "netactuate_ssl_certificate" "prod" {
  name        = "prod-wildcard"
  description = "Wildcard cert for *.example.com"
  certificate = file("${path.module}/certs/fullchain.pem")
  private_key = file("${path.module}/certs/privkey.pem")
}

output "cert_id" {
  value = netactuate_ssl_certificate.prod.ssl_certificate_id
}
```

## Argument Reference

### Required

- `name` (String) — Name of the SSL certificate.
- `certificate` (String, Sensitive) — PEM-encoded certificate (including any intermediate chain).
- `private_key` (String, Sensitive) — PEM-encoded private key.

### Optional

- `description` (String) — Description of the SSL certificate.

### Computed

- `ssl_certificate_id` (Number) — The certificate ID assigned by the API.
- `fingerprint` (String) — SHA-256 fingerprint of the certificate.
- `domains` (List of String) — Domain names covered by the certificate (from the SAN extension).
- `is_active` (Boolean) — Whether the certificate is active.
- `status` (String) — Certificate status.
- `expiration` (String) — Certificate expiration date.

## Notes

- `certificate` and `private_key` are marked sensitive and will not appear in plan output.
- All fields can be updated in-place without recreation.

## Import

```
terraform import netactuate_ssl_certificate.prod <ssl_certificate_id>
```
