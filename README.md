# Terraform Provider for NetActuate

Manage NetActuate infrastructure with Terraform: cloud and dedicated compute, VPCs and their
gateways, load balancers, BGP and anycast, object and block storage, DNS, secrets, and NKE
clusters with their add-ons.

Documentation for every resource and data source is published at
[registry.terraform.io/providers/netactuate/netactuate](https://registry.terraform.io/providers/netactuate/netactuate/latest/docs).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0 or later
- [Go](https://go.dev/doc/install) 1.25 or later, to build the provider from source
- A NetActuate account and an API key

## Using the provider

```terraform
terraform {
  required_providers {
    netactuate = {
      source  = "netactuate/netactuate"
      version = "~> 0.4"
    }
  }
}

provider "netactuate" {}

resource "netactuate_server" "web" {
  hostname    = "web1.example.com"
  plan        = "VR1x1x25"
  location_id = 3
  image       = "Ubuntu 22.04 LTS x64"
}
```

Run `terraform init` to download the provider from the registry.

### Beta surface

The cloud router resources, `netactuate_router` and every resource named `netactuate_router_*`,
are in beta. Their schema and behaviour may change in a minor release, and some operations
depend on platform work that is still in progress. The rest of the provider is stable.

## Authentication

The provider reads an API key from the `NETACTUATE_API_KEY` environment variable, so the key
need not appear in a configuration file:

```bash
export NETACTUATE_API_KEY="your-api-key"
terraform apply
```

Set it in the provider block instead when a configuration manages more than one account:

```terraform
provider "netactuate" {
  api_key = var.netactuate_api_key
}
```

> **Note**
> Hard coded credentials are not recommended in any Terraform configuration, and risk secret
> leakage should the file ever be committed to version control.

### Custom API URL

Override the default endpoint when you need to point at something other than production:

```terraform
provider "netactuate" {
  api_url = "https://api.example.com/"
}
```

## Developing the provider

Clone the repository and build:

```bash
git clone https://github.com/netactuate/terraform-provider-netactuate.git
cd terraform-provider-netactuate
go build ./...
```

Run the unit tests:

```bash
go test ./...
```

Acceptance tests create real, billable infrastructure. They run only when `TF_ACC` is set and
are guarded by a build tag:

```bash
TF_ACC=1 NETACTUATE_API_KEY="your-api-key" go test -tags acctest ./... -run TestAccNetactuate -v
```

### Using a locally built binary

Terraform's `dev_overrides` points the CLI at a local binary and skips the registry, so no
`terraform init` is needed and no lock file has to be deleted after a rebuild.

Add the following to `~/.terraformrc`, replacing the path with `go env GOPATH` plus `/bin`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/netactuate/netactuate" = "/home/you/go/bin"
  }
  direct {}
}
```

Build and install the binary, then run Terraform without initialising:

```bash
go build -o $(go env GOPATH)/bin/terraform-provider-netactuate .
cd examples/pop/01-vm-s3
terraform plan
```

Terraform prints a warning that dev_overrides are active, which is expected. The `direct {}`
block keeps every other provider resolving from the registry as normal.

### Generating documentation

The registry documentation under `docs/` is generated from the provider schema, the examples
under `examples/`, and the templates under `templates/`:

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs generate --provider-name netactuate
```

## License

Mozilla Public License 2.0. See [LICENSE.md](LICENSE.md).
