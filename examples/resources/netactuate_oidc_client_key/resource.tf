resource "netactuate_oidc_client_key" "example" {
  label          = "example"
  oidc_client_id = 12345
  public_key     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEexamplekeyexamplekeyexamplekeyexamplekey example"
}
