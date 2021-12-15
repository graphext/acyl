#!/bin/bash

# Port forwards an Acyl pod in a local minikube environment and tails logs

NS=$(kubectl get ns --field-selector=status.phase=Active --output=jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' |grep -e '^nitro\-')
if [[ -z "${NS}" || $(echo "${NS}" |wc -l) -ne 1 ]]; then
  echo "empty or multiple matching namespaces: ${NS}"
  exit 1
fi

POD=$(kubectl -n "${NS}" get pods --selector=appsel=acyl --field-selector=status.phase=Running --output=jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
if [[ -z "${POD}" || $(echo "${POD}" |wc -l) -ne 1 ]]; then
  echo "empty or multiple matching pods: ${POD}"
  exit 1
fi

kubectl -n "${NS}" port-forward "${POD}" 4000:4000 &
kubectl -n "${NS}" logs -f "${POD}"