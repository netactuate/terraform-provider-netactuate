# Terraform Provider NetActuate

## Usage

Currently in this stage of development you'll need both gona-dev and terraform-netactuate-provider from this repo.
the go.mod replace expects gona to be ../ from terraform-netactuate-provider.

Basic Steps to test:
```
git clone gona-dev
git clone terraform-netactuate-provider

cd terraform-netactuate-provider
make deps
make install-all

cd examples/basic
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

### Run locally
Do the following to run and test the TF provider locally:
1. Install the dependencies:
    ```bash
    make deps 
    ```
2. Compile and install the TF provider's binaries to the local TF plugins directory:
    ```bash
    make install-all
    ```
3. Install TF providers for the test [example](examples/basic):
    ```bash
    cd example
    terraform init
    ```
   Every time the provider is re-built, `.terraform.lock.hcl` file must be removed and the
   test example modules re-initialize, because the provider dependency's hash changes
4. Build the infrastructure:
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
