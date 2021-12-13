package pvc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Default mapping for this backend
const (
	DefaultFileTreeMapping  = "{{ .ID }}"
	DefaultFileTreeRootPath = "/vault/secrets"
)

// MaxFileTreeSizeBytes indicates the maximum file size we will read
var MaxFileTreeFileSizeBytes int64 = 2_000_000 // 2 MB

type fileTreeBackendGetter struct {
	mapper   SecretMapper
	config   *fileTreeBackend
	rootPath string
}

func newFileTreeBackendGetter(ft *fileTreeBackend) (*fileTreeBackendGetter, error) {
	if ft.mapping == "" {
		ft.mapping = DefaultFileTreeMapping
	}
	sm, err := newSecretMapper(ft.mapping)
	if err != nil {
		return nil, fmt.Errorf("file tree error with mapping: %v", err)
	}
	if ft.rootPath == "" {
		ft.rootPath = DefaultFileTreeRootPath
	}
	return &fileTreeBackendGetter{
		mapper: sm,
		config: ft,
	}, nil
}

func (ftg *fileTreeBackendGetter) Get(id string) ([]byte, error) {
	key, err := ftg.mapper.MapSecret(id)
	if err != nil {
		return nil, fmt.Errorf("error mapping secret id to filetree path: %v", err)
	}
	secretFilePath := filepath.Join(ftg.config.rootPath, key)
	if !filepath.IsAbs(secretFilePath) {
		return nil, fmt.Errorf("filetree path must be absolute: %v", secretFilePath)
	}
	f, err := os.Open(secretFilePath)
	if err != nil {
		return nil, fmt.Errorf("file tree error opening file %v: %v", secretFilePath, err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file stat: %v", err)
	}
	size := stat.Size()
	if size > MaxFileTreeFileSizeBytes {
		return nil, fmt.Errorf("file too large (max: %v bytes): %v", MaxFileTreeFileSizeBytes, size)
	}
	c, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	return c, nil
}
