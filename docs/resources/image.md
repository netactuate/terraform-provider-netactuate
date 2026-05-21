# netactuate_image

Creates a custom OS image from an existing cloud server. The image is captured from the running server and can then be used as the `image` argument when deploying new servers.

## Example Usage

```hcl
resource "netactuate_image" "golden" {
  name        = "ubuntu-24-golden"
  description = "Golden Ubuntu 24.04 image with base packages"
  server_id   = netactuate_server.base.id
}
```

## Argument Reference

### Required

- `name` (String) — Name for the custom image.
- `server_id` (Number) — Source server package ID (`mbpkgid`) to capture the image from. Forces recreation.

### Optional

- `description` (String) — Description of the custom image.
- `keep_ssh_userdirs` (Boolean) — Whether to preserve SSH user directories (`~/.ssh`) in the image. Defaults to `false`. Forces recreation.

### Computed

- `os_enabled` (Number) — Image availability status: `1` = enabled, `0` = disabled.
- `bits` (String) — Architecture (`32` or `64`).
- `category` (String) — OS category (e.g. `ubuntu`, `centos`).
- `image_type` (String) — Image type identifier.
- `subtype` (String) — Image subtype.
- `size` (String) — Image disk size.
- `tech` (String) — Virtualization technology.
- `script_bash` (Boolean) — Whether the image supports bash provisioning scripts.
- `script_cloudinit` (Boolean) — Whether the image supports cloud-init.
- `created` (String) — Creation timestamp.

## Notes

- Image creation is asynchronous. The provider waits for the capture job to complete before returning.
- Only `name` and `description` can be updated in-place. All other arguments require recreation.

## Import

```
terraform import netactuate_image.golden <image_id>
```
