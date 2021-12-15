#!/bin/bash

# sets up kubernetes secrets from CircleCI secret env vars

if [[ -z "${GITHUB_TOKEN}" ]]; then
  echo "github token missing from env"
  exit 1
fi

if [[ -z "${GITHUB_APP_HOOK_SECRET}" ]]; then
  echo "hook secret missing from env"
  exit 1
fi

if [[ -z "${GITHUB_APP_ID}" ]]; then
  echo "github app id missing from env"
  exit 1
fi

if [[ -z "${GITHUB_APP_PRIVATE_KEY}" ]]; then
  echo "github app private key missing from env"
  exit 1
fi

set -e

cp ./integration-test-secret.yaml ./secret.yaml
sed -e "s/{GTKVALUE}/${GITHUB_TOKEN}/g" < ./secret.yaml > ./secret2.yaml
sed -e "s/{GAHSVALUE}/${GITHUB_APP_HOOK_SECRET}/g" < ./secret2.yaml > ./secret3.yaml
sed -e "s/{GAIDVALUE}/${GITHUB_APP_ID}/g" < ./secret3.yaml > ./secret4.yaml
sed -e "s/{GAPKVALUE}/${GITHUB_APP_PRIVATE_KEY}/g" < ./secret4.yaml > ./secret5.yaml

rm ./secret.yaml ./secret2.yaml ./secret3.yaml ./secret4.yaml
mv ./secret5.yaml ./secret.yaml

kubectl apply -f ./secret.yaml

rm ./secret.yaml