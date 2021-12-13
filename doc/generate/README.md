# OPENAPI References
- https://github.com/OpenAPITools/openapi-generator-cli/tree/master/apps/generator-cli/src
- https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md
---
## Update API Docs
1) Update the openapi.yml configuration file within `/acyl/doc/generate`
2) Run `$ make docs` or from the root of the acyl directory `$ openapi-generator-cli generate -g html2 -i doc/generate/openapi.yml -o ui/apidocs/ --generate-alias-as-model`