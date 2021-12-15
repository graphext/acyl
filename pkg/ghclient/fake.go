package ghclient

import "context"

// FakeRepoClient is fake implementaiton of RepoClient that runs user-supplied functions for each method
type FakeRepoClient struct {
	GetBranchFunc                 func(context.Context, string, string) (BranchInfo, error)
	GetBranchesFunc               func(context.Context, string) ([]BranchInfo, error)
	GetTagsFunc                   func(context.Context, string) ([]BranchInfo, error)
	SetStatusFunc                 func(context.Context, string, string, *CommitStatus) error
	GetPRStatusFunc               func(context.Context, string, uint) (string, error)
	GetCommitMessageFunc          func(context.Context, string, string) (string, error)
	GetFileContentsFunc           func(ctx context.Context, repo string, path string, ref string) ([]byte, error)
	GetDirectoryContentsFunc      func(ctx context.Context, repo, path, ref string) (map[string]FileContents, error)
	GetUserAppInstallationsFunc   func(ctx context.Context) (AppInstallations, error)
	GetUserAppReposFunc           func(ctx context.Context, instID int64) ([]string, error)
	GetUserFunc                   func(ctx context.Context) (string, error)
	GetUserAppRepoPermissionsFunc func(ctx context.Context, instID int64) (map[string]AppRepoPermissions, error)
	GetRepoArchiveFunc            func(ctx context.Context, repo, ref string) (string, error)
}

var _ RepoClient = &FakeRepoClient{}
var _ GitHubAppInstallationClient = &FakeRepoClient{}

func (frc *FakeRepoClient) GetBranch(ctx context.Context, repo string, branch string) (BranchInfo, error) {
	if frc.GetBranchFunc != nil {
		return frc.GetBranchFunc(ctx, repo, branch)
	}
	return BranchInfo{}, nil
}
func (frc *FakeRepoClient) GetBranches(ctx context.Context, repo string) ([]BranchInfo, error) {
	if frc.GetBranchesFunc != nil {
		return frc.GetBranchesFunc(ctx, repo)
	}
	return []BranchInfo{}, nil
}
func (frc *FakeRepoClient) GetTags(ctx context.Context, repo string) ([]BranchInfo, error) {
	if frc.GetTagsFunc != nil {
		return frc.GetTagsFunc(ctx, repo)
	}
	return []BranchInfo{}, nil
}
func (frc *FakeRepoClient) SetStatus(ctx context.Context, repo string, sha string, status *CommitStatus) error {
	if frc.SetStatusFunc != nil {
		return frc.SetStatusFunc(ctx, repo, sha, status)
	}
	return nil
}
func (frc *FakeRepoClient) GetPRStatus(ctx context.Context, repo string, pr uint) (string, error) {
	if frc.GetPRStatusFunc != nil {
		return frc.GetPRStatusFunc(ctx, repo, pr)
	}
	return "", nil
}
func (frc *FakeRepoClient) GetCommitMessage(ctx context.Context, repo string, sha string) (string, error) {
	if frc.GetCommitMessageFunc != nil {
		return frc.GetCommitMessageFunc(ctx, repo, sha)
	}
	return "", nil
}
func (frc *FakeRepoClient) GetFileContents(ctx context.Context, repo string, path string, ref string) ([]byte, error) {
	if frc.GetFileContentsFunc != nil {
		return frc.GetFileContentsFunc(ctx, repo, path, ref)
	}
	return []byte{}, nil
}
func (frc *FakeRepoClient) GetDirectoryContents(ctx context.Context, repo, path, ref string) (map[string]FileContents, error) {
	if frc.GetDirectoryContentsFunc != nil {
		return frc.GetDirectoryContentsFunc(ctx, repo, path, ref)
	}
	return map[string]FileContents{}, nil
}

func (frc *FakeRepoClient) GetUserAppInstallations(ctx context.Context) (AppInstallations, error) {
	if frc.GetUserAppInstallationsFunc != nil {
		return frc.GetUserAppInstallationsFunc(ctx)
	}
	return AppInstallations{AppInstallation{ID: 1}}, nil
}

func (frc *FakeRepoClient) GetUserAppRepos(ctx context.Context, instID int64) ([]string, error) {
	if frc.GetUserAppReposFunc != nil {
		return frc.GetUserAppReposFunc(ctx, instID)
	}
	return []string{"foo/bar"}, nil
}

func (frc *FakeRepoClient) GetUser(ctx context.Context) (string, error) {
	if frc.GetUserFunc != nil {
		return frc.GetUserFunc(ctx)
	}
	return "johndoe", nil
}

func (frc *FakeRepoClient) GetUserAppRepoPermissions(ctx context.Context, instID int64) (map[string]AppRepoPermissions, error) {
	if frc.GetUserAppRepoPermissionsFunc != nil {
		return frc.GetUserAppRepoPermissionsFunc(ctx, instID)
	}
	return map[string]AppRepoPermissions{}, nil
}

func (frc *FakeRepoClient) GetRepoArchive(ctx context.Context, repo, ref string) (string, error) {
	if frc.GetRepoArchiveFunc != nil {
		return frc.GetRepoArchiveFunc(ctx, repo, ref)
	}
	return "foo.tar.gz", nil
}

type FakeRepoAppClient struct {
	GetInstallationTokenForRepoFunc func(ctx context.Context, instID int64, reponame string) (string, error)
}

func (fra *FakeRepoAppClient) GetInstallationTokenForRepo(ctx context.Context, instID int64, reponame string) (string, error) {
	if fra.GetInstallationTokenForRepoFunc != nil {
		return fra.GetInstallationTokenForRepoFunc(ctx, instID, reponame)
	}
	return "asdf", nil
}
