#!/bin/bash

export AWS_ACCESS_KEY_ID="${KUBE_AWS_ACCESS_KEY_ID}"
export AWS_SECRET_ACCESS_KEY="${KUBE_AWS_SECRET_ACCESS_KEY}"

gunzip ./testing/bin/aws-iam-authenticator.gz && mv ./testing/bin/aws-iam-authenticator /usr/local/bin/
gunzip ./testing/bin/kubectl.gz && mv ./testing/bin/kubectl /usr/local/bin/
chmod a+x /usr/local/bin/kubectl /usr/local/bin/aws-iam-authenticator

mkdir -p ~/.kube
sed -e "s|{{CERT}}|${KUBE_CERT_DATA}|g; s|{{KUBE_API_ENDPOINT}}|${KUBE_API_ENDPOINT}|g" < ./testing/circle-ci-kube.config > ~/.kube/kube.config

export KUBECONFIG=~/.kube/kube.config

JOBNAME="acyl-integration-$RANDOM"
TAG="${CIRCLE_SHA1}-integration"
sed -e "s/{{TAG}}/${TAG}/g; s/{{NAME}}/${JOBNAME}/g" < ./testing/integration-test-job.yaml > ~/job.yaml

set -x

kubectl config set-context acyl-nitro-integration
kubectl version
kubectl apply -f ~/job.yaml
sleep 1
kubectl wait "job/${JOBNAME}" --for condition=Complete --timeout=300s
SUCCESS=$?
kubectl logs "job/${JOBNAME}"
kubectl delete "job/${JOBNAME}"

# clean up any lingering namespaces
kubectl get ns --no-headers=true |grep nitro- |awk '{print $1}' |xargs kubectl delete --wait=false ns

exit $SUCCESS
