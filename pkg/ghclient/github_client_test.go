package ghclient

import (
	"context"
	"fmt"
	"github.com/palantir/go-githubapp/githubapp"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	testingRepo = "dollarshaveclub/acyl"
	testingPath = ".helm/charts"
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

var (
	GitHubV3APIURL = "https://api.github.com/"
	GitHubV4APIURL = "https://api.github.com/graphql"
	appIDs         = os.Getenv("TEST_GH_APP_ID")
	instIDs        = os.Getenv("TEST_GH_INST_ID")
	pkeyPEMPath    = os.Getenv("TEST_GH_APP_PKEY")
	webhookSecret  = os.Getenv("TEST_GH_APP_WHSECRET")
)

func TestGetDirectoryContentsGithubApp(t *testing.T) {
	appID, err := strconv.Atoi(appIDs)
	if err != nil {
		t.Fatalf("invalid app id: %v", err)
	}
	instID, err := strconv.Atoi(instIDs)
	if err != nil {
		t.Fatalf("invalid installation id: %v", err)
	}
	privateKeyPEM, err := ioutil.ReadFile(pkeyPEMPath)
	if err != nil {
		t.Fatalf("error reading pem key: %v", err)
	}
	c := githubapp.Config{}
	c.V3APIURL = GitHubV3APIURL
	c.V4APIURL = GitHubV4APIURL
	c.App.IntegrationID = int64(appID)
	c.App.PrivateKey = string(privateKeyPEM)
	c.App.WebhookSecret = webhookSecret
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 60 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 60 * time.Second,
	}
	cc, err := githubapp.NewDefaultCachingClientCreator(c,
		githubapp.WithClientTimeout(60*time.Second),
		githubapp.WithTransport(tr))
	if err != nil {
		t.Fatalf("error creating GH app client creator: %v", err)
	}
	gc, err := cc.NewInstallationClient(int64(instID))
	if err != nil {
		t.Fatalf("error getting app client: %v", err)
	}
	ghc := NewGitHubClient(token)
	ghc.c = gc

	pcnt, err := strconv.Atoi(parallelism)
	if err != nil || pcnt < 1 || pcnt > 1000 {
		pcnt = 5
	}

	t.Logf("running with %v calls in parallel...", pcnt)

	eg := errgroup.Group{}
	for i := 0; i < pcnt; i++ {
		j := i
		eg.Go(func() error {
			fc, err := ghc.GetDirectoryContents(context.Background(), testingRepo, testingPath, testingRef)
			for k, v := range fc {
				if len(v.Contents) == 0 {
					fmt.Printf("EMPTY: %v: %v", j, k)
				}
			}
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		t.Fatalf("failed: %v", err)
	}
}
