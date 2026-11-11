# NKE in a VPC with a NodePort Service

Creates a VPC and an NKE cluster in it, then exposes nginx on a floating IP through the VPC network
load balancer, without the load balancer add-on. Use the `nke-vpc-lb` example for the add-on.

## Steps

1. Create the VPC, the cluster and the load balancer. `contract_id` is your NKE usage contract ID.

   ```shell
   export NETACTUATE_API_KEY="..."
   terraform init
   terraform apply -var contract_id=12345
   ```

2. Deploy the app:

   ```shell
   terraform output -raw kubeconfig > kubeconfig.yaml
   export KUBECONFIG=$PWD/kubeconfig.yaml
   kubectl apply -f app.yaml
   ```

   The kubeconfig is a cluster-admin token valid for `kubeconfig_expiration_seconds` (default 7 days). The first `terraform apply` after it expires issues a new one; write it out again then.

3. Test it:

   ```shell
   curl "$(terraform output -raw url)"
   ```

The load balancer's backends come from `data.netactuate_nke_worker_private_ips`. When you change `minimum_nodes` or `maximum_nodes`, the same `terraform apply` updates them; when workers are replaced or the cluster autoscales, the next `terraform apply` does.

## Clean up

```shell
terraform destroy -var contract_id=12345
```
