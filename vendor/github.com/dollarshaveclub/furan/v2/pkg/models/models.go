package models

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"

	"github.com/dollarshaveclub/furan/v2/pkg/generated/furanrpc"
)

//go:generate stringer -type=BuildStatus

type BuildStatus int

const (
	// Invalid or unknown status
	BuildStatusUnknown BuildStatus = iota
	// Build has been requested but not started yet
	BuildStatusNotStarted
	// Build was requested but determined to be unnecessary
	BuildStatusSkipped
	// Build is currently running in a k8s job
	BuildStatusRunning
	// Build failed or internal error
	BuildStatusFailure
	// Build successfully completed & pushed
	BuildStatusSuccess
	// Build cancellation was requested but build has not yet aborted
	BuildStatusCancelRequested
	// Build was aborted due to cancellation request
	BuildStatusCancelled
)

func (bs BuildStatus) State() furanrpc.BuildState {
	switch bs {
	case BuildStatusUnknown:
		return furanrpc.BuildState_UNKNOWN
	case BuildStatusNotStarted:
		return furanrpc.BuildState_NOTSTARTED
	case BuildStatusSkipped:
		return furanrpc.BuildState_SKIPPED
	case BuildStatusRunning:
		return furanrpc.BuildState_RUNNING
	case BuildStatusFailure:
		return furanrpc.BuildState_FAILURE
	case BuildStatusSuccess:
		return furanrpc.BuildState_SUCCESS
	case BuildStatusCancelRequested:
		return furanrpc.BuildState_CANCEL_REQUESTED
	case BuildStatusCancelled:
		return furanrpc.BuildState_CANCELLED
	default:
		return furanrpc.BuildState_UNKNOWN
	}
}

func BuildStatusFromState(s furanrpc.BuildState) BuildStatus {
	switch s {
	case furanrpc.BuildState_UNKNOWN:
		return BuildStatusUnknown
	case furanrpc.BuildState_NOTSTARTED:
		return BuildStatusNotStarted
	case furanrpc.BuildState_SKIPPED:
		return BuildStatusSkipped
	case furanrpc.BuildState_RUNNING:
		return BuildStatusRunning
	case furanrpc.BuildState_FAILURE:
		return BuildStatusFailure
	case furanrpc.BuildState_SUCCESS:
		return BuildStatusSuccess
	case furanrpc.BuildState_CANCEL_REQUESTED:
		return BuildStatusCancelRequested
	case furanrpc.BuildState_CANCELLED:
		return BuildStatusCancelled
	default:
		return BuildStatusUnknown
	}
}

// TerminalState returns whether the status is in a final (terminal) state that will not change
func (bs BuildStatus) TerminalState() bool {
	switch bs {
	case BuildStatusSuccess:
		fallthrough
	case BuildStatusFailure:
		fallthrough
	case BuildStatusSkipped:
		fallthrough
	case BuildStatusCancelled:
		return true
	default:
		return false
	}
}

type Build struct {
	ID                          uuid.UUID
	Created, Updated, Completed time.Time
	GitHubRepo, GitHubRef       string
	EncryptedGitHubCredential   []byte
	ImageRepos                  []string
	Tags                        []string
	CommitSHATag                bool
	BuildOptions                BuildOpts
	Request                     furanrpc.BuildRequest
	Status                      BuildStatus
	Events                      []string
}

// CanAddEvent indicates whether b is in a state where events can be added
func (b Build) CanAddEvent() bool {
	return b.EventListenable()
}

// EventListenable indicates where b is in a state where events can be listened for
func (b Build) EventListenable() bool {
	return !b.Status.TerminalState()
}

func (b Build) Running() bool {
	return b.Status == BuildStatusRunning
}

// TimeFromRPCTimestamp returns a UTC time.Time for an RPC timestamp
func TimeFromRPCTimestamp(ts furanrpc.Timestamp) time.Time {
	return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
}

// RPCTimestampFromTime takes a time.Time and returns an RPC timestamp
func RPCTimestampFromTime(t time.Time) furanrpc.Timestamp {
	return furanrpc.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// EncryptAndSetGitHubCredential takes a GitHub credential, encrypts it and sets EncryptedGitHubCredential accordingly
func (b *Build) EncryptAndSetGitHubCredential(cred []byte, key [32]byte) error {
	var nonce [24]byte
	if n, err := rand.Read(nonce[:]); err != nil || n != len(nonce) {
		return errors.Wrapf(err, "error reading random bytes for nonce (read: %v)", n)
	}
	b.EncryptedGitHubCredential = secretbox.Seal(nonce[:], cred, &nonce, &key)
	return nil
}

// GetGitHubCredential returns the decrypted user token using key or error
func (b Build) GetGitHubCredential(key [32]byte) (string, error) {
	var nonce [24]byte
	copy(nonce[:], b.EncryptedGitHubCredential[:24])
	tkn, ok := secretbox.Open(nil, b.EncryptedGitHubCredential[24:], &nonce, &key)
	if !ok {
		return "", errors.New("decryption error (incorrect key?)")
	}
	if tkn == nil {
		return "", errors.New("decrypted token was nil")
	}
	return string(tkn), nil
}

// BuildOpts models all options required to perform a build
type BuildOpts struct {
	BuildID                uuid.UUID               `json:"-"`
	ContextPath, CommitSHA string                  `json:"-"` // set by Builder
	RelativeDockerfilePath string                  `json:"relative_dockerfile_path"`
	BuildArgs              map[string]string       `json:"build_args"`
	Cache                  furanrpc.BuildCacheOpts `json:"cache_opts"`
	Resources              furanrpc.BuildResources `json:"resources"`
}

// Job describes methods on a single abstract build job
type Job interface {
	// Error returns a channel that will contain any errors associated with this Job
	Error() chan error
	// Running returns a channel that signals that the build the Job is executing has been updated to status Running
	// This indicates that the Furan sidecar has started and is executing successfully and will take responsibility for
	// tracking the build status from this point forward
	Running() chan struct{}
	// Logs returns all pod logs associated with the Job
	Logs() (map[string]map[string][]byte, error)
}

type JobRunner interface {
	Run(build Build) (Job, error)
}

// CacheFetcher describes an object that fetches and saves build cache
type CacheFetcher interface {
	// Fetch fetches the build cache for a build and returns a local filesystem
	// path where it was written. Caller is responsible for cleaning up the path when finished.
	Fetch(ctx context.Context, b Build) (string, error)
	// Save persists the build cache for a build located at path.
	// Caller is responsible for cleaning up the path afterward.
	Save(ctx context.Context, b Build, path string) error
}

// CodeFetcher represents an object capable of fetching code
type CodeFetcher interface {
	GetCommitSHA(ctx context.Context, repo, ref string) (string, error)
	Fetch(ctx context.Context, repo, ref, destinationPath string) error
}

// Builder describes an image build backend
type Builder interface {
	Build(ctx context.Context, opts BuildOpts) error
}

// BuilderManager describes an object that manages builds
type BuildManager interface {
	Start(ctx context.Context, opts BuildOpts) error
	Run(ctx context.Context, id uuid.UUID) error
}

type TagChecker interface {
	AllTagsExist(tags []string, repo string) (bool, []string, error)
}

// APIKey models a user-created API key
type APIKey struct {
	ID                            uuid.UUID
	Created                       time.Time
	GitHubUser, Name, Description string
	ReadOnly                      bool
}
