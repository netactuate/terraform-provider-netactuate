# netactuate_secret_list_value

Manages an individual key/value entry within a `netactuate_secret_list`. Each value is stored as a named secret and can be updated in-place.

## Example Usage

```hcl
resource "netactuate_secret_list" "app" {
  name = "prod-app-secrets"
}

resource "netactuate_secret_list_value" "db_url" {
  secret_list_id = netactuate_secret_list.app.id
  secret_key     = "DATABASE_URL"
  secret_value   = "postgres://user:pass@db.internal:5432/prod"
}

resource "netactuate_secret_list_value" "jwt_secret" {
  secret_list_id = netactuate_secret_list.app.id
  secret_key     = "JWT_SECRET"
  secret_value   = var.jwt_secret
}
```

## Argument Reference

### Required

- `secret_list_id` (String) — The ID of the parent secret list. Forces recreation.
- `secret_key` (String) — The key name for this secret, e.g. `"DB_PASSWORD"`.
- `secret_value` (String) — The secret value. Can be updated in-place.

### Computed

- `id` (String) — The API-assigned secret value ID.

## Import

```
terraform import netactuate_secret_list_value.db_url <secret_list_id>/<secret_key>
```
