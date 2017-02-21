#! /bin/bash

set -ex

echo "Create Monitoring"
kubectl create -f ./prometheus-manifests.yml

echo ""
echo "Create go-ipfs deployment"
kubectl create -f ./go-ipfs-deployment.yml
