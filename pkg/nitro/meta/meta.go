// Package meta reads and interprets repo metadata (acyl.yml)
package meta

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dollarshaveclub/acyl/pkg/eventlogger"
	"github.com/dollarshaveclub/acyl/pkg/ghclient"
	"github.com/dollarshaveclub/acyl/pkg/match"
	"github.com/dollarshaveclub/acyl/pkg/models"
	nitroerrors "github.com/dollarshaveclub/acyl/pkg/nitro/errors"
	"github.com/pkg/errors"
	billy "gopkg.in/src-d/go-billy.v4"
	yaml "gopkg.in/yaml.v2"
)

// Getter describes an object that fetches and parses metadata (acyl.yml) from a set of repositories
type Getter interface {
	Get(ctx context.Context, rd models.RepoRevisionData) (*models.RepoConfig, error)
	FetchCharts(ctx context.Context, rc *models.RepoConfig, basePath string) (ChartLocations, error)
	GetAcylYAML(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error)
}

var _ Getter = &DataGetter{}

// DataGetter is an object that fetches and parses metadata (acyl.yml) from a set of repositories
type DataGetter struct {
	// RepoRefOverrides is a map for reponame to ref override (primarily for local development)
	RepoRefOverrides map[string]string
	RC               ghclient.RepoClient
	FS               billy.Filesystem
}

func log(ctx context.Context, msg string, args ...interface{}) {
	eventlogger.GetLogger(ctx).Printf(msg, args...)
}

// GetAcylYAML fetches acyl.yaml from repo at ref and deserializes it into rc, returning an error if the version is < 2.
func (g DataGetter) GetAcylYAML(ctx context.Context, rc *models.RepoConfig, repo, ref string) (err error) {
	defer func() {
		if err != nil {
			log(ctx, "error in acyl.yml for %v@%v: %v", repo, ref, err)
		}
	}()
	log(ctx, "fetching acyl.yml for %v@%v", repo, ref)
	b, err := g.RC.GetFileContents(ctx, repo, "acyl.yml", ref)
	if err != nil {
		return errors.Wrap(err, "error getting acyl.yml")
	}
	if err := yaml.Unmarshal(b, &rc); err != nil {
		return errors.Wrap(err, "error unmarshaling acyl.yml")
	}
	if rc.Version < 2 {
		return ErrUnsupportedVersion
	}
	if rc.Monorepo.Enabled {
		rc.Monorepo.SetValueDefaults()
		rc.Monorepo.SetRepo(repo)
		rc.Monorepo.SetRef(ref)
	} else {
		rc.Application.SetValueDefaults()
		rc.Application.Repo = repo
		rc.Application.Ref = ref
	}
	return nil
}

func (g DataGetter) getChartName(ctx context.Context, repo, ref, path string) (_ string, err error) {
	defer func() {
		if err != nil {
			log(ctx, "error getting chart name for %v@%v:%v: %v", repo, ref, path, err)
		}
	}()
	if repo == "" || ref == "" || path == "" {
		return "", fmt.Errorf("one of repo (%v), ref (%v) or path (%v) is empty", repo, ref, path)
	}
	log(ctx, "getting file contents: %v@%v: %v", repo, ref, path+"/Chart.yaml")
	b, err := g.RC.GetFileContents(ctx, repo, path+"/Chart.yaml", ref)
	if err != nil {
		return "", errors.Wrap(err, "error getting Chart.yaml")
	}
	cd := struct {
		Name string `yaml:"name"`
	}{}
	if err := yaml.Unmarshal(b, &cd); err != nil {
		return "", errors.Wrap(err, "error unmarshaling Chart.yaml")
	}
	if cd.Name == "" {
		return "", errors.New("chart name field is empty")
	}
	return cd.Name, nil
}

func (g DataGetter) getDependencyChartName(ctx context.Context, d *models.RepoConfigDependency) (cname string, err error) {
	defer func() {
		if err != nil {
			log(ctx, "error getting dependency chart name: %v", err)
			return
		}
		log(ctx, "chart name for dependency: %v", cname)
	}()
	if d == nil {
		return "", errors.New("dependency is nil")
	}
	var crepo, cref, cpath string
	switch {
	case d.AppMetadata.ChartPath != "":
		crepo, cref, cpath = d.AppMetadata.Repo, d.AppMetadata.Ref, d.AppMetadata.ChartPath
	case d.AppMetadata.ChartRepoPath != "":
		rp := &repoPath{}
		if err := rp.parseFromString(d.AppMetadata.ChartRepoPath); err != nil {
			return "", errors.Wrapf(err, "error parsing ChartRepoPath for repo dependency: %v", d.Repo)
		}
		crepo, cref, cpath = rp.repo, rp.ref, rp.path
	default:
		return "", fmt.Errorf("repo dependency lacks ChartPath/ChartRepoPath: %v", d.Repo)
	}
	return g.getChartName(ctx, crepo, cref, cpath)
}

func (g DataGetter) getRefForRepoDependency(ctx context.Context, d *models.RepoConfigDependency, rd models.RepoRevisionData, defaultBranch string, branchMatch bool) (sha, branch string, err error) {
	defer func() {
		if err != nil {
			log(ctx, "error getting ref for repo dependency: %v", err)
			return
		}
		log(ctx, "calculated ref for repo dependency: %v: %v (%v)", d.Repo, branch, sha)
	}()
	if d == nil || d.Repo == "" {
		return "", "", errors.New("empty Repo field")
	}
	log(ctx, "fetching branches for %v", d.Repo)
	branches, err := g.RC.GetBranches(ctx, d.Repo)
	if err != nil {
		return "", "", errors.Wrap(err, "error getting repo branches")
	}
	bi := make([]match.BranchInfo, len(branches))
	for i := range branches {
		bi[i] = match.BranchInfo{Name: branches[i].Name, SHA: branches[i].SHA}
	}
	defb := rd.BaseBranch
	if d.DefaultBranch != "" {
		defb = d.DefaultBranch
	}
	ri := match.RepoInfo{
		SourceBranch:  rd.SourceBranch,
		BaseBranch:    rd.BaseBranch,
		BranchMatch:   !d.DisableBranchMatch,
		DefaultBranch: defb,
	}
	sha, branch, err = match.GetRefForRepo(ri, bi)
	if err != nil {
		return "", "", errors.Wrap(err, "error getting ref for repo")
	}
	// get override if present, but only override the SHA
	if override, ok := g.RepoRefOverrides[d.Repo]; ok {
		sha = override
	}
	return sha, branch, nil
}

var ErrUnsupportedVersion = errors.New("acyl.yml is previous or unsupported version")

const DefaultFallbackBranch = "master"

// Get fetches and parses acyl.yml from owner/repo and any dependent repositories, calculates refs and returns the parsed data.
func (g DataGetter) Get(ctx context.Context, rd models.RepoRevisionData) (*models.RepoConfig, error) {
	var renameRequires = func(ds []models.RepoConfigDependency, old, new string) {
		for i, d := range ds {
			for j, r := range d.Requires {
				if r == old {
					ds[i].Requires[j] = new
				}
			}
		}
	}
	repomap := map[string]struct{}{}
	transitiveDeps := []models.RepoConfigDependency{}
	// d is current dependency
	// parent is the parent dependency (if applicable)
	// ancestor is the "root" dependency (for recursive transitive dependency trees)
	/*
		triggering repo
			 |
			 |---> repo dependency  (ancestor)
			             |
						 |----> transitive dependency (parent)
						                |
										|----> child transitive dependency (d)
	*/
	var processDep func(d, parent, ancestor *models.RepoConfigDependency) (err error)
	processDep = func(d, parent, ancestor *models.RepoConfigDependency) (err error) {
		switch {
		case d.MonorepoAppMetadata.Enabled:
			return errors.New("processDep: cannot yet use monorepo as a dependency")
		case d == nil:
			return errors.New("processDep: d is nil")
		case parent == nil:
			return errors.New("processDep: parent is nil")
		case ancestor == nil:
			return errors.New("processDep: ancestor is nil")
		}
		defer func() {
			if err != nil {
				log(ctx, "error processing dependency: %v: %v", d.Name, err)
				return
			}
			log(ctx, "processing completed for dependency: %v", d.Name)
		}()
		log(ctx, "processing dependency: %v (parent: %v, ancestor: %v)", d.Name, parent.Name, ancestor.Name)
		select {
		case <-ctx.Done():
			return errors.New("context was cancelled")
		default:
			break
		}
		switch {
		case d.Repo != "" && (d.ChartPath != "" || d.ChartRepoPath != ""):
			return fmt.Errorf("dependency error: %v: only one of Repo, ChartPath, or ChartRepoPath may be used", d.Name)
		case d.ChartPath != "" && d.ChartRepoPath != "":
			return fmt.Errorf("dependency error: %v: either ChartPath or ChartRepoPath may be used, not both", d.Name)
		case d.Repo != "":
			if _, ok := repomap[d.Repo]; ok {
				return fmt.Errorf("duplicate repository dependency: %v (check for circular dependency declarations)", d.Repo)
			}
			repomap[d.Repo] = struct{}{}
			bm := !ancestor.DisableBranchMatch
			defb := DefaultFallbackBranch
			if ancestor.DefaultBranch != "" {
				defb = ancestor.DefaultBranch
			}
			dref, dbranch, err := g.getRefForRepoDependency(ctx, d, rd, defb, bm)
			if err != nil {
				return errors.Wrapf(err, "error getting ref for repo dependency: %v", d.Repo)
			}
			drc := models.RepoConfig{}
			if err := g.GetAcylYAML(ctx, &drc, d.Repo, dref); err != nil {
				return errors.Wrapf(err, "error processing %v acyl.yml", d.Repo)
			}
			drc.Application.Branch = dbranch
			d.AppMetadata = drc.Application
			if d.Name == "" {
				name, err := g.getDependencyChartName(ctx, d)
				if err != nil {
					return errors.Wrapf(err, "error getting chart name for repo dependency: %v", d.Repo)
				}
				d.Name = name
			}

			renames := map[string]string{}
			deps := []models.RepoConfigDependency{}
			for i := range drc.Dependencies.Direct {
				dd := &drc.Dependencies.Direct[i]
				if err := processDep(dd, d, ancestor); err != nil {
					return errors.Wrapf(err, "error processing direct dependency of %v: %v", d.Name, dd.Name)
				}
				old, new := dd.Name, models.GetName(d.Repo)+"-"+dd.Name
				dd.Name = new
				renames[old] = new
				d.Requires = append(d.Requires, dd.Name)
				dd.Parent = d.Name
				deps = append(deps, *dd)
			}
			for old, new := range renames {
				renameRequires(deps, old, new)
			}
			transitiveDeps = append(transitiveDeps, deps...)

		case d.ChartPath != "" || d.ChartRepoPath != "":
			if d.DisableBranchMatch || d.DefaultBranch != "" {
				return fmt.Errorf("branch matching and default branch not available if ChartPath or ChartRepoPath is used: %v", d.Name)
			}
			var drepo, dref, dbranch string
			if d.ChartPath == "" {
				rp := &repoPath{}
				if err := rp.parseFromString(d.ChartRepoPath); err != nil {
					return errors.Wrapf(err, "dependency error: %v: malformed ChartRepoPath", d.Name)
				}
				drepo = rp.repo
				dref = rp.ref
			} else {
				drepo = parent.AppMetadata.Repo
				dref = parent.AppMetadata.Ref
				dbranch = parent.AppMetadata.Branch
			}
			d.AppMetadata = models.RepoConfigAppMetadata{
				Repo:              drepo,
				Ref:               dref,
				Branch:            dbranch,
				ChartPath:         d.ChartPath,
				ChartRepoPath:     d.ChartRepoPath,
				ChartVarsPath:     d.ChartVarsPath,
				ChartVarsRepoPath: d.ChartVarsRepoPath,
			}
			if d.Name == "" {
				name, err := g.getDependencyChartName(ctx, d)
				if err != nil {
					return errors.Wrapf(err, "error getting chart name for dependency: %v", d.AppMetadata.Repo)
				}
				d.Name = name
			}
		default:
			return fmt.Errorf("dependency error: %v: exactly one of Repo, ChartPath, or ChartRepoPath must be used", d.Name)
		}
		d.AppMetadata.SetValueDefaults()
		return nil
	}
	rc := models.RepoConfig{}
	if err := g.GetAcylYAML(ctx, &rc, rd.Repo, rd.SourceSHA); err != nil {
		return nil, errors.Wrap(err, "error processing target repo acyl.yml")
	}
	repomap[rd.Repo] = struct{}{}
	parent := &models.RepoConfigDependency{}
	if rc.Monorepo.Enabled {
		rc.Monorepo.SetBranch(rd.SourceBranch)
		parent.MonorepoAppMetadata = rc.Monorepo
	} else {
		rc.Application.Branch = rd.SourceBranch
		parent.AppMetadata = rc.Application
	}
	for i := range rc.Dependencies.Direct {
		d := &rc.Dependencies.Direct[i]
		if err := processDep(d, parent, d); err != nil {
			return nil, errors.Wrap(err, "error processing direct dependencies")
		}
	}
	for _, td := range transitiveDeps {
		rc.Dependencies.Direct = append(rc.Dependencies.Direct, td)
	}

	transitiveDeps = []models.RepoConfigDependency{}
	for i := range rc.Dependencies.Environment {
		d := &rc.Dependencies.Environment[i]
		if err := processDep(d, parent, d); err != nil {
			return nil, errors.Wrap(err, "error processing environment dependencies")
		}
	}
	for _, td := range transitiveDeps {
		rc.Dependencies.Environment = append(rc.Dependencies.Environment, td)
	}

	if ok, err := rc.Dependencies.ValidateNames(); !ok {
		return nil, errors.Wrap(err, "error validating dependency names")
	}
	return &rc, nil
}

// repoPath models a path within a repo at a ref
type repoPath struct {
	repo, path, ref string
}

func (rp *repoPath) parseFromString(crp string) error {
	// chart_repo_path: dollarshaveclub/helm-charts@master:path/to/chart
	psl := strings.Split(crp, ":")
	if len(psl) != 2 {
		return fmt.Errorf("malformed repo path: exactly one ':'' required: %v", crp)
	}
	rp.path = psl[1]
	if strings.Contains(psl[0], "@") {
		rsl := strings.Split(psl[0], "@")
		if len(rsl) != 2 {
			return fmt.Errorf("malformed repo path: no more than one '@' may be present: %v", psl[0])
		}
		rp.ref = rsl[1]
		psl[0] = rsl[0]
	}
	if len(strings.Split(psl[0], "/")) != 2 {
		return fmt.Errorf("malformed repo: exactly one '/' is required: %v", psl[0])
	}
	rp.repo = psl[0]
	return nil
}

// chartLocation models the location for a chart and the associated vars file
type chartLocation struct {
	chart repoPath
	vars  repoPath
}

func getChartLocation(d models.RepoConfigDependency) (chartLocation, error) {
	loc := chartLocation{}
	if d.AppMetadata.ChartPath == "" {
		if d.AppMetadata.ChartRepoPath == "" {
			return loc, errors.New("one of ChartPath or ChartRepoPath must be defined")
		}
		rp := &repoPath{}
		err := rp.parseFromString(d.AppMetadata.ChartRepoPath)
		if err != nil {
			return loc, errors.Wrap(err, "error validating ChartRepoPath")
		}
		loc.chart = *rp
	} else {
		loc.chart.repo = d.AppMetadata.Repo
		loc.chart.path = d.AppMetadata.ChartPath
		loc.chart.ref = d.AppMetadata.Ref
	}
	switch {
	case d.AppMetadata.ChartVarsPath != "":
		loc.vars.repo = d.AppMetadata.Repo
		loc.vars.path = d.AppMetadata.ChartVarsPath
		loc.vars.ref = d.AppMetadata.Ref
	case d.AppMetadata.ChartVarsRepoPath != "":
		rp := &repoPath{}
		err := rp.parseFromString(d.AppMetadata.ChartVarsRepoPath)
		if err != nil {
			return loc, errors.Wrap(err, "error validating ChartVarsRepoPath")
		}
		loc.vars = *rp
	}
	return loc, nil
}

// ChartLocation models the local filesystem path for the chart and the associated vars file
type ChartLocation struct {
	ChartPath, VarFilePath string
	ValueOverrides         map[string]string
}

// ChartLocations is a map of dependency name to ChartLocation
type ChartLocations map[string]ChartLocation

// FetchCharts fetches the charts for the repo and all dependencies, writing them to the filesystem g.FS at basePath/[offset]/[name] and returns a map of dependency name to filesystem path
func (g DataGetter) FetchCharts(ctx context.Context, rc *models.RepoConfig, basePath string) (ChartLocations, error) {
	// returns the local filesystem path of the chart and vars file (in order)
	fetchChartAndVars := func(i int, d models.RepoConfigDependency) (_ *ChartLocation, err error) {
		defer func() {
			if err != nil {
				log(ctx, "error fetching chart and vars for dependency: %v: %v", d.Name, err)
				return
			}
			log(ctx, "success fetching chart and vars for dependency: %v", d.Name)
		}()
		log(ctx, "fetching chart and vars for dependency: %v", d.Name)
		out := &ChartLocation{ValueOverrides: make(map[string]string)}
		cloc, err := getChartLocation(d)
		if err != nil {
			return nil, errors.Wrap(err, "error getting chart location")
		}
		cd := path.Join(basePath, strconv.Itoa(i), d.Name)
		log(ctx, "getting directory contents: %v@%v: %v", cloc.chart.repo, cloc.chart.ref, cloc.chart.path)
		dc, err := g.RC.GetDirectoryContents(ctx, cloc.chart.repo, cloc.chart.path, cloc.chart.ref)
		if err != nil {
			return nil, errors.Wrap(err, "error fetching chart contents")
		}
		for n, c := range dc {
			n = strings.Replace(n, filepath.Clean(cloc.chart.path), "", -1) // remove chart path
			fp := path.Join(cd, n)
			if err = g.FS.MkdirAll(path.Dir(fp), os.ModePerm); err != nil {
				return nil, errors.Wrap(nitroerrors.SystemError(err), "error creating directory")
			}
			if c.Symlink {
				if err := g.FS.Symlink(c.SymlinkTarget, fp); err != nil {
					return nil, errors.Wrap(nitroerrors.SystemError(err), "error creating symlink")
				}
				continue
			}
			f, err := g.FS.Create(fp)
			if err != nil {
				return nil, errors.Wrap(nitroerrors.SystemError(err), "error creating file")
			}
			defer f.Close()
			var n int
			for {
				i, err := f.Write(c.Contents[n:len(c.Contents)])
				if err != nil {
					return nil, errors.Wrap(nitroerrors.SystemError(err), "error writing to file")
				}
				n += i
				if n == len(c.Contents) {
					break
				}
			}
		}
		out.ChartPath = cd
		if cloc.vars.repo != "" {
			log(ctx, "getting file contents: %v@%v: %v", cloc.vars.repo, cloc.vars.ref, cloc.vars.path)
			fc, err := g.RC.GetFileContents(ctx, cloc.vars.repo, cloc.vars.path, cloc.vars.ref)
			if err != nil {
				return nil, errors.Wrap(err, "error getting vars file")
			}
			vcd := path.Join(cd, "vars.yml")
			vf, err := g.FS.Create(vcd)
			if err != nil {
				return nil, errors.Wrap(nitroerrors.SystemError(err), "error creating vars file")
			}
			defer vf.Close()
			n, err := vf.Write(fc)
			if err != nil {
				return nil, errors.Wrap(nitroerrors.SystemError(err), "error writing to vars file")
			}
			if n < len(fc) {
				return nil, nitroerrors.SystemError(io.ErrShortWrite)
			}
			for _, v := range d.AppMetadata.ValueOverrides {
				vsl := strings.Split(v, "=")
				if len(vsl) != 2 {
					return nil, fmt.Errorf("malformed value override: %v", v)
				}
				out.ValueOverrides[vsl[0]] = vsl[1]
			}
			out.VarFilePath = vcd
		}
		return out, nil
	}
	var fetcher QAEnvironmentChartsFetcher
	if rc.Monorepo.Enabled {
		fetcher = MonorepoChartsFetcher{}
	} else {
		fetcher = SingleAppRepoChartsFetcher{}
	}
	return fetcher.Fetch(ctx, rc, basePath, fetchChartAndVars)
}
