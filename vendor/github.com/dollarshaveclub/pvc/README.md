# PVC
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]
[![CircleCI](https://circleci.com/gh/dollarshaveclub/pvc.svg?style=svg)](https://circleci.com/gh/dollarshaveclub/pvc)

[godocs]: https://pkg.go.dev/github.com/dollarshaveclub/pvc

PVC (polyvinyl chloride) is a simple, generic secret retrieval library that supports
multiple backends.

PVC lets applications access secrets without caring too much about where they
happen to be stored. The use case is to allow secrets to come from local/insecure
backends during development and testing, and then from Vault in production without
significant code changes required.

## Backends

- [Vault KV Version 1](https://www.vaultproject.io/docs/secrets/kv)
- Environment variables
- JSON file
- File Tree (local filesystem, one file per secret)

## Secret Values

PVC makes some assumptions about how your secrets are stored in the various backends:

- If using Vault, there must be exactly one key called "value" for any given secret path (this can be overridden with 
`WithVaultValueKey("foo")`). The data associated with the value key will be retrieved and returned literally to the 
client as a byte slice. Binary values must be Base64-encoded.
- If using JSON or environment variables, the value will be treated as a string and returned as a byte slice. Binary values
should be Base64-encoded (same as Vault).
- If using the file tree backend, you must supply an absolute root path which will be combined with the secret ID (after
mapping). This file path will be read as the secret contents.

## File Tree
This is intended to be useful for local development secrets in the filesystem, or using 
the [Vault Sidecar Injector](https://www.vaultproject.io/docs/platform/k8s/injector).

Example:

    Given the following:
    
    -> Root Path: /vault/secrets
    -> Secret ID: webservice/production/db/password
    -> Mapping: {{ .ID }}.txt
    
    PVC would attempt to read this file when Get() is called:
    
    /vault/secrets/webservice/production/db/password.txt

## Vault Authentication

The Vault backend supports token, Kubernetes, and AppRole authentication.

## Example

```go
package main

import (
	"fmt"

	"github.com/dollarshaveclub/pvc"
)

func main() {

	// environment variable backend
	sc, _ := pvc.NewSecretsClient(pvc.WithEnvVarBackend(), pvc.WithMapping("SECRET_MYAPP_{{ .ID }}"))
	secret, _ := sc.Get("foo") // fetches the env var "SECRET_MYAPP_FOO"

	// JSON file backend
	sc, _ = pvc.NewSecretsClient(pvc.WithJSONFileBackend("secrets.json"))
	secret, _ = sc.Get("foo") // fetches the value in secrets.json under the key "foo"

	fmt.Printf("foo: %v\n", string(secret))

	// Vault backend
	sc, _ = pvc.NewSecretsClient(
		pvc.WithVaultBackend(pvc.TokenVaultAuth, "http://vault.example.com:8200"),
		pvc.WithVaultToken("some token"),
		pvc.WithMapping("secret/development/{{ .ID }}"))
	secret, _ = sc.Get("foo") // fetches the value from Vault (using token auth) from path secret/development/foo

	fmt.Printf("foo: %v\n", string(secret))

	// Automatic struct filling
	type Secrets struct {
		Username      string `secret:"secret/username"` // secret id: secret/username
		Password      string `secret:"secret/password"`
		EncryptionKey []byte `secret:"secret/enc_key"` // fields can be strings or byte slices
	}

	secrets := Secrets{}

	// Fill automatically fills the fields in the secrets struct that have "secret" tags
	err := sc.Fill(&secrets)
	if err != nil {
		panic(err)
	}

	fmt.Printf("my username is: %v\n", secrets.Username)
	fmt.Printf("my password is: %v\n", secrets.Password)
	fmt.Printf("my key length is %d\n", len(secrets.EncryptionKey))
}
```

See also `example/`
