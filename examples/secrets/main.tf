provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
}



resource "netactuate_secret_list" "example" {
  name = "Terraform secrets test updated"
}

output "secret_list_id" {
  value = netactuate_secret_list.example.id
}

resource "netactuate_secret_list_value" "secret_1" {
  secret_list_id = netactuate_secret_list.example.id
  secret_key     = "terraform-test-secret"
  secret_value   = "terraform-test-secret-valueqwer"
}

resource "netactuate_secret_list_value" "secret_2" {
  secret_list_id = netactuate_secret_list.example.id
  secret_key     = "terraform-test-secret2"
  secret_value   = "terraform-test-secret-value123"
}


resource "netactuate_secret_list_value" "secret_3" {
  secret_list_id = netactuate_secret_list.example.id
  secret_key     = "terraform-test-secret3updated"
  secret_value   = "test secret value123qwer"
}
