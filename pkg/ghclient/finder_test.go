package ghclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	desfacer "gopkg.in/jfontan/go-billy-desfacer.v0"
	git "gopkg.in/src-d/go-git.v4"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	gitcache "gopkg.in/src-d/go-git.v4/plumbing/cache"
	gitobj "gopkg.in/src-d/go-git.v4/plumbing/object"
	gitfs "gopkg.in/src-d/go-git.v4/storage/filesystem"
)

// testRepos creates sample test repos in an in-memory filesystem
// /repo1 - valid git repo with GH SSH remote
// /repo2 - valid git repo with GH HTTPS remote
// /repo3 - valid git repo, no remotes
// /repo4 - invalid git repo
// /somethingelse - just a directory
func testRepos(t *testing.T) afero.Fs {
	afs := afero.NewMemMapFs()
	fs := desfacer.New(afs)
	// repo1
	if err := fs.MkdirAll("repo1", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir repo1: %v", err)
	}
	rfs, err := fs.Chroot("repo1")
	if err != nil {
		t.Fatalf("error in chroot: %v", err)
	}
	if err := rfs.MkdirAll(".git", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir .git: %v", err)
	}
	dot, _ := rfs.Chroot(".git")
	repo, err := git.Init(gitfs.NewStorage(dot, gitcache.NewObjectLRUDefault()), rfs)
	if err != nil {
		t.Fatalf("error initializing repo1: %v", err)
	}
	if _, err := repo.CreateRemote(&gitcfg.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:owner/repo1.git"},
	}); err != nil {
		t.Fatalf("error creating remote for repo1: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("error getting working tree: %v", err)
	}
	err = rfs.MkdirAll("foo", os.ModeDir|os.ModePerm)
	if err != nil {
		t.Fatalf("error in mkdir: %v", err)
	}
	f, err := afs.OpenFile("repo1/foo/bar.txt", os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		t.Fatalf("error writing file: %v", err)
	}
	f.Write([]byte("asdf"))
	f.Close()
	if _, err := wt.Add("foo/bar.txt"); err != nil {
		t.Fatalf("error adding changed file: %v", err)
	}
	_, err = wt.Commit("first commit", &git.CommitOptions{Author: &gitobj.Signature{Name: "someguy", Email: "asdf@asdf.com", When: time.Now().UTC()}})
	if err != nil {
		t.Fatalf("error commiting 1: %v", err)
	}
	// repo2
	if err := fs.MkdirAll("repo2", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir repo2: %v", err)
	}
	rfs, err = fs.Chroot("repo2")
	if err != nil {
		t.Fatalf("error in chroot: %v", err)
	}
	if err := rfs.MkdirAll(".git", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir .git: %v", err)
	}
	dot, _ = rfs.Chroot(".git")
	repo, err = git.Init(gitfs.NewStorage(dot, gitcache.NewObjectLRUDefault()), rfs)
	if err != nil {
		t.Fatalf("error initializing repo2: %v", err)
	}
	if _, err := repo.CreateRemote(&gitcfg.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/owner/repo2.git"},
	}); err != nil {
		t.Fatalf("error creating remote for repo2: %v", err)
	}
	// repo3
	if err := fs.MkdirAll("repo3", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir repo3: %v", err)
	}
	rfs, err = fs.Chroot("repo3")
	if err != nil {
		t.Fatalf("error in chroot: %v", err)
	}
	if err := rfs.MkdirAll(".git", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir .git: %v", err)
	}
	dot, _ = rfs.Chroot(".git")
	repo, err = git.Init(gitfs.NewStorage(dot, gitcache.NewObjectLRUDefault()), rfs)
	if err != nil {
		t.Fatalf("error initializing repo3: %v", err)
	}
	// repo4
	if err := fs.MkdirAll("repo4", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir repo4: %v", err)
	}
	rfs, err = fs.Chroot("repo4")
	if err != nil {
		t.Fatalf("error in chroot: %v", err)
	}
	if err := rfs.MkdirAll(".git", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir .git: %v", err)
	}
	// don't init the repo for repo4 so it's invalid

	// somethingelse
	if err := fs.MkdirAll("somethingelse", os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("error in mkdir somethingelse: %v", err)
	}
	// write a file
	f2, err := fs.Create("somethingelse/foo.txt")
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	f2.Write([]byte("foobar\n"))
	f2.Close()
	return afs
}

func TestFinderFind(t *testing.T) {
	fs := testRepos(t)
	rf := &RepoFinder{
		GitHubHostname: "github.com",
		FSFunc:         func(string) afero.Fs { return fs },
		LF:             t.Logf,
	}
	found, err := rf.Find([]string{""})
	if err != nil {
		t.Fatalf("should have succeded: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("bad count: %v", len(found))
	}
	v, ok := found["owner/repo1"]
	if !ok {
		t.Fatalf("owner/repo1 missing")
	}
	if v != "repo1" {
		t.Fatalf("bad path for repo1: %v", v)
	}
	v, ok = found["owner/repo2"]
	if !ok {
		t.Fatalf("owner/repo2 missing")
	}
	if v != "repo2" {
		t.Fatalf("bad path for repo2: %v", v)
	}
}

func TestFinderLocalDisk(t *testing.T) {
	if os.Getenv("ACYL_FINDER_PATHS") == "" {
		t.SkipNow()
	}
	rf := &RepoFinder{
		GitHubHostname: "github.com",
		FSFunc:         func(string) afero.Fs { return afero.NewOsFs() },
		LF:             t.Logf,
	}
	found, err := rf.Find(strings.Split(os.Getenv("ACYL_FINDER_PATHS"), ","))
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	t.Logf("found repos (%d):\n", len(found))
	for k, v := range found {
		t.Logf("%v:\t%v\n", k, v)
	}
}

func TestFinderRepoInfo(t *testing.T) {
	fs := testRepos(t)
	ri, err := RepoInfo(fs, "repo1", "github.com")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ri.GitHubRepoName != "owner/repo1" {
		t.Fatalf("bad GH name: %v", ri.GitHubRepoName)
	}
	if ri.HeadBranch != "master" {
		t.Fatalf("bad branch: %v", ri.HeadBranch)
	}
}

func TestFinderRepoInfoLocalDisk(t *testing.T) {
	if os.Getenv("ACYL_TEST_FINDER_REPO_INFO") == "" {
		t.SkipNow()
	}
	p, _ := filepath.Abs("../..")
	ri, err := RepoInfo(afero.NewOsFs(), p, "github.com")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if ri.GitHubRepoName != "dollarshaveclub/acyl" {
		t.Fatalf("bad GH name: %v", ri.GitHubRepoName)
	}
	t.Logf("%+v", ri)
}
