# kubernetes terraform

based on https://github.com/hobby-kube/guide

## Config

```bash
export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""

# For DigitalOcean, use either
export DIGITALOCEAN_TOKEN=""
# or
export TF_VAR_digitalocean_token=""
```

In `terraform.auto.tfvars`:

- update `backend "s3"` block to use a unique S3 bucket name
- update `domain` key's value to a domain that is set as a hosted zone in AWS Route53 to resolve DNS
- add GitHub usernames to `ssh_authorized_keys_github`. These keys will be deployed to the cluster members under the `root` user
- add the DigitalOcean SSH keys to write to the hosts
  - this is used to provision kubernetes on the nodes
  - this key should be available to terraform, e.g. in an `ssh-agent`
- region and instance sizes can be modified


Terraform state is stored in S3 with DynamoDB locking.

### Usage

```sh
# fetch the required modules
$ terraform init hobby-kube/

# see what `terraform apply` will do
$ terraform plan hobby-kube/

# execute it
$ terraform apply hobby-kube/
```
