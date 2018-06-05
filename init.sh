#!/bin/bash

set -euo pipefail

main()  {
  echo "Create Monitoring"
  kubectl apply -f ./prometheus-manifests.yml

  echo
  echo "Create go-ipfs deployment"
  kubectl delete -f ./go-ipfs-deployment.yml || true
  sleep 3
  kubectl apply -f ./go-ipfs-deployment.yml

  echo
  echo "To access Grafana for viewing metrics gathered by Prometheus, run the following commands:"
  echo
  echo $'pod=$(kubectl get pods --namespace=monitoring | grep grafana-core | awk \'{print $1}\')'
  echo 'kubectl port-forward --namespace=monitoring $pod 3000:3000'
  echo
  echo "Then navigate to localhost:3000 in your browser."
  echo
  echo "Awaiting go-ipfs startup..."
  kwaitpod go-ipfs

  echo "Pod is started, waiting for readiness..."
  kwaitpod-ready go-ipfs
}

kwaitpod() {
  local POD="${1}"
  local TIMEOUT=30
  local COUNT=1
  while [[ $(kubectl get pods --all-namespaces --field-selector=status.phase=Running -o name \
    | grep -E "^pods?/${POD}" --count) -eq 0 ]]; do

    let COUNT=$((COUNT + 1))
    [[ "$COUNT" -gt "${TIMEOUT}" ]] && {
      echo "Timeout waiting for pods matching ${POD}"
      exit 1
    }
    printf '.'
    sleep 2
  done
  printf "\nWait complete in ${COUNT} seconds\n" 1>&2
  sleep 3
}

kwaitpod-ready() {
  local POD="${1}"
  local TIMEOUT=30
  local COUNT=1
  while kubectl get pods --all-namespaces --field-selector=status.phase=Running -o json \
    | jq '.items[]? | "\(.metadata.name) readyStatus:\(.status.containerStatuses[]? .ready)"' -r \
    | grep -E "^${POD}" \
    | grep -q -E ' readyStatus:false$'; do

    let COUNT=$((COUNT + 1))
    [[ "$COUNT" -gt "${TIMEOUT}" ]] && {
      echo "Timeout waiting for pods matching ${POD} to be ready"
      exit 1
    }
    printf '.'
    sleep 2
  done
  printf "\nWait complete in ${COUNT} seconds\n" 1>&2
  sleep 3
}

main
