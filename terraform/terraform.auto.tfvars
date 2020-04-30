job_name                   = "dev-ipfs"
environment                = "dev-ipfs"

terraform {
  backend "s3" {
    bucket         = "ipfs-cluster-terraform-state"
    key            = "dev-ipfs.tfstate"
    region         = "eu-west-1"

    dynamodb_table = "terragrunt_locks_ipfs_cluster"

    encrypt        = true
  }
}

# ---

// this domain must exist in the configuration of the cloud provider - for AWS, this means adding a hosted zone to Route 53
domain                     = "ctlplane.io"
hosts                      = 3
hostname_format            = "kube-a-%d"

// these SSH keys are added to every node
ssh_authorized_keys_github = [
  "hsanjuan"
]

/* digitalocean */

# retrieve ssh_keys with `doctl compute ssh-key list`
digitalocean_ssh_keys      = [
  "19585644"
]
digitalocean_region        = "lon1"
digitalocean_size          = "16gb"

/* aws dns */
aws_region                 = "eu-west-1"
