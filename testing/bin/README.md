## Test Binaries
This directory contains vendored binaries for use in CircleCI

#### Upgrade Kubectl Binary
1. rename current gzip'd binary: `$ mv kubectl.gz kubectl_<x.xx.x>.gz`
2. download desired linuz binary: `$ curl -LO https://dl.k8s.io/release/<v1.xx.x>/bin/linux/amd64/kubectl`
3. gzip the downloaded kubectl binary: `$ gzip kubectl`
4. commit the binary and confirm works before deleting the old binary: `$ rm kubectl_<x.xx.x>.gz`
