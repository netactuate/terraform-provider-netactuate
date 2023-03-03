# How to use Terraform with NetActuate

Terraform is an "infrastructure as code" tool that allows you to define and provision infrastructure resources in a repeatable and automated way. This guide will walk you through the steps required to use Terraform with NetActuate.



## Prerequisites

Before we can get started, you will need the following prerequisites installed:

Terraform (https://www.terraform.io/downloads.html)

NetActuate account (https://www.netactuate.com/) with your API key from the portal.



> Your API settings for your account in the portal will need to have the API checkmarks for allowing VM controls through the API.

## Establish Terraform working directory for your infrastructure

With terraform installed we now need to create a directory for our infrastructure configuration to live in, and add our API key as an Enviornment Variable:

```
export NETACTUATE_API_KEY="YOUR_API_KEY"
mkdir terraform
cd terraform
```

> You can ultimately name the configurations anything you want to separate your resource definitions.  


## Set up the NetActuate provider (versions.tf):

To use Terraform with NetActuate, you will need to set up the NetActuate provider. This entails creating a few configuration files for terraform's initialization. ( versions.tf , main.tf , outputs.tf )

We first need tell terraform which provider to use from the Hashicorp Terraform Registry by adding the following code block to the file: versions.tf

```
terraform {
  required_providers {
    netactuate = {
      source = "netactuate/netactuate"
    }
  }
}
```

Set up the infrastructure (main.tf):
Next we need to configure terraform with any additional provider parameters we might need. Create a new file called main.tf and add the following code snippet:

> Since we added our API Key as an ENV Var , we can leave the api_key section commented out.

```
provider "netactuate" {
  // Optional: API key can be set here, rather than as an environment variable
  // api_key = "NETACTUATE_API_KEY"
}
```

Now that we've configured Terraform to use the NetActuate provider and established the provider block, we can now start to define our new infrastructure in main.tf under the provider block:


## Define SSH Key to use for server login
```
resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 REDACTED_SSH_KEY user@email.test"
}
```

## Define locations, OS, and instance sizing
```
resource "netactuate_server" "server" {
  hostname    = "terraform.example.com"
  plan        = "VR1x1x25"
  location    = "SJC"
  image       = "Ubuntu 22.04 (20221110)"
  ssh_key_id  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE
}
```
> Make sure to edit: 
> 1. your SSH Key 
> 2. your package_billing_contract_id
> 3. Billing defaults to usage , so you will need a contract ID in place.
> 4. If you do NOT have a usage billing contract - 
> 5. Replace package_billing_contract_id with the package_billing and package_billing_opt_in variables.


Here we have defined an SSH key to use for logging into the server, and we created the configuration for: 1 ubuntu server in San Jose, California, with the plan sizing: VR1x1x25

These 2 files are all you will need to spawn a server using Terraform, however we can also modify the standard output from the provisioning process by adding an additonal file: outputs.tf  

## Create Additional Outputs
We can print some of the additional information to the screen by adding the following to

```
output "ipv4" {
  value       = netactuate_server.server.primary_ipv4
  description = "The primary IPv4 address of the VM."
}

output "ipv6" {
  value       = netactuate_server.server.primary_ipv6
  description = "The primary IPv6 address of the VM."
}
```

## Spawn your configured resources:


Now that we have terraform configuration files established we can initialize terraform and show the deployment plan:

```
terraform init
terraform plan
```

Once  you have reviewed the plan, we can now deploy our resources:

```
terraform apply
```



And finally, to destroy all the provisioned resources in the terraform configuration/statefile:

```
terraform destroy
```

If you incur error messages and need full debug output:

```
export TF_LOG=DEBUG

export NA_API_DEBUG=1
```


You can find more examples of Terraform configuration files on GitHub (You can find the provider source code and additional documentation on GitHub (https://github.com/netactuate/terraform-provider-netactuate/).

 
