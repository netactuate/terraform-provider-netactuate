# Read the certificate and its key from files rather than inlining them, so the key does not
# end up in the configuration or in version control.
resource "netactuate_ssl_certificate" "example" {
  name        = "example"
  certificate = file("${path.module}/example.crt")
  private_key = file("${path.module}/example.key")
}
