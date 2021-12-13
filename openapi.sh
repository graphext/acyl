#!/bin/bash

set -e

if [[ $(which npm) == "" ]]; then
    echo "npm not installed"
    exit 1
fi
npm list -g | grep openapitools/openapi-generator-cli || npm install -g @openapitools/openapi-generator-cli
openapi-generator-cli generate -g html2 -i doc/generate/openapi.yml -o ui/apidocs/ --generate-alias-as-model
