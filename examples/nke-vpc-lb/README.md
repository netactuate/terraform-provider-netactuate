# NKE with the VPC load balancer add-on

Creates a VPC, an NKE cluster in it and the VPC load balancer add-on, then exposes nginx two ways: a LoadBalancer Service (`app.yaml`) and an Ingress (`ingress.yaml`), each on its own public floating IP.

## Steps

1. Set your API key and run Terraform. `contract_id` is your NKE usage contract ID.

   ```shell
   export NETACTUATE_API_KEY="..."
   terraform init
   terraform apply -var contract_id=12345
   ```

2. Write out the kubeconfig and deploy the app:

   ```shell
   terraform output -raw kubeconfig > kubeconfig.yaml
   export KUBECONFIG=$PWD/kubeconfig.yaml
   kubectl apply -f app.yaml -f ingress.yaml
   ```

   The kubeconfig is a cluster-admin token valid for `kubeconfig_expiration_seconds` (default 7 days). The first `terraform apply` after it expires issues a new one; write it out again then.

3. Wait for the addresses, then test them:

   ```shell
   kubectl get svc web          # EXTERNAL-IP changes from <pending> to a floating IP
   kubectl get ingress web      # ADDRESS shows the Ingress's floating IP
   curl http://<EXTERNAL-IP>/
   curl -H 'Host: web.example.com' http://<ADDRESS>/
   ```

4. Read the addresses through Terraform:

   ```shell
   terraform apply -refresh-only -var contract_id=12345
   terraform output lb_addresses
   ```

## What makes it work

- The Service sets `type: LoadBalancer` and `loadBalancerClass: netactuate.com/vpc-lb`. Without the class, `EXTERNAL-IP` stays `<pending>`.
- The Ingress sets `ingressClassName: netactuate`, and its backend is a NodePort Service.

## Clean up

```shell
terraform destroy -var contract_id=12345
```

Destroying the cluster also releases the floating IPs the add-on assigned.
