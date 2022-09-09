provider "netactuate" {
  //api_key = "REDACTED_API_KEY"
}

resource "netactuate_server" "server" {
  hostname         = "terraform.example.com"
  plan             = "VR1x1x25"
  location         = "SJC - San Jose, CA"
  image            = "Ubuntu 20.04.4 LTS (20220404)"
  password         = random_password.password.result
}

resource "random_password" "password" {
  length           = 30
  min_lower        = 1
  min_upper        = 1
  min_numeric      = 1
  min_special      = 1
  override_special = "!#$%*()-_=+[]{}:?"
}
