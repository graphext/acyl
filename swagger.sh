#!/bin/bash

set -e

if [[ $(which npm) == "" ]]; then
    echo "npm not installed"
    exit 1
fi
npm list -g | grep bootprint || npm install -g bootprint
npm list -g | grep bootprint-openapi || npm install -g bootprint-openapi
bootprint openapi doc/swagger/swagger.yml ui/swagger/
