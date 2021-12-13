package pvc

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

// SecretsClient is the client that retrieves secret values
type SecretsClient struct {
	backend secretBackend
}

// Get returns the value of a secret from the configured backend
func (sc *SecretsClient) Get(id string) ([]byte, error) {
	if sc.backend == nil {
		return nil, fmt.Errorf("SecretsClient is uninitialized: backend is nil")
	}
	return sc.backend.Get(id)
}

type secretBackend interface {
	Get(id string) ([]byte, error)
}

// SecretDefinition defines a secret and how it can be accessed via the various backends
type SecretDefinition struct {
	ID         string // arbitrary identifier for this secret
	VaultPath  string // path in Vault (no leading slash, eg "secret/foo/bar")
	EnvVarName string // environment variable name
	JSONKey    string // key in JSON object
}

type vaultBackend struct {
	host               string
	authentication     VaultAuthentication
	authRetries        uint
	authRetryDelaySecs uint
	token              string
	k8sjwt             string
	k8sauthpath        string
	roleid             string
	mapping            string
	valuekey           string
}

type envVarBackend struct {
	mapping string
}

type jsonFileBackend struct {
	fileLocation string
	mapping      string
}

type fileTreeBackend struct {
	rootPath string
	mapping  string
}

//go:generate stringer -type=backendType
type backendType int

const (
	unknownBackendType backendType = iota
	vaultBackendType
	envVarBackendType
	jsonBackendType
	fileTreeBackendType
)

type secretsClientConfig struct {
	mapping         string
	backendCount    int
	betype          backendType
	vaultBackend    *vaultBackend
	envVarBackend   *envVarBackend
	jsonFileBackend *jsonFileBackend
	fileTreeBackend *fileTreeBackend
}

// SecretsClientOption defines options when creating a SecretsClient
type SecretsClientOption func(*secretsClientConfig)

// WithMapping sets the template string mapping to determine the location for each secret in the backend. The secret ID will be interpolated as ".ID".
// Example (Vault Backend): "secret/foo/bar/{{ .ID }}".
// Example (Env Var Backend): "MYAPP_SECRET_{{ .ID }}"
// Example (JSON Backend): "{{ .ID }}"
func WithMapping(mapping string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		s.mapping = mapping
	}
}

// WithFileTree enables the FileTreeBackend. With this backend, PVC reads one individual file per secret ID. Sub-paths
// under the root should be implemented with directory separators in the secret ID.
// The path that results from the root path + secret ID mapping will be read as the secret. This must be an absolute
// filesystem path.
func WithFileTreeBackend(rootPath string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		s.betype = fileTreeBackendType
		s.backendCount++
		if s.fileTreeBackend == nil {
			s.fileTreeBackend = &fileTreeBackend{}
		}
		s.fileTreeBackend.rootPath = rootPath
	}
}

// WithVaultBackend enables the Vault backend with the requested authentication type and host (ex: https//my.vault.com:8300)
func WithVaultBackend(auth VaultAuthentication, host string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		s.betype = vaultBackendType
		s.backendCount++
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.authentication = auth
		s.vaultBackend.host = host
	}
}

// WithVaultAuthRetries sets the number of retries if authentication fails (default: 0)
func WithVaultAuthRetries(retries uint) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.authRetries = retries
	}
}

// WithVaultAuthRetryDelay sets the delay in seconds between authentication attempts (default: 0)
func WithVaultAuthRetryDelay(secs uint) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.authRetryDelaySecs = secs
	}
}

// WithVaultToken sets the token to use when using token auth
func WithVaultToken(token string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.token = token
	}
}

// WithVaultK8sAuth sets the Kubernetes JWT and role to use for authentication
func WithVaultK8sAuth(jwt, role string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.k8sjwt = jwt
		s.vaultBackend.roleid = role
	}
}

// WithVaultK8sAuthPath sets the path for the k8s Vault auth backend (defaults to "kubernetes" otherwise)
func WithVaultK8sAuthPath(path string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.k8sauthpath = path
	}
}

// WithVaultRoleID sets the RoleID when using AppRole authentication
func WithVaultRoleID(roleid string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.roleid = roleid
	}
}

func WithVaultValueKey(key string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		if s.vaultBackend == nil {
			s.vaultBackend = &vaultBackend{}
		}
		s.vaultBackend.valuekey = key
	}
}

// WithEnvVarBackend enables the environment variable backend. Any characters in the secret ID that are not alphanumeric ASCII or underscores (legal env var characters) will be replaced by underscores after mapping.
func WithEnvVarBackend() SecretsClientOption {
	return func(s *secretsClientConfig) {
		s.betype = envVarBackendType
		s.backendCount++
	}
}

// WithJSONFileBackend enables the JSON file backend. The file should contain a single JSON object associating a name with a value: { "mysecret": "pa55w0rd"}.
// Path is required and must be a valid path to the JSON file.
func WithJSONFileBackend(path string) SecretsClientOption {
	return func(s *secretsClientConfig) {
		s.betype = jsonBackendType
		s.backendCount++
		if s.jsonFileBackend == nil {
			s.jsonFileBackend = &jsonFileBackend{}
		}
		s.jsonFileBackend.fileLocation = path
	}
}

// NewSecretsClient returns a SecretsClient configured according to the SecretsClientOptions supplied. Exactly one backend must be enabled.
// Weird things will happen if you mix options with incompatible backends.
func NewSecretsClient(ops ...SecretsClientOption) (*SecretsClient, error) {
	config := &secretsClientConfig{}
	for _, op := range ops {
		op(config)
	}
	if config.betype == unknownBackendType || config.backendCount != 1 {
		return nil, fmt.Errorf("exactly one backend must be enabled")
	}
	sc := SecretsClient{}
	switch config.betype {
	case vaultBackendType:
		if config.vaultBackend == nil {
			config.vaultBackend = &vaultBackend{}
		}
		if config.vaultBackend.authentication == UnknownVaultAuth {
			return nil, fmt.Errorf("vault backend requires an authentication type")
		}
		if config.vaultBackend.host == "" {
			return nil, fmt.Errorf("vault host is required")
		}
		config.vaultBackend.mapping = config.mapping
		vc, err := getVaultClient(config.vaultBackend)
		if err != nil {
			return nil, fmt.Errorf("error creating vault client: %v", err)
		}
		vbe, err := newVaultBackendGetter(config.vaultBackend, vc)
		if err != nil {
			return nil, fmt.Errorf("error getting vault backend: %v", err)
		}
		sc.backend = vbe
	case envVarBackendType:
		if config.envVarBackend == nil {
			config.envVarBackend = &envVarBackend{}
		}
		config.envVarBackend.mapping = config.mapping
		ebe, err := newEnvVarBackendGetter(config.envVarBackend)
		if err != nil {
			return nil, fmt.Errorf("error getting env var backend: %v", err)
		}
		sc.backend = ebe
	case jsonBackendType:
		if config.jsonFileBackend == nil {
			config.jsonFileBackend = &jsonFileBackend{}
		}
		if config.jsonFileBackend.fileLocation == "" {
			return nil, fmt.Errorf("json file location is required")
		}
		config.jsonFileBackend.mapping = config.mapping
		jbe, err := newjsonFileBackendGetter(config.jsonFileBackend)
		if err != nil {
			return nil, fmt.Errorf("error getting JSON file backend: %v", err)
		}
		sc.backend = jbe
	case fileTreeBackendType:
		if config.fileTreeBackend == nil {
			config.fileTreeBackend = &fileTreeBackend{}
		}
		if config.fileTreeBackend.rootPath == "" {
			return nil, fmt.Errorf("filetree backend requires a root path")
		}
		if !filepath.IsAbs(config.fileTreeBackend.rootPath) {
			return nil, fmt.Errorf("filetree root path must be absolute: %v", config.fileTreeBackend.rootPath)
		}
		config.fileTreeBackend.mapping = config.mapping
		ftg, err := newFileTreeBackendGetter(config.fileTreeBackend)
		if err != nil {
			return nil, fmt.Errorf("error getting FileTree backend: %v", err)
		}
		sc.backend = ftg
	default:
		return nil, fmt.Errorf("invalid or unknown backend type: %v", config.betype)
	}
	return &sc, nil
}

// SecretMapper maps secrets
type SecretMapper interface {
	MapSecret(id string) (string, error)
}

// secretMapper manages turning secret IDs into a location suitable for a backend to use
type secretMapper struct {
	mappingTmpl *template.Template
}

// newSecretMapper returns a secret mapper using the supplied mapping string
func newSecretMapper(mapping string) (*secretMapper, error) {
	if !strings.Contains(mapping, "{{ .ID") && !strings.Contains(mapping, "{{.ID") {
		return nil, fmt.Errorf("mapping must contain {{ .ID }}")
	}
	tmpl, err := template.New("secret-mapper").Parse(mapping)
	if err != nil {
		return nil, fmt.Errorf("error parsing mapping: %v", err)
	}
	return &secretMapper{
		mappingTmpl: tmpl,
	}, nil
}

// mapSecret maps a secret ID to a location via the mapping string
func (sm *secretMapper) MapSecret(id string) (string, error) {
	d := struct{ ID string }{ID: id}
	b := bytes.Buffer{}
	err := sm.mappingTmpl.Execute(&b, d)
	if err != nil {
		return "", fmt.Errorf("error executing mapping template: %v", err)
	}
	return string(b.Bytes()), nil
}
