package ghclient

import (
	"context"
	"golang.org/x/sync/errgroup"
	"os"
	"strconv"
	"testing"
)

const (
	testingRepo = "dollarshaveclub/acyl"
	testingPath = ".helm/charts/acyl"
	testingRef  = "master"
)

var token = os.Getenv("GITHUB_TOKEN")
var parallelism = os.Getenv("TEST_PARALLELISM")

func TestGetDirectoryContents(t *testing.T) {
	if token == "" || os.Getenv("CIRCLECI") == "true" {
		t.Skip()
	}
	pcnt, err := strconv.Atoi(parallelism)
	if err != nil || pcnt < 1 || pcnt > 1000 {
		pcnt = 5
	}
	t.Logf("running with %v calls in parallel...", pcnt)
	ghc := NewGitHubClient(token)
	eg := errgroup.Group{}
	for i := 0; i < pcnt; i++ {
		eg.Go(func() error {
			_, err := ghc.GetDirectoryContents(context.Background(), testingRepo, testingPath, testingRef)
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		t.Fatalf("failed: %v", err)
	}
}
