package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/lib/pq"
	"github.com/lib/pq/hstore"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

//go:generate stringer -type=EnvironmentStatus
//go:generate stringer -type=QADestroyReason

// EnvironmentStatus enumerates the states an environment can be in
type EnvironmentStatus int

const (
	UnknownStatus EnvironmentStatus = iota // UnknownStatus means unknown
	Spawned                                // Spawned means created but not yet registered (we don't know InstanceID or IPs yet)
	Success                                // Success means provisioning was successful and applications are working
	Failure                                // Failure means there was an error during provisioning or application startup
	Destroyed                              // Destroyed means the environment was destroyed explicitly or as part of a PR synchronize or close
	Updating                               // Updating means an existing env is being updated (replaced behind the scenes)
)

// EnvironmentStatusFromString returns the EnvironmentStatus constant for a string or error if unknown
func EnvironmentStatusFromString(es string) (EnvironmentStatus, error) {
	es = strings.ToLower(es)
	switch es {
	case "spawned":
		return Spawned, nil
	case "success":
		return Success, nil
	case "failure":
		return Failure, nil
	case "destroyed":
		return Destroyed, nil
	case "updating":
		return Updating, nil
	default:
		return UnknownStatus, fmt.Errorf("unknown status")
	}
}

// QADestroyReason enumerates the reasons for destroying an environment
type QADestroyReason int

const (
	ReapAgeSpawned               QADestroyReason = iota // Environment exceeds max age
	ReapAgeFailure                                      // Environment has been in state Failure for too long
	ReapPrClosed                                        // PR associated with the environment is now closed
	ReapEnvironmentLimitExceeded                        // Environment destroyed by Reaper to bring environment count into compliance with the global limit
	CreateFoundStale                                    // The environment is a stale environment associated with a PR that we are executing a create for
	DestroyApiRequest                                   // Explicit API destroy request
	EnvironmentLimitExceeded                            // Environment destroyed by a new environment create request to bring environment count into compliance with the global limit
)

// RefMap is a mapping of Github repository to a ref.
type RefMap map[string]string

// RepoRevisionData models the GitHub repo data
type RepoRevisionData struct {
	User         string `json:"user"`
	Repo         string `json:"repo"`
	PullRequest  uint   `json:"pull_request"`
	SourceSHA    string `json:"source_sha"`
	BaseSHA      string `json:"base_sha"`
	SourceBranch string `json:"source_branch"`
	BaseBranch   string `json:"base_branch"`
	SourceRef    string `json:"source_ref"` // if environment is not based on a PR
	IsFork       bool   `json:"is_fork"`    // set if PR head is from a different repo (fork) from base
}

// QAEnvironment describes an individual QA environment
// Fields prefixed with "Raw" are directly out of the database without processing
type QAEnvironment struct {
	Name                     string               `json:"name"`
	Created                  time.Time            `json:"created"`
	CreatedDate              string               `json:"created_date"`
	RawEvents                []string             `json:"-"`
	Events                   []QAEnvironmentEvent `json:"events"`
	Hostname                 string               `json:"hostname"`
	QAType                   string               `json:"qa_type"`
	User                     string               `json:"user"`
	Repo                     string               `json:"repo"`
	PullRequest              uint                 `json:"pull_request"`
	SourceSHA                string               `json:"source_sha"`
	BaseSHA                  string               `json:"base_sha"`
	SourceBranch             string               `json:"source_branch"`
	BaseBranch               string               `json:"base_branch"`
	SourceRef                string               `json:"source_ref"`
	RawStatus                string               `json:"status"`
	Status                   EnvironmentStatus    `json:"status_int"`
	RefMap                   RefMap               `json:"ref_map"`
	CommitSHAMap             RefMap               `json:"commit_sha_map"`
	AminoServiceToPort       map[string]int64     `json:"amino_service_to_port"`
	AminoServiceToPortRaw    map[string]string    `json:"-"`
	AminoKubernetesNamespace string               `json:"amino_kubernetes_namespace"`
	AminoEnvironmentID       int                  `json:"amino_environment_id"`

	rmapHS  hstore.Hstore
	csmapHS hstore.Hstore
	as2pHS  hstore.Hstore
}

// Columns returns a comma-separated string of column names suitable for a SELECT
func (qae QAEnvironment) Columns() string {
	return "name, created, raw_events, hostname, qa_type, username, repo, pull_request, source_sha, base_sha, source_branch, base_branch, source_ref, status, ref_map, commit_sha_map, amino_service_to_port, amino_kubernetes_namespace, amino_environment_id"
}

// InsertParams returns the query placeholder params for a full model insert
func (qae QAEnvironment) InsertParams() string {
	params := []string{}
	for i := range strings.Split(qae.Columns(), ", ") {
		params = append(params, fmt.Sprintf("$%v", i+1))
	}
	return strings.Join(params, ", ")
}

// ScanValues returns a slice of values suitable for a query Scan()
func (qae *QAEnvironment) ScanValues() []interface{} {
	return []interface{}{&qae.Name, &qae.Created, pq.Array(&qae.RawEvents), &qae.Hostname, &qae.QAType, &qae.User, &qae.Repo, &qae.PullRequest, &qae.SourceSHA, &qae.BaseSHA, &qae.SourceBranch, &qae.BaseBranch, &qae.SourceRef, &qae.Status, qae.RefMapHStore(), qae.CommitSHAMapHStore(), qae.AminoServiceToPortHStore(), &qae.AminoKubernetesNamespace, &qae.AminoEnvironmentID}
}

// RefMapHStore returns the HStore struct suitable for scanning during queries
func (qae *QAEnvironment) RefMapHStore() *hstore.Hstore {
	qae.rmapHS.Map = make(map[string]sql.NullString)
	for k, v := range qae.RefMap {
		qae.rmapHS.Map[k] = sql.NullString{
			String: v,
			Valid:  true,
		}
	}
	return &qae.rmapHS
}

// CommitSHAMapHStore returns the HStore struct suitable for scanning during queries
func (qae *QAEnvironment) CommitSHAMapHStore() *hstore.Hstore {
	qae.csmapHS.Map = make(map[string]sql.NullString)
	for k, v := range qae.CommitSHAMap {
		qae.csmapHS.Map[k] = sql.NullString{
			String: v,
			Valid:  true,
		}
	}
	return &qae.csmapHS
}

// AminoServiceToPortHStore returns the HStore struct suitable for scanning during queries
func (qae *QAEnvironment) AminoServiceToPortHStore() *hstore.Hstore {
	qae.as2pHS.Map = make(map[string]sql.NullString)
	for k, v := range qae.AminoServiceToPort {
		qae.as2pHS.Map[k] = sql.NullString{
			String: strconv.Itoa(int(v)),
			Valid:  true,
		}
	}
	return &qae.as2pHS
}

// ProcessHStores processes raw HStores into their respective struct fields
func (qae *QAEnvironment) ProcessHStores() error {
	if qae.rmapHS.Map == nil || qae.csmapHS.Map == nil || qae.as2pHS.Map == nil {
		return errors.New("one or more hstore maps are nil")
	}
	qae.RefMap = make(map[string]string)
	qae.CommitSHAMap = make(map[string]string)
	qae.AminoServiceToPort = make(map[string]int64)
	for k, v := range qae.rmapHS.Map {
		if v.Valid {
			qae.RefMap[k] = v.String
		}
	}
	for k, v := range qae.csmapHS.Map {
		if v.Valid {
			qae.CommitSHAMap[k] = v.String
		}
	}
	for k, v := range qae.as2pHS.Map {
		if v.Valid {
			i, err := strconv.Atoi(v.String)
			if err != nil {
				return errors.Wrapf(err, "error parsing amino service port: %v", v.String)
			}
			qae.AminoServiceToPort[k] = int64(i)
		}
	}
	return nil
}

func (qae *QAEnvironment) BeforeInsert(db orm.DB) error {
	if err := qae.SetRaw(); err != nil {
		return errors.Wrapf(err, "error setting raw model values")
	}
	return nil
}

func (qae *QAEnvironment) AfterInsert(db orm.DB) error {
	if err := qae.SetRaw(); err != nil {
		return errors.Wrapf(err, "error setting raw model values")
	}
	return nil
}

func (qae *QAEnvironment) AfterSelect(db orm.DB) error {
	if err := qae.ProcessRaw(); err != nil {
		return errors.Wrapf(err, "error parsing raw data")
	}
	return nil
}

// RepoRevisionDataFromQA returns a RepoRevisionData from a QAEnvironment
func (qa QAEnvironment) RepoRevisionDataFromQA() *RepoRevisionData {
	return &RepoRevisionData{
		User:         qa.User,
		Repo:         qa.Repo,
		PullRequest:  qa.PullRequest,
		SourceSHA:    qa.SourceSHA,
		SourceBranch: qa.SourceBranch,
		BaseSHA:      qa.BaseSHA,
		BaseBranch:   qa.BaseBranch,
		SourceRef:    qa.SourceRef,
	}
}

// QAEnvironments is a slice of QAEnvironment to allow sorting by Created timestamp
type QAEnvironments []QAEnvironment

func (slice QAEnvironments) Len() int {
	return len(slice)
}

func (slice QAEnvironments) Less(i, j int) bool {
	return slice[i].Created.Before(slice[j].Created)
}

func (slice QAEnvironments) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// SetCreatedDate sets the CreatedDate field from Created for the secondary table
func (qa *QAEnvironment) SetCreatedDate() {
	qa.CreatedDate = qa.Created.Format("2006-01-02") // YYYY-MM-DD
}

// ProcessRaw converts raw values from the database into associated data fields
func (qae *QAEnvironment) ProcessRaw() error {
	qae.ProcessStatus()
	err := qae.ProcessEvents()
	if err != nil {
		return err
	}
	if err := qae.ProcessAminoServiceToPort(); err != nil {
		return errors.Wrapf(err, "error parsing out AminoServiceToPort")
	}
	return nil
}

// ProcessAminoServiceToPort parses out a map[string]int64 port
// mapping from AminoServiceToPortRaw since hstore doesn't integers.
func (qae *QAEnvironment) ProcessAminoServiceToPort() error {
	if len(qae.AminoServiceToPortRaw) > 0 {
		qae.AminoServiceToPort = make(map[string]int64)
		for k, v := range qae.AminoServiceToPortRaw {
			value, err := strconv.Atoi(v)
			if err != nil {
				return errors.Wrapf(err, "error parsing port")
			}
			qae.AminoServiceToPort[k] = int64(value)
		}
	}
	return nil
}

// ProcessEvents unmarshals a list of JSON-encoded event strings into
func (qae *QAEnvironment) ProcessEvents() error {
	if len(qae.RawEvents) == 0 || len(qae.Events) > 0 {
		// if there are no raw events or Events has already been populated, abort
		return nil
	}
	events := []QAEnvironmentEvent{}
	for _, re := range qae.RawEvents {
		event := QAEnvironmentEvent{}
		err := json.Unmarshal([]byte(re), &event)
		if err != nil {
			return err
		}
		events = append(events, event)
	}
	qae.Events = events
	return nil
}

// SetRaw processes/serializes fields into their respective Raw* fields in
// preparation for insertion into the database
func (qae *QAEnvironment) SetRaw() error {
	var err error
	var rawe []byte
	qae.RawEvents = []string{}
	for _, e := range qae.Events {
		rawe, err = json.Marshal(&e)
		if err != nil {
			return fmt.Errorf("error serializing event: %v", err)
		}
		qae.RawEvents = append(qae.RawEvents, string(rawe))
	}
	qae.RawStatus = strings.ToLower(qae.Status.String())
	if len(qae.AminoServiceToPort) > 0 {
		qae.AminoServiceToPortRaw = make(map[string]string)
		for k, v := range qae.AminoServiceToPort {
			value := strconv.Itoa(int(v))
			if err != nil {
				return errors.Wrapf(err, "error parsing port")
			}
			qae.AminoServiceToPortRaw[k] = value
		}
	}
	return nil
}

// ProcessStatus sets Status from RawStatus
func (qae *QAEnvironment) ProcessStatus() {
	s, _ := EnvironmentStatusFromString(qae.RawStatus)
	qae.Status = s
}

// QAEnvironmentEvent is an event registered by the QA environment during provisioning
type QAEnvironmentEvent struct {
	Timestamp time.Time `json:"created"`
	Message   string    `json:"message"`
}

// QAType defines a QA environment type and how it needs to be triggered
// This is the type for the legacy (version < 2) acyl.yml
type QAType struct {
	ID              int               `json:"id"`
	Version         uint              `json:"version" yaml:"version"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Name            string            `json:"name" yaml:"name"`
	Template        string            `json:"template" yaml:"template"`                 // dollarshaveclub/resource-manifests
	TargetRepo      string            `json:"target_repo" yaml:"target_repo"`           // GitHub repo to watch for PR events
	OtherRepos      []string          `json:"other_repos" yaml:"other_repos"`           // Any other repos used by the template that require images built or branch coordination
	TargetBranch    string            `json:"target_branch" yaml:"target_branch"`       // Branch to watch on target repo
	TargetBranches  []string          `json:"target_branches" yaml:"target_branches"`   // Branch list to watch on target repo
	BranchOverrides map[string]string `json:"branch_overrides" yaml:"branch_overrides"` // Map of repo name to override branch (Instead of using base branch of PR, fallback to this branch)
	EnvType         string            `json:"env_type" yaml:"env_type"`
	TrackRefs       []string          `json:"track_refs" yaml:"track_refs"` // Create an environment for each of these refs (branch/tag) and keep it continuously updated as commits are pushed
}

// FromYAML unmarshals the provided YAML data into the QAType struct
func (qat *QAType) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, &qat)
}

// EnvSearchParameters models the possible parameters for an environment search
type EnvSearchParameters struct {
	Repo         string
	Repos        []string // include all these repos (mutually exclusive with Repo)
	Pr           uint
	SourceSHA    string
	SourceBranch string
	User         string
	Status       EnvironmentStatus
	Statuses     []EnvironmentStatus // include all these statuses (mutually exclusive with Status)
	CreatedSince time.Duration       // Duration prior to time.Now().UTC()
	TrackingRef  string
}
