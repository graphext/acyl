package pvc

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/vault/api"
)

// Default mapping for this backend
const (
	DefaultVaultMapping = "secret/{{ .ID }}"
)

// VaultAuthentication enumerates the supported Vault authentication methods
type VaultAuthentication int

// Supported Vault authentication methods
const (
	UnknownVaultAuth VaultAuthentication = iota // Unknown/unset
	TokenVaultAuth                              // Token authentication
	AppRoleVaultAuth                            // AppRole
	K8sVaultAuth                                // Kubernetes
)

type vaultBackendGetter struct {
	vc     vaultIO
	mapper SecretMapper
	config *vaultBackend
}

func newVaultBackendGetter(vb *vaultBackend, vc vaultIO) (*vaultBackendGetter, error) {
	var err error
	if vb.host == "" {
		return nil, fmt.Errorf("Vault host is required")
	}
	switch vb.authentication {
	case TokenVaultAuth:
		err = vc.TokenAuth(vb.token)
		if err != nil {
			return nil, fmt.Errorf("error authenticating with supplied token: %v", err)
		}
	case AppRoleVaultAuth:
		return nil, fmt.Errorf("AppRole authentication not implemented")
	case K8sVaultAuth:
		err = vc.K8sAuth(vb.k8sjwt, vb.roleid)
		if err != nil {
			return nil, fmt.Errorf("error performing Kubernetes authentication: %v", err)
		}
	default:
		return nil, fmt.Errorf("unknown authentication method: %v", vb.authentication)
	}
	if vb.mapping == "" {
		vb.mapping = DefaultVaultMapping
	}
	sm, err := newSecretMapper(vb.mapping)
	if err != nil {
		return nil, fmt.Errorf("error with mapping: %v", err)
	}
	return &vaultBackendGetter{
		vc:     vc,
		mapper: sm,
		config: vb,
	}, nil
}

func (vbg *vaultBackendGetter) Get(id string) ([]byte, error) {
	path, err := vbg.mapper.MapSecret(id)
	if err != nil {
		return nil, fmt.Errorf("error mapping id to path: %v", err)
	}
	v, err := vbg.vc.GetValue(path)
	if err != nil {
		return nil, fmt.Errorf("error reading value: %v", err)
	}
	return v, nil
}

// vaultIO describes an object capable of interacting with Vault
type vaultIO interface {
	TokenAuth(token string) error
	AppRoleAuth(roleid string) error
	K8sAuth(jwt, roleid string) error
	GetValue(path string) ([]byte, error)
}

// vaultClient is the concrete implementation of vaultIO interacting with a real Vault server
type vaultClient struct {
	client *api.Client
	config *vaultBackend
	token  string
}

var _ vaultIO = &vaultClient{}

type vaultClientFactory func(config *vaultBackend) (vaultIO, error)

// Allow tests to override and supply a fake vault client
var getVaultClient vaultClientFactory = newVaultClient

// newVaultClient returns a vaultClient object or error
func newVaultClient(config *vaultBackend) (vaultIO, error) {
	vc := vaultClient{}
	c, err := api.NewClient(&api.Config{Address: config.host})
	vc.client = c
	vc.config = config
	return &vc, err
}

// tokenAuth sets the client token but doesn't check validity
func (c *vaultClient) TokenAuth(token string) error {
	c.token = token
	c.client.SetToken(token)
	ta := c.client.Auth().Token()
	var err error
	for i := 0; i <= int(c.config.authRetries); i++ {
		_, err = ta.LookupSelf()
		if err == nil {
			break
		}
		log.Printf("Token auth failed: %v, retrying (%v/%v)", err, i+1, c.config.authRetries)
		time.Sleep(time.Duration(c.config.authRetryDelaySecs) * time.Second)
	}
	if err != nil {
		return fmt.Errorf("error performing auth call to Vault (retries exceeded): %v", err)
	}
	return nil
}

func (c *vaultClient) getTokenAndConfirm(route string, payload interface{}) error {
	var resp *api.Response
	var err error
	for i := 0; i <= int(c.config.authRetries); i++ {
		req := c.client.NewRequest("POST", route)
		jerr := req.SetJSONBody(payload)
		if jerr != nil {
			return fmt.Errorf("error setting auth JSON body: %v", jerr)
		}
		resp, err = c.client.RawRequest(req)
		if err == nil {
			break
		}
		log.Printf("auth failed: %v, retrying (%v/%v)", err, i+1, c.config.authRetries)
		time.Sleep(time.Duration(c.config.authRetryDelaySecs) * time.Second)
	}
	if err != nil {
		return fmt.Errorf("error performing auth call to Vault (retries exceeded): %v", err)
	}

	var output interface{}
	jd := json.NewDecoder(resp.Body)
	err = jd.Decode(&output)
	if err != nil {
		return fmt.Errorf("error unmarshaling Vault auth response: %v", err)
	}
	body := output.(map[string]interface{})
	auth := body["auth"].(map[string]interface{})
	c.token = auth["client_token"].(string)
	return nil
}

func (c *vaultClient) AppRoleAuth(roleid string) error {
	return nil
}

func (c *vaultClient) K8sAuth(jwt, roleid string) error {
	payload := struct {
		JWT  string `json:"jwt"`
		Role string `json:"role"`
	}{
		JWT:  jwt,
		Role: roleid,
	}
	if c.config.k8sauthpath == "" {
		c.config.k8sauthpath = "kubernetes"
	}
	return c.getTokenAndConfirm(fmt.Sprintf("/v1/auth/%v/login", c.config.k8sauthpath), &payload)
}

var DefaultVaultValueKey = "value"

// getValue retrieves value at path
func (c *vaultClient) getValue(path string) (interface{}, error) {
	c.client.SetToken(c.token)
	lc := c.client.Logical()
	s, err := lc.Read(path)
	if err != nil {
		return nil, fmt.Errorf("error reading secret from Vault: %v: %v", path, err)
	}
	if s == nil {
		return nil, fmt.Errorf("secret not found")
	}
	key := DefaultVaultValueKey
	if c.config.valuekey != "" {
		key = c.config.valuekey
	}
	if _, ok := s.Data[key]; !ok {
		return nil, fmt.Errorf("secret missing value key: %v", key)
	}
	return s.Data[key], nil
}

// GetValue retrieves a value
func (c *vaultClient) GetValue(path string) ([]byte, error) {
	val, err := c.getValue(path)
	if err != nil {
		return nil, err
	}
	switch val := val.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	default:
		return nil, fmt.Errorf("unexpected type for %v value: %T", path, val)
	}
}
