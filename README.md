# Terraform Provider NetActuate

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
3. Install TF providers for the test [example](example):
    ```bash
    cd example
    terraform init
    ```
   Every time the provider is re-built, [.terraform.lock.hcl](example/.terraform.lock.hcl) file must be removed and the
   test example modules re-initialize, because the provider dependency's hash changes
4. Build the infrastructure:
    ```bash
    terraform apply
    ```
