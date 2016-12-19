#! /bin/bash

echo "Create Monitoring"
kubectl create -f ./prometheus-manifests.yml

echo ""
echo "Create go-ipfs deployment"
kubectl create -f ./go-ipfs-deployment.yml

echo ""
echo "Create js-ipfs deployment"
kubectl create -f ./js-ipfs-deployment.yml
