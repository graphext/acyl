package ghclient

import (
	"context"
	"os"
	"testing"
)

const (
	testingRepo = "dollarshaveclub/acyl"
)

var token = os.Getenv("GITHUB_TOKEN")

func TestGetDirectoryContents(t *testing.T) {
	if token == "" || os.Getenv("CIRCLECI") == "true" {
		t.Skip()
	}
	ghc := NewGitHubClient(token)
	cm, err := ghc.GetDirectoryContents(context.Background(), testingRepo, "pkg/persistence", "master")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	var i int
	for k, v := range cm {
		t.Logf("path: %v; size: %v\n", k, len(v.Contents))
		if v.Symlink {
			t.Logf("... symlink to: %v", v.SymlinkTarget)
			i++
		}
	}
	t.Logf("symlink count: %v", i)
}
