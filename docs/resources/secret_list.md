# netactuate_secret_list

Manages a named secret list. A secret list is a container that holds one or more key/value secrets managed via `netactuate_secret_list_value`. Secret lists can be injected into virtual machines at boot.

## Example Usage

```hcl
resource "netactuate_secret_list" "app_secrets" {
  name = "prod-app-secrets"
}

resource "netactuate_secret_list_value" "db_password" {
  secret_list_id = netactuate_secret_list.app_secrets.id
  secret_key     = "DB_PASSWORD"
  secret_value   = var.db_password
}

resource "netactuate_secret_list_value" "api_token" {
  secret_list_id = netactuate_secret_list.app_secrets.id
  secret_key     = "API_TOKEN"
  secret_value   = var.api_token
}

output "secret_list_id" {
  value = netactuate_secret_list.app_secrets.id
}
```

## Argument Reference

### Required

- `name` (String) — The name of the secret list.

### Computed

- `id` (String) — The API-assigned secret list ID.

## Import

```
terraform import netactuate_secret_list.app_secrets <secret_list_id>
```
