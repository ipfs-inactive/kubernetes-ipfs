#! /bin/bash

echo "Create Monitoring"
kubectl create -f ./prometheus-manifests.yml

echo "Create go-ipfs deployment"
kubectl create -f ./go-ipfs-deployment.yml

