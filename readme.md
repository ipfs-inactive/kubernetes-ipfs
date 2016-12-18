> Experiment with setting up a Kubernetes
> cluster with go-ipfs for e2e and stress testing

# Get Started

Have golang, minikube and kubectl installed on the machine.

`./reset-minikube.sh` to make sure you have a funky fresh install

`./init.sh` to create go-ipfs and grafana deployments on minikube

`go run main.go tests/simple-add-and-cat.yml` and see it passing in the bottom

# Example Reports

## Simple Add > Cat test with 2 nodes

go-ipfs:v0.4.0 - https://snapshot.raintank.io/dashboard/snapshot/Jk2Ek5pR28xF3vXAeO75thmmA6WBNbR0

go-ipfs:v0.4.4 - https://snapshot.raintank.io/dashboard/snapshot/JVmEIB1Ofac3jRBeOP01XQmdYWEMcxSa

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/JteUjtvn3hRjlGolcmcP2vn5Ud6gYwPW

## Simple Add > Pin test with 5 nodes

go-ipfs:v0.4.5-pre1 - https://snapshot.raintank.io/dashboard/snapshot/DfHSuzIo3TNDrMmJedqmSMvFniGLIWwH

# Manual instructions

## Start local cluster

```
minikube start

# Start Monitoring
kubectl create -f ./prometheus-manifests.yml

# Start go-ipfs deployment
kubectl create -f ./deployment.yml

# Get first pods name
PODNAME=$(kubectl get pods -o=json | jq -r '.items[0].metadata.name')

# Sending commands to pods
kubectl exec "$PODNAME" -- ipfs id
```

## Creating

## Manual Steps

### Create ipfs deployment

```
kubectl run ipfs --image=ipfs/go-ipfs --replicas=1 --port=5001
```

### Create service (expose deployment)

```
kubectl expose deployment ipfs --port=5001 --type=LoadBalancer
```

### Should be able to get ID via API now

```
curl --silent $(minikube service ipfs --url)/api/v0/id | jq .
```

### Create 4 more nodes

```
kubectl scale --replicas=5 deployments/ipfs
```
