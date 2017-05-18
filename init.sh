#! /bin/bash

set -ex

echo "Create Monitoring"
kubectl create -f ./prometheus-manifests.yml

echo ""
echo "Create go-ipfs deployment"
kubectl create -f ./go-ipfs-deployment.yml

set +ex
echo
echo "To access Grafana for viewing metrics gathered by Prometheus, run the following commands:"
echo
echo $'pod=$kubectl get pods --namespace=monitoring | grep grafana-core | awk \'{print $1}\''
echo 'kubectl port-forward --namespace=monitoring $pod 3000:3000'
echo 
echo 'Then navigate to localhost:3000 in your browser.'
