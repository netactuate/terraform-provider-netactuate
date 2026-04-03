# netactuate_vpc_ssh_key

Associates an account-level SSH key with a VPC, enabling or disabling it for use within that VPC. The SSH key must already exist as a `netactuate_sshkey` resource.

## Example Usage

```hcl
resource "netactuate_sshkey" "deploy" {
  name       = "deploy-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "netactuate_vpc" "main" {
  label       = "prod-vpc"
  description = "Production VPC"
  location    = "SJC"
}

resource "netactuate_vpc_ssh_key" "deploy" {
  vpc_id     = netactuate_vpc.main.vpc_id
  ssh_key_id = netactuate_sshkey.deploy.id
  enabled    = true
}

output "ssh_key_fingerprint" {
  value = netactuate_vpc_ssh_key.deploy.fingerprint
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the VPC. Forces recreation.
- `ssh_key_id` (Number) — The account-level SSH key ID (from `netactuate_sshkey.id`). Forces recreation.

### Optional

- `enabled` (Boolean) — Whether the SSH key is enabled for use in this VPC. Default: `true`. Can be updated in-place.

### Computed

- `name` (String) — The name of the SSH key.
- `fingerprint` (String) — The SSH key fingerprint.
- `public_key` (String) — The public key content.

## Import

```
terraform import netactuate_vpc_ssh_key.deploy <vpc_id>/<ssh_key_id>
```

## Notes

- Deleting this resource disables the SSH key in the VPC rather than deleting the underlying account key.
