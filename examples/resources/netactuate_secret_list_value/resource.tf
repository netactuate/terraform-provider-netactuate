resource "netactuate_secret_list_value" "example" {
  secret_key     = "example_key"
  secret_list_id = 12345
  secret_value   = "change-me-example-value"
}
