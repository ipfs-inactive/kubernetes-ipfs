#! /bin/bash

set -e

# echo "Create Monitoring"
kubectl create -f ./prometheus-manifests.yml

echo
echo "Create go-ipfs deployment"
kubectl create -f ./ipfs-cluster-deployment.yml

sleep 2

echo
echo "Waiting for all containers to be running"

while true; do
    sleep 10
    statuses=`kubectl get pods -l 'app=ipfs-cluster' -o jsonpath='{.items[*].status.phase}' | xargs -n1`
    echo $statuses
    all_running="yes"
    for s in $statuses; do
        if [[ "$s" != "Running" ]]; then
            all_running="no"
        fi
    done
    if [[ $all_running == "yes" ]]; then
        break
    fi
done

sleep 5
if [[ $1 == "quick" ]]; then
    exit 0;
fi

echo
echo "Adding peers to cluster"
pods=`kubectl get pods -l 'app=ipfs-cluster,role=peer' -o jsonpath='{.items[*].metadata.name}' | xargs -n1`
bootstrapper=`kubectl get pods -l 'app=ipfs-cluster,role=bootstrapper' -o jsonpath='{.items[*].metadata.name}'`

for p in $pods; do
    addr=$(echo "/ip4/"`kubectl get pods $p -o jsonpath='{.status.podIP}'`"/tcp/9096/ipfs/"`kubectl exec $p -- ipfs-cluster-ctl --enc json id | jq -r .id`)
    kubectl exec $bootstrapper -- ipfs-cluster-ctl peers add "$addr"
done

set +ex
echo
echo "To access Grafana for viewing metrics gathered by Prometheus, run the following commands:"
echo
echo $'pod=$(kubectl get pods --namespace=monitoring | grep grafana-core | awk \'{print $1}\')'
echo 'kubectl port-forward --namespace=monitoring $pod 3000:3000'
echo
echo "Then navigate to localhost:3000 in your browser."
