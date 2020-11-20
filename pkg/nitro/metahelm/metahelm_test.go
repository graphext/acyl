package metahelm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/dollarshaveclub/acyl/pkg/config"
	"github.com/dollarshaveclub/acyl/pkg/eventlogger"
	"github.com/dollarshaveclub/acyl/pkg/match"
	"github.com/dollarshaveclub/acyl/pkg/models"
	"github.com/dollarshaveclub/acyl/pkg/nitro/images"
	"github.com/dollarshaveclub/acyl/pkg/nitro/metrics"
	"github.com/dollarshaveclub/acyl/pkg/persistence"
	"github.com/dollarshaveclub/metahelm/pkg/metahelm"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	mtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	rls "k8s.io/helm/pkg/proto/hapi/release"
)

func chartMap(charts []metahelm.Chart) map[string]metahelm.Chart {
	out := map[string]metahelm.Chart{}
	for _, c := range charts {
		out[c.Title] = c
	}
	return out
}

func TestMetahelmGenerateCharts(t *testing.T) {
	cases := []struct {
		name, inputNS, inputEnvName string
		inputRC                     models.RepoConfig
		inputCL                     ChartLocations
		isError, disableDefaults    bool
		errContains                 string
		verifyf                     func([]metahelm.Chart) error
	}{
		{
			name:         "Valid",
			inputNS:      "fake-name",
			inputEnvName: "fake-env-name",
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue:  "image.tag",
					Repo:           "foo/bar",
					Ref:            "aaaa",
					ValueOverrides: []string{"something=qqqq"},
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Name: "bar-baz",
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue:  "image.tag",
								Repo:           "bar/baz",
								Ref:            "bbbb",
								ValueOverrides: []string{"somethingelse=zzzz", "yetanotherthing=xxxx"},
							},
							ValueOverrides: []string{"yetanotherthing=yyyy"},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
				"bar-baz": ChartLocation{ChartPath: ".chart"},
			},
			verifyf: func(charts []metahelm.Chart) error {
				if len(charts) != 2 {
					return fmt.Errorf("bad chart length: %v", len(charts))
				}
				cm := chartMap(charts)
				if c, ok := cm["foo-bar"]; ok {
					if len(c.DependencyList) != 1 {
						return fmt.Errorf("bad dependencies for foo/bar: %v", c.DependencyList)
					}
					if c.DependencyList[0] != "bar-baz" {
						return fmt.Errorf("bad dependency for foo/bar: %v", c.DependencyList[0])
					}
				} else {
					return errors.New("foo-bar missing")
				}
				if c, ok := cm["bar-baz"]; ok {
					if len(c.DependencyList) != 0 {
						return fmt.Errorf("bad dependencies for bar-baz: %v", c.DependencyList)
					}
				} else {
					return errors.New("bar-baz missing")
				}
				checkOverrideString := func(overrides []byte, name, value string) error {
					om := map[string]interface{}{}
					if err := yaml.Unmarshal(overrides, &om); err != nil {
						return fmt.Errorf("error unmarshaling overrides: %w", err)
					}
					vi, ok := om[name]
					if !ok {
						return fmt.Errorf("override is missing: %v", name)
					}
					v, ok := vi.(string)
					if !ok {
						return fmt.Errorf("%v value is unexpected type: %T", name, vi)
					}
					if v != value {
						return fmt.Errorf("bad value for %v: %v", name, v)
					}
					return nil
				}
				for _, chart := range charts {
					if err := checkOverrideString(chart.ValueOverrides, models.DefaultNamespaceValue, "fake-name"); err != nil {
						return fmt.Errorf("error checking override: %w", err)
					}
				}
				chart := cm["foo-bar"]
				if err := checkOverrideString(chart.ValueOverrides, "something", "qqqq"); err != nil {
					return fmt.Errorf("error checking something override: %w", err)
				}
				chart = cm["bar-baz"]
				if err := checkOverrideString(chart.ValueOverrides, "somethingelse", "zzzz"); err != nil {
					return fmt.Errorf("error checking somethingelse override: %w", err)
				}
				if err := checkOverrideString(chart.ValueOverrides, "yetanotherthing", "yyyy"); err != nil {
					return fmt.Errorf("error checking yetanotherthing override: %w", err)
				}
				return nil
			},
		},
		{
			name:         "missing ref on dep",
			inputNS:      "fake-name",
			inputEnvName: "fake-env-name",
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue: "image.tag",
					Repo:          "foo/bar",
					Ref:           "aaaa",
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Name: "bar-baz",
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue: "image.tag",
								Repo:          "bar/baz",
							},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
				"bar-baz": ChartLocation{ChartPath: ".chart"},
			},
			isError:     true,
			errContains: "Ref is empty",
		},
		{
			name:         "missing name on dep",
			inputNS:      "fake-name",
			inputEnvName: "fake-env-name",
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue: "image.tag",
					Repo:          "foo/bar",
					Ref:           "aaaa",
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue: "image.tag",
								Repo:          "bar/baz",
								Ref:           "bbbb",
							},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
				"bar-baz": ChartLocation{ChartPath: ".chart"},
			},
			isError:     true,
			errContains: "Name is empty",
		},
		{
			name:         "missing dep from chart locations",
			inputNS:      "fake-name",
			inputEnvName: "fake-env-name",
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue: "image.tag",
					Repo:          "foo/bar",
					Ref:           "aaaa",
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Name: "bar-baz",
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue: "image.tag",
								Repo:          "bar/baz",
								Ref:           "bbbb",
							},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
			},
			isError:     true,
			errContains: "dependency not found in ChartLocations",
		},
		{
			name:         "unknown requires",
			inputNS:      "fake-name",
			inputEnvName: "fake-env-name",
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue: "image.tag",
					Repo:          "foo/bar",
					Ref:           "aaaa",
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Name: "bar-baz",
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue: "image.tag",
								Repo:          "bar/baz",
								Ref:           "bbbb",
							},
							Requires: []string{"doesnotexist"},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
				"bar-baz": ChartLocation{ChartPath: ".chart"},
			},
			isError:     true,
			errContains: "unknown requires on chart",
		},
		{
			name:            "empty chart tag value",
			inputNS:         "fake-name",
			inputEnvName:    "fake-env-name",
			disableDefaults: true,
			inputRC: models.RepoConfig{
				Application: models.RepoConfigAppMetadata{
					ChartTagValue: "",
					Repo:          "foo/bar",
					Ref:           "aaaa",
				},
				Dependencies: models.DependencyDeclaration{
					Direct: []models.RepoConfigDependency{
						models.RepoConfigDependency{
							Name: "bar-baz",
							Repo: "bar/baz",
							AppMetadata: models.RepoConfigAppMetadata{
								ChartTagValue: "image.tag",
								Repo:          "bar/baz",
								Ref:           "bbbb",
							},
						},
					},
				},
			},
			inputCL: ChartLocations{
				"foo-bar": ChartLocation{ChartPath: ".chart"},
				"bar-baz": ChartLocation{ChartPath: ".chart"},
			},
			isError:     true,
			errContains: "ChartTagValue is empty",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !c.disableDefaults {
				c.inputRC.Application.SetValueDefaults()
				for i := range c.inputRC.Dependencies.Direct {
					c.inputRC.Dependencies.Direct[i].AppMetadata.SetValueDefaults()
				}
				for i := range c.inputRC.Dependencies.Environment {
					c.inputRC.Dependencies.Environment[i].AppMetadata.SetValueDefaults()
				}
			}
			newenv := &EnvInfo{Env: &models.QAEnvironment{Name: c.inputEnvName}, RC: &c.inputRC}
			cl, err := ChartInstaller{mc: &metrics.FakeCollector{}}.GenerateCharts(context.Background(), c.inputNS, newenv, c.inputCL)
			if err != nil {
				if !c.isError {
					t.Fatalf("should have succeeded: %v", err)
				}
				if !strings.Contains(err.Error(), c.errContains) {
					t.Fatalf("error missing string (%v): %v", c.errContains, err)
				}
				return
			}
			if c.isError {
				t.Fatalf("should have failed")
			}
			if c.verifyf != nil {
				if err := c.verifyf(cl); err != nil {
					t.Fatalf(err.Error())
				}
			}
		})
	}
}

func TestMetahelmGetTillerPods(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "foo",
			Labels:    map[string]string{"app": "helm"},
		},
	}
	fkc := fake.NewSimpleClientset(pod)
	ci := ChartInstaller{kc: fkc}
	pl, err := ci.getTillerPods("foo")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if len(pl.Items) != 1 {
		t.Fatalf("bad pods length: %v", len(pl.Items))
	}
}

func TestMetahelmCheckTillerPods(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "foo",
			Labels:    map[string]string{"app": "helm"},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				v1.ContainerStatus{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
		},
	}
	fkc := fake.NewSimpleClientset(pod)
	ci := ChartInstaller{kc: fkc}
	ok, err := ci.checkTillerPods("foo")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if !ok {
		t.Fatalf("should have returned true")
	}
	pod.Status.ContainerStatuses[0].State = v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason: "something",
		},
	}
	fkc = fake.NewSimpleClientset(pod)
	ci = ChartInstaller{kc: fkc}
	ok, err = ci.checkTillerPods("foo")
	if err == nil {
		t.Fatalf("waiting should have returned error")
	}
	if !ok {
		t.Fatalf("waiting should have returned true")
	}
	pod.Status.ContainerStatuses[0].State = v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason: "ImagePullBackOff",
		},
	}
	fkc = fake.NewSimpleClientset(pod)
	ci = ChartInstaller{kc: fkc}
	ok, err = ci.checkTillerPods("foo")
	if err == nil {
		t.Fatalf("imagepullbackoff should have returned error")
	}
	if ok {
		t.Fatalf("imagepullbackoff should have returned false")
	}
}

func TestMetahelmCreateNamespace(t *testing.T) {
	fkc := fake.NewSimpleClientset()
	dl := persistence.NewFakeDataLayer()
	ci := ChartInstaller{kc: fkc, dl: dl}
	valid := func(ns string) error {
		if !strings.HasPrefix(ns, "nitro-") {
			return errors.New("prefix missing")
		}
		if utf8.RuneCountInString(ns) > 63 {
			return errors.New("namespace name too long")
		}
		return nil
	}
	cases := []struct {
		name, env string
	}{
		{"Valid", "foo-bar"},
		{"Name too long", "sadfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns, err := ci.createNamespace(context.Background(), c.env)
			if err != nil {
				t.Fatalf("should have succeeded: %v", err)
			}
			if err := valid(ns); err != nil {
				t.Fatalf(err.Error())
			}
		})
	}
}

func TestMetahelmInstallTiller(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	var server net.Conn
	go func() {
		defer ln.Close()
		time.Sleep(10 * time.Millisecond)
		server, err = ln.Accept()
	}()
	server = server
	asl := strings.Split(ln.Addr().String(), ":")
	ip := asl[0]
	port, _ := strconv.Atoi(asl[1])
	tcfg := TillerConfig{
		Port:                    uint(port),
		ServerConnectRetryDelay: 1 * time.Millisecond,
	}
	tcfg = tcfg.SetDefaults()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "foo",
			Labels:    map[string]string{"app": "helm"},
		},
		Status: v1.PodStatus{
			PodIP: ip,
		},
	}
	fkc := fake.NewSimpleClientset(pod)
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), &models.QAEnvironment{Name: "foo-bar"})
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{}, nil
		},
		tcfg: tcfg,
	}
	podip, err := ci.installTiller(context.Background(), "foo-bar", "foo")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if podip != ln.Addr().String() {
		t.Fatalf("bad pod ip (expected %v): %v", ip, podip)
	}
	pod.Status.ContainerStatuses = make([]v1.ContainerStatus, 1)
	pod.Status.ContainerStatuses[0].State = v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason: "ImagePullBackOff",
		},
	}
	fkc = fake.NewSimpleClientset()
	ci = ChartInstaller{kc: fkc, dl: dl}
	_, err = ci.installTiller(context.Background(), "foo-bar", "foo")
	if err == nil {
		t.Fatalf("should have failed")
	}
	if !strings.Contains(err.Error(), "timed out waiting for Tiller") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// generate mock k8s objects for the supplied charts
func gentestobjs(charts []metahelm.Chart) []runtime.Object {
	objs := []runtime.Object{}
	reps := int32(1)
	iscontroller := true
	rsl := appsv1.ReplicaSetList{Items: []appsv1.ReplicaSet{}}
	for _, c := range charts {
		r := &appsv1.ReplicaSet{}
		d := &appsv1.Deployment{}
		d.Spec.Replicas = &reps
		d.Spec.Template.Labels = map[string]string{"app": c.Name()}
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
		d.Spec.Template.Name = c.Name()
		d.Spec.Selector = &metav1.LabelSelector{}
		d.Spec.Selector.MatchLabels = map[string]string{"app": c.Name()}
		r.Spec.Selector = d.Spec.Selector
		r.Spec.Replicas = &reps
		r.Status.ReadyReplicas = 1
		r.Name = "replicaset-" + c.Name()
		r.Namespace = "foo"
		r.Labels = d.Spec.Template.Labels
		d.Labels = d.Spec.Template.Labels
		d.ObjectMeta.UID = mtypes.UID(c.Name() + "-deployment")
		r.ObjectMeta.OwnerReferences = []metav1.OwnerReference{metav1.OwnerReference{UID: d.ObjectMeta.UID, Controller: &iscontroller}}
		d.Name = c.Name()
		d.Namespace = "foo"
		r.Spec.Template = d.Spec.Template
		objs = append(objs, d)
		rsl.Items = append(rsl.Items, *r)
	}
	return append(objs, &rsl)
}

func tillerpod() *v1.Pod {
	tpod := &v1.Pod{}
	tpod.Namespace = "foo"
	tpod.Labels = map[string]string{"app": "helm"}
	tpod.Status.PodIP = "10.0.0.1"
	return tpod
}

func TestMetahelmInstallCharts(t *testing.T) {
	charts := []metahelm.Chart{
		metahelm.Chart{Title: "foo", Location: "foo/bar", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "foo", DependencyList: []string{"bar"}},
		metahelm.Chart{Title: "bar", Location: "bar/baz", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "bar"},
	}
	tobjs := gentestobjs(charts)
	tobjs = append(tobjs, tillerpod())
	fkc := fake.NewSimpleClientset(tobjs...)
	ib := &images.FakeImageBuilder{BatchCompletedFunc: func(envname, repo string) (bool, error) { return true, nil }}
	rc := &models.RepoConfig{
		Application: models.RepoConfigAppMetadata{
			Repo:          "foo",
			Ref:           "aaaa",
			Branch:        "master",
			Image:         "foo",
			ChartTagValue: "image.tag",
		},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "bar",
					Repo: "bar",
					AppMetadata: models.RepoConfigAppMetadata{
						Repo:          "bar",
						Ref:           "bbbbbb",
						Branch:        "foo",
						Image:         "bar",
						ChartTagValue: "image.tag",
					},
				},
			},
		},
	}
	b, err := ib.StartBuilds(context.Background(), "foo-bar", rc)
	if err != nil {
		t.Fatalf("StartBuilds failed: %v", err)
	}
	defer b.Stop()
	nenv := &EnvInfo{
		Env: &models.QAEnvironment{Name: "foo-bar"},
		RC:  rc,
	}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), nenv.Env)
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
		ib: ib,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{}, nil
		},
		mc: &metrics.FakeCollector{},
	}
	metahelm.ChartWaitPollInterval = 10 * time.Millisecond
	el := &eventlogger.Logger{DL: dl}
	el.Init([]byte{}, "foo/bar", 99)
	ctx := eventlogger.NewEventLoggerContext(context.Background(), el)
	if err := ci.installOrUpgradeCharts(ctx, "127.0.0.1:4404", "foo", charts, nenv, b, false); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestMetahelmInstallAndUpgradeChartsBuildError(t *testing.T) {
	charts := []metahelm.Chart{
		metahelm.Chart{Title: "foo", Location: "foo/bar", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "foo", DependencyList: []string{"bar"}},
		metahelm.Chart{Title: "bar", Location: "bar/baz", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "bar"},
	}
	tobjs := gentestobjs(charts)
	tobjs = append(tobjs, tillerpod())
	fkc := fake.NewSimpleClientset(tobjs...)
	berr := errors.New("build error")
	ib := &images.FakeImageBuilder{
		BatchCompletedFunc: func(envname, repo string) (bool, error) {
			return true, berr
		},
	}
	rc := &models.RepoConfig{
		Application: models.RepoConfigAppMetadata{
			Repo:          "foo",
			Ref:           "aaaa",
			Branch:        "master",
			Image:         "foo",
			ChartTagValue: "image.tag",
		},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "bar",
					Repo: "bar",
					AppMetadata: models.RepoConfigAppMetadata{
						Repo:          "bar",
						Ref:           "bbbbbb",
						Branch:        "foo",
						Image:         "bar",
						ChartTagValue: "image.tag",
					},
				},
			},
		},
	}
	b, err := ib.StartBuilds(context.Background(), "foo-bar", rc)
	if err != nil {
		t.Fatalf("StartBuilds failed: %v", err)
	}
	defer b.Stop()
	nenv := &EnvInfo{
		Env: &models.QAEnvironment{Name: "foo-bar"},
		RC:  rc,
	}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), nenv.Env)
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
		ib: ib,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{}, nil
		},
		mc: &metrics.FakeCollector{},
	}
	metahelm.ChartWaitPollInterval = 10 * time.Millisecond
	err = ci.installOrUpgradeCharts(context.Background(), "127.0.0.1:4404", "foo", charts, nenv, b, false)
	if err == nil {
		t.Fatalf("install should have failed")
	}
	if err != berr {
		t.Fatalf("install did not return build error: %v", err)
	}
	b2, err := ib.StartBuilds(context.Background(), "foo-bar", rc)
	if err != nil {
		t.Fatalf("StartBuilds failed: %v", err)
	}
	defer b2.Stop()
	nenv.Releases = map[string]string{"foo": "foo", "bar": "bar"}
	ci = ChartInstaller{
		kc: fkc,
		dl: dl,
		ib: ib,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{}, nil
		},
		mc: &metrics.FakeCollector{},
	}
	metahelm.ChartWaitPollInterval = 10 * time.Millisecond
	err = ci.installOrUpgradeCharts(context.Background(), "127.0.0.1:4404", "foo", charts, nenv, b2, true)
	if err == nil {
		t.Fatalf("upgrade should have failed")
	}
	if err != berr {
		t.Fatalf("upgrade did not return build error: %v", err)
	}
}

func TestMetahelmWriteReleaseNames(t *testing.T) {
	rmap := map[string]string{
		"foo-bar":  "random",
		"foo-bar2": "random2",
	}
	rc := models.RepoConfig{
		Application: models.RepoConfigAppMetadata{Repo: "foo/bar", Ref: "asdf", Branch: "random"},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "foo-bar2",
					AppMetadata: models.RepoConfigAppMetadata{
						Ref:    "1234",
						Branch: "random2",
					},
				},
			},
		},
	}
	name := "foo-bar"
	newenv := &EnvInfo{Env: &models.QAEnvironment{Name: name}, RC: &rc}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), newenv.Env)
	ci := ChartInstaller{dl: dl}
	if err := ci.writeReleaseNames(context.Background(), rmap, "fake-namespace", newenv); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	releases, err := dl.GetHelmReleasesForEnv(context.Background(), name)
	if err != nil {
		t.Fatalf("get should have succeeded: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("bad length: %v", len(releases))
	}
	for i, r := range releases {
		if r.K8sNamespace != "fake-namespace" {
			t.Fatalf("bad namespace at offset %v: %v", i, r.K8sNamespace)
		}
	}
	// test writing with existing releases
	if err := ci.writeReleaseNames(context.Background(), rmap, "fake-namespace", newenv); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestMetahelmUpdateReleaseRevisions(t *testing.T) {
	rmap := map[string]string{
		"foo-bar":  "random",
		"foo-bar2": "random2",
	}
	rc := models.RepoConfig{
		Application: models.RepoConfigAppMetadata{Repo: "foo/bar", Ref: "1234", Branch: "random"},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "foo-bar2",
					AppMetadata: models.RepoConfigAppMetadata{
						Ref:    "1234",
						Branch: "random2",
					},
				},
			},
		},
	}
	name := "foo-bar"
	env := &EnvInfo{Env: &models.QAEnvironment{Name: name}, Releases: rmap, RC: &rc}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), env.Env)
	releases := []models.HelmRelease{
		models.HelmRelease{EnvName: name, Release: "random", RevisionSHA: "9999"},
		models.HelmRelease{EnvName: name, Release: "random2", RevisionSHA: "9999"},
	}
	dl.CreateHelmReleasesForEnv(context.Background(), releases)
	ci := ChartInstaller{dl: dl}
	if err := ci.updateReleaseRevisions(context.Background(), env); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	releases, err := dl.GetHelmReleasesForEnv(context.Background(), name)
	if err != nil {
		t.Fatalf("get should have succeeded: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("bad length: %v", len(releases))
	}
	for _, r := range releases {
		if r.RevisionSHA != "1234" {
			t.Fatalf("bad SHA: %v", r.RevisionSHA)
		}
	}
}

func TestMetahelmWriteK8sEnvironment(t *testing.T) {
	name := "foo-bar"
	rc := &models.RepoConfig{
		Application: models.RepoConfigAppMetadata{
			Repo:   "foo/bar",
			Ref:    "aaaa",
			Branch: "bar",
		},
	}
	newenv := &EnvInfo{Env: &models.QAEnvironment{Name: name}, RC: rc}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), newenv.Env)
	ci := ChartInstaller{dl: dl}
	if err := ci.writeK8sEnvironment(context.Background(), newenv, "foo"); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	k8s, err := dl.GetK8sEnv(context.Background(), name)
	if err != nil {
		t.Fatalf("get should have succeeded: %v", err)
	}
	refmap2 := map[string]string{}
	if err := json.Unmarshal([]byte(k8s.RefMapJSON), &refmap2); err != nil {
		t.Fatalf("json umarshal failed: %v", err)
	}
	rm, _ := newenv.RC.RefMap()
	if refmap2["foo/bar"] != rm["foo/bar"] {
		t.Fatalf("bad refmap value: %v", refmap2["foo/bar"])
	}
	if k8s.Namespace != "foo" {
		t.Fatalf("bad namespace: %v", k8s.Namespace)
	}
	// test writing an existing record
	fkc := fake.NewSimpleClientset(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	ci.kc = fkc
	if err := ci.writeK8sEnvironment(context.Background(), newenv, "foo"); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestMetahelmMergeVars(t *testing.T) {
	cases := []struct {
		name, inputYAML string
		inputOverrides  map[string]string
		output          []byte
		isError         bool
		errContains     string
	}{
		{
			"empty", "", map[string]string{"image": "foo"}, []byte("image: foo\n"), false, "",
		},
		{
			"simple", "image: foo\n", map[string]string{"image": "bar"}, []byte("image: bar\n"), false, "",
		},
		{
			"adding", "image: foo\n", map[string]string{"image": "bar", "something": "else"}, []byte("image: bar\nsomething: else\n"), false, "",
		},
		{
			"array item replacement", "array:\n - asdf\n - qwerty\n", map[string]string{"array[1]": "bar"}, []byte("array:\n- asdf\n- bar\n"), false, "",
		},
		{
			"nested", "image:\n tag: foo\n", map[string]string{"image.tag": "bar"}, []byte("image:\n  tag: bar\n"), false, "",
		},
		{
			"nested array item replacement", "image:\n stuff:\n  - asdf\n", map[string]string{"image.stuff[0]": "1234"}, []byte("image:\n  stuff:\n  - 1234\n"), false, "",
		},
		{
			"integer", "count: 1\n", map[string]string{"count": "75"}, []byte("count: 75\n"), false, "",
		},
		{
			"boolean", "enabled: true\n", map[string]string{"enabled": "false"}, []byte("enabled: false\n"), false, "",
		},
	}

	cl := ChartLocation{
		VarFilePath: "foo.yml",
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := memfs.New()
			f, _ := fs.Create(cl.VarFilePath)
			f.Write([]byte(c.inputYAML))
			f.Close()
			out, err := cl.MergeVars(fs, c.inputOverrides)
			if err != nil {
				if c.isError {
					if !strings.Contains(err.Error(), c.errContains) {
						t.Fatalf("error does not contain expected string (%v): %v", c.errContains, err)
					}
				} else {
					t.Fatalf("should have succeeded: %v", err)
				}
			}
			if !bytes.Equal(out, c.output) {
				fmt.Printf("out: %v\n", string(out))
				fmt.Printf("wanted: %v\n", string(c.output))
				t.Fatalf("bad output: %v; expected: %v", out, c.output)
			}
		})
	}
}

func TestMetahelmBuildAndInstallCharts(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	var server net.Conn
	go func() {
		defer ln.Close()
		time.Sleep(10 * time.Millisecond)
		server, _ = ln.Accept()
	}()
	server = server
	asl := strings.Split(ln.Addr().String(), ":")
	ip := asl[0]
	port, _ := strconv.Atoi(asl[1])
	tcfg := TillerConfig{
		Port:                    uint(port),
		ServerConnectRetryDelay: 1 * time.Millisecond,
	}
	tcfg = tcfg.SetDefaults()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "foo",
			Labels:    map[string]string{"app": "helm"},
		},
		Status: v1.PodStatus{
			PodIP: ip,
		},
	}
	cl := ChartLocations{
		"foo": ChartLocation{ChartPath: "foo/bar"},
		"bar": ChartLocation{ChartPath: "bar/baz"},
	}
	charts := []metahelm.Chart{
		metahelm.Chart{Title: "foo", Location: "foo/bar", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "foo", DependencyList: []string{"bar"}},
		metahelm.Chart{Title: "bar", Location: "bar/baz", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "bar"},
	}
	tobjs := gentestobjs(charts)
	tobjs = append(tobjs, pod)
	fkc := fake.NewSimpleClientset(tobjs...)
	ib := &images.FakeImageBuilder{BatchCompletedFunc: func(envname, repo string) (bool, error) { return true, nil }}
	rc := &models.RepoConfig{
		Application: models.RepoConfigAppMetadata{
			Repo:          "foo",
			Ref:           "asdf",
			Branch:        "feature-foo",
			Image:         "foo",
			ChartTagValue: "image.tag",
		},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "bar",
					Repo: "bar",
					AppMetadata: models.RepoConfigAppMetadata{
						Repo:          "bar",
						Ref:           "asdf",
						Branch:        "feature-foo",
						Image:         "bar",
						ChartTagValue: "image.tag",
					},
				},
			},
		},
	}
	nenv := &EnvInfo{
		Env: &models.QAEnvironment{Name: "foo-bar"},
		RC:  rc,
	}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), nenv.Env)
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
		ib: ib,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{}, nil
		},
		tcfg: tcfg,
		mc:   &metrics.FakeCollector{},
	}
	metahelm.ChartWaitPollInterval = 10 * time.Millisecond
	overrideNamespace = "foo"
	defer func() { overrideNamespace = "" }()
	if err := ci.BuildAndInstallCharts(context.Background(), nenv, cl); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestMetahelmBuildAndUpgradeCharts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	var server net.Conn
	go func() {
		defer ln.Close()
		time.Sleep(10 * time.Millisecond)
		server, err = ln.Accept()
	}()
	server = server
	asl := strings.Split(ln.Addr().String(), ":")
	ip := asl[0]
	port, _ := strconv.Atoi(asl[1])
	tcfg := TillerConfig{
		Port:                    uint(port),
		ServerConnectRetryDelay: 1 * time.Millisecond,
	}
	tcfg = tcfg.SetDefaults()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiller",
			Namespace: "foo",
			Labels:    map[string]string{"app": "helm"},
		},
		Status: v1.PodStatus{
			PodIP: ip,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultTillerDeploymentName,
			Namespace: "foo",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
		},
	}
	cl := ChartLocations{
		"foo": ChartLocation{ChartPath: "foo/bar"},
		"bar": ChartLocation{ChartPath: "bar/baz"},
	}
	charts := []metahelm.Chart{
		metahelm.Chart{Title: "foo", Location: "foo/bar", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "foo", DependencyList: []string{"bar"}},
		metahelm.Chart{Title: "bar", Location: "bar/baz", DeploymentHealthIndication: metahelm.AtLeastOnePodHealthy, WaitUntilDeployment: "bar"},
	}
	tobjs := gentestobjs(charts)
	tobjs = append(tobjs, pod, deployment)
	fkc := fake.NewSimpleClientset(tobjs...)
	ib := &images.FakeImageBuilder{BatchCompletedFunc: func(envname, repo string) (bool, error) { return true, nil }}
	stop := make(chan struct{})
	defer close(stop)
	rm := match.RefMap{"foo": match.BranchInfo{Name: "master", SHA: "aaaa"}, "bar": match.BranchInfo{Name: "foo", SHA: "bbbbbb"}}
	rc := &models.RepoConfig{
		Application: models.RepoConfigAppMetadata{
			Repo:          "foo",
			Ref:           rm["foo"].SHA,
			Image:         "foo",
			ChartTagValue: "image.tag",
		},
		Dependencies: models.DependencyDeclaration{
			Direct: []models.RepoConfigDependency{
				models.RepoConfigDependency{
					Name: "bar",
					Repo: "bar",
					AppMetadata: models.RepoConfigAppMetadata{
						Repo:          "bar",
						Ref:           rm["bar"].SHA,
						Image:         "bar",
						ChartTagValue: "image.tag",
					},
				},
			},
		},
	}
	nenv := &EnvInfo{
		Env: &models.QAEnvironment{Name: "foo-bar"},
		RC:  rc,
		Releases: map[string]string{
			"foo": "foo-release",
			"bar": "bar-release",
		},
	}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), nenv.Env)
	dl.CreateK8sEnv(context.Background(), &models.KubernetesEnvironment{
		EnvName:    nenv.Env.Name,
		Namespace:  "foo",
		TillerAddr: ln.Addr().String(),
	})
	rlses := []models.HelmRelease{
		models.HelmRelease{
			EnvName:     nenv.Env.Name,
			Name:        "foo",
			Release:     nenv.Releases["foo"],
			RevisionSHA: "1234",
		},
		models.HelmRelease{
			EnvName:     nenv.Env.Name,
			Name:        "bar",
			Release:     nenv.Releases["bar"],
			RevisionSHA: "5678",
		},
	}
	k8senv := &models.KubernetesEnvironment{
		EnvName:   nenv.Env.Name,
		Namespace: "foo",
	}
	dl.CreateK8sEnv(context.Background(), k8senv)
	dl.CreateHelmReleasesForEnv(context.Background(), rlses)
	rels := []*rls.Release{}
	for _, r := range rlses {
		rels = append(rels, &rls.Release{Name: r.Release})
	}
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
		ib: ib,
		hcf: func(tillerNS, tillerAddr string, rcfg *rest.Config, kc kubernetes.Interface) (helm.Interface, error) {
			return &helm.FakeClient{Rels: rels}, nil
		},
		tcfg: tcfg,
		mc:   &metrics.FakeCollector{},
	}
	metahelm.ChartWaitPollInterval = 10 * time.Millisecond
	overrideNamespace = "foo"
	defer func() { overrideNamespace = "" }()
	if err := ci.BuildAndUpgradeCharts(context.Background(), nenv, k8senv, cl); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	releases, err := dl.GetHelmReleasesForEnv(context.Background(), nenv.Env.Name)
	if err != nil {
		t.Fatalf("get helm releases should have succeeded: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("bad release count: %v", len(releases))
	}
	for _, r := range releases {
		if r.Name != "foo" && r.Name != "bar" {
			t.Fatalf("bad release name: %v", r.Name)
		}
		for _, n := range []string{"foo", "bar"} {
			if r.Name == n {
				if r.RevisionSHA != rm[n].SHA {
					t.Fatalf("bad revision for %v release: %v", n, r.RevisionSHA)
				}
			}
		}
	}
}

func TestMetahelmDeleteNamespace(t *testing.T) {
	nenv := &EnvInfo{
		Env: &models.QAEnvironment{Name: "foo-bar"},
	}
	k8senv := &models.KubernetesEnvironment{
		EnvName:   nenv.Env.Name,
		Namespace: "foo",
	}
	dl := persistence.NewFakeDataLayer()
	dl.CreateQAEnvironment(context.Background(), nenv.Env)
	dl.CreateK8sEnv(context.Background(), k8senv)
	fkc := fake.NewSimpleClientset(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	ci := ChartInstaller{kc: fkc, dl: dl}
	if err := ci.DeleteNamespace(context.Background(), k8senv); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	ke, err := dl.GetK8sEnv(context.Background(), nenv.Env.Name)
	if err != nil {
		t.Fatalf("get k8s env should have succeeded: %v", err)
	}
	if ke != nil {
		t.Fatalf("get k8s env should have returned nothing: %v", ke)
	}
}

type fakeSecretFetcher struct{}

func (fsf *fakeSecretFetcher) Get(id string) ([]byte, error) { return []byte{}, nil }

func TestMetahelmSetupNamespace(t *testing.T) {
	dl := persistence.NewFakeDataLayer()
	fkc := fake.NewSimpleClientset(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	k8scfg := config.K8sConfig{}
	k8scfg.ProcessGroupBindings("foo=edit")
	k8scfg.ProcessPrivilegedRepos("foo/bar")
	k8scfg.ProcessSecretInjections(&fakeSecretFetcher{}, "mysecret=some/vault/path")
	ci := ChartInstaller{kc: fkc, dl: dl, k8sgroupbindings: k8scfg.GroupBindings, k8srepowhitelist: k8scfg.PrivilegedRepoWhitelist, k8ssecretinjs: k8scfg.SecretInjections}
	if err := ci.setupNamespace(context.Background(), "some-name", "foo/bar", "foo"); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
}

func TestMetahelmCleanup(t *testing.T) {
	maxAge := 1 * time.Hour
	expires := time.Now().UTC().Add(-(maxAge + (72 * time.Hour)))
	orphanedNamespaces := []*v1.Namespace{
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "foo",
				CreationTimestamp: meta.NewTime(expires),
				Labels: map[string]string{
					objLabelKey: objLabelValue,
				},
			},
		},
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bar",
				CreationTimestamp: meta.NewTime(expires),
				Labels: map[string]string{
					objLabelKey: objLabelValue,
				},
			},
		},
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "uninvolved",
				CreationTimestamp: meta.NewTime(expires),
			},
		},
	}
	orphanedCRBs := []*rbacv1.ClusterRoleBinding{
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "foo",
				CreationTimestamp: meta.NewTime(expires),
				Labels: map[string]string{
					objLabelKey: objLabelValue,
				},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bar",
				CreationTimestamp: meta.NewTime(expires),
				Labels: map[string]string{
					objLabelKey: objLabelValue,
				},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "uninvolved",
				CreationTimestamp: meta.NewTime(expires),
			},
		},
	}
	var objs []runtime.Object
	for _, ns := range orphanedNamespaces {
		objs = append(objs, ns)
	}
	for _, crb := range orphanedCRBs {
		objs = append(objs, crb)
	}
	fkc := fake.NewSimpleClientset(objs...)
	dl := persistence.NewFakeDataLayer()
	ci := ChartInstaller{
		kc: fkc,
		dl: dl,
	}
	ci.Cleanup(context.Background(), maxAge)
	if _, err := fkc.CoreV1().Namespaces().Get("foo", metav1.GetOptions{}); err == nil {
		t.Fatalf("should have failed to find namespace foo")
	}
	if _, err := fkc.CoreV1().Namespaces().Get("bar", metav1.GetOptions{}); err == nil {
		t.Fatalf("should have failed to find namespace bar")
	}
	if _, err := fkc.CoreV1().Namespaces().Get("uninvolved", metav1.GetOptions{}); err != nil {
		t.Fatalf("should have found namespace uninvolved: %v", err)
	}
	if _, err := fkc.RbacV1().ClusterRoleBindings().Get("foo", metav1.GetOptions{}); err == nil {
		t.Fatalf("should have failed to find CRB foo")
	}
	if _, err := fkc.RbacV1().ClusterRoleBindings().Get("bar", metav1.GetOptions{}); err == nil {
		t.Fatalf("should have failed to find CRB bar")
	}
	if _, err := fkc.RbacV1().ClusterRoleBindings().Get("uninvolved", metav1.GetOptions{}); err != nil {
		t.Fatalf("should have found CRB uninvolved: %v", err)
	}
}

func TestTruncateLongDQAName(t *testing.T) {
	testCases := []struct {
		input          string
		expectedOutput string
	}{
		{
			input:          "amino-qa-80093-anosmic-basal-body-temperature-method-of-family-planning",
			expectedOutput: "amino-qa-80093-anosmic-basal-body-temperature-method-of-family",
		},
		{
			input:          "small-input",
			expectedOutput: "small-input",
		},
		{
			input:          "amino-qa-80093-anosmic-basal-body-temperature-method-of-familyȨȨȨȨȨ-planning",
			expectedOutput: "amino-qa-80093-anosmic-basal-body-temperature-method-of-family",
		},
	}

	for _, test := range testCases {
		output := truncateToDNS1123Label(test.input)
		if output != test.expectedOutput {
			t.Fatalf("string was truncated incorrectly (%v)", test.input)
		}
	}
}

func TestMetahelmGetK8sEnvPodList(t *testing.T) {
	ci := FakeKubernetesReporter{}
	pl, err := ci.GetPodList(context.Background(), "foo")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if len(pl) != 2 {
		t.Fatalf("expected 2, got %v", len(pl))
	}
	for _, p := range pl {
		if p.Ready != "1/1" && p.Status != "Running" {
			t.Fatalf("expected Ready: 1/1 & Status: Running, got %v, %v", p.Ready, p.Status)
		}
	}
}

func TestMetahelmGetK8sEnvPodContainers(t *testing.T) {
	ci := FakeKubernetesReporter{}
	pc, err := ci.GetPodContainers(context.Background(), "foo", "foo-app-abc123")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if len(pc.Containers) != 2 {
		t.Fatalf("expected 2, got %v", len(pc.Containers))
	}
	pc, err = ci.GetPodContainers(context.Background(), "foo", "bar-app-abc123")
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if len(pc.Containers) != 1 {
		t.Fatalf("expected 1, got %v", len(pc.Containers))
	}
	pc, err = ci.GetPodContainers(context.Background(), "foo", "baz-app-abc123")
	if err == nil {
		t.Fatalf("should have failed: %v", err)
	}
}

func TestMetahelmGetK8sEnvPodLogs(t *testing.T) {
	ci := FakeKubernetesReporter{
		FakePodLogFilePath: "testdata/pod_logs.log",
	}
	nLogLines := MaxPodContainerLogLines + 1
	_, err := ci.GetPodLogs(context.Background(), "foo", "foo-app-abc123", "", uint(nLogLines))
	if err != nil {
		t.Fatalf("should have failed: %v", err)
	}
	nLogLines--
	pl, err := ci.GetPodLogs(context.Background(), "foo", "foo-app-abc123", "", uint(nLogLines))
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	defer pl.Close()
	buf := make([]byte, 68*1024)
	_, err = pl.Read(buf)
	if err != nil {
		t.Fatalf("error reading pod logs: %v", err)
	}
	lineCount := bytes.Count(buf, []byte{'\n'})
	if lineCount > nLogLines {
		t.Fatalf("error lines returned exceeded expected %v, actual %v", nLogLines, lineCount)
	}
}
