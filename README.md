# Terraform Provider NetActuate

## Usage
Currently in this stage of development -

Basic Steps to test:
```bash
# Grab the provider:
git clone git@github.com:netactuate/terraform-provider-netactuate.git

# Install and download all the dependencies and compile all related binaries
cd terraform-netactuate-provider
make install-all

# Edit an example: [basic, full, cluster]
cd examples/basic
export NETACTUATE_API_KEY="my-api-key"
edit main.tf
terraform init
terraform plan
terraform apply
```

### Authentication
There are the following ways of providing credentials for authentication:
1. Static credentials
2. Environment variable

#### Static credentials
> **_NOTE:_** \
> Hard-coded credentials are not recommended in any Terraform configuration and risks secret leakage should this file
> ever be committed to a public version control system.

Static credentials can be provided by adding an `api_key` in-line in the provider block:
```terraform
provider "netactuate" {
  api_key = "my-api-key"
}
```

#### Environment Variables
You can provide your credentials via the `NETACTUATE_API_KEY` environment variable, representing your NetActuate API Key:
```terraform
provider "netactuate" {}
```
```bash
export NETACTUATE_API_KEY="my-api-key"
terraform apply
```

## Development

### dev_overrides (recommended)

Terraform's `dev_overrides` mechanism points the CLI directly at a local binary, bypassing the registry entirely. This means no `terraform init` is needed and you never have to delete `.terraform.lock.hcl` after a rebuild.

**1. Create or edit `~/.terraformrc`:**

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/netactuate/netactuate" = "/home/<you>/go/bin"
  }
  direct {}
}
```

Replace `/home/<you>/go/bin` with the output of:

```bash
go env GOPATH
# result: /home/<you>/go  →  append /bin
```

**2. Build and install the binary:**

```bash
go build -o $GOPATH/bin/terraform-provider-netactuate .
```

**3. Run Terraform — skip `terraform init`:**

```bash
cd examples/pop/01-vm-s3
terraform plan
terraform apply
```

Terraform will print a warning that dev_overrides are active. This is expected and can be ignored.

> **Note:** `direct {}` is required in the `provider_installation` block so that all other providers (e.g. `hashicorp/terraform_data`) are still fetched from the registry normally.

### Local `gona` replace (development only)

For provider development and integration testing in this workspace, `go.mod` includes:

```go
replace github.com/netactuate/gona => ../gona-modifications
```

This is intentional for local testing only. Before publishing/releasing the provider, remove that `replace` line so builds consume the upstream `github.com/netactuate/gona` module version.

### Run locally (legacy plugin directory)
Do the following to run and test the TF provider locally:
1. Compile and install the TF provider's binaries to the local TF plugins directory:
    ```bash
    make install-all
    ```
2. Install TF providers for the test [example](examples/basic,full,cluster):
    ```bash
    cd example
    terraform init
    ```
   Every time the provider is re-built, `.terraform.lock.hcl` file must be removed and the
   test example modules re-initialize, because the provider dependency's hash changes
3. Build the infrastructure:
    ```bash
    terraform apply
    ```

### Custom API URL
If necessary, you can override the default NetActuate API URL by specifying a custom `api_url` in the provider block:
```terraform
provider "netactuate" {
  api_url = "https://api.example.com/"
}
```
