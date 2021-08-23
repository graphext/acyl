package metahelm

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	mtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func TestErrorPopulateFromRelease(t *testing.T) {
	rls := &release.Release{
		Namespace: DefaultK8sNamespace,
		Manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  labels:
    app: foo
    chart: foo-0.1.0
    release: foo-release
    heritage: Tiller
---
apiVersion: batch/v1
kind: Job
metadata:
  name: bar
  labels:
    job: bar
    chart: bar-0.1.0
    release: bar-release
    heritage: Tiller`,
	}
	r := &appsv1.ReplicaSet{}
	d := &appsv1.Deployment{}
	j := &batchv1.Job{}
	j.ObjectMeta.Name = "bar"
	j.Namespace = DefaultK8sNamespace
	j.Spec.Selector = &metav1.LabelSelector{}
	j.Spec.Selector.MatchLabels = map[string]string{"job": "bar"}
	p := &corev1.Pod{}
	p.ObjectMeta.Name = "foo-1234"
	p.Namespace = DefaultK8sNamespace
	p.ObjectMeta.Labels = map[string]string{"app": "foo"}
	p.Status = corev1.PodStatus{
		Phase: corev1.PodFailed,
		Conditions: []corev1.PodCondition{
			corev1.PodCondition{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			corev1.ContainerStatus{
				Name: "foo",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 128,
					},
				},
			},
		},
	}
	pj := &corev1.Pod{}
	pj.ObjectMeta.Name = "bar-1234"
	pj.Namespace = DefaultK8sNamespace
	pj.ObjectMeta.Labels = map[string]string{"job": "bar"}
	pj.Status = corev1.PodStatus{
		Phase: corev1.PodSucceeded,
		Conditions: []corev1.PodCondition{
			corev1.PodCondition{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			corev1.ContainerStatus{
				Name: "bar",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 0,
					},
				},
			},
		},
	}
	reps := int32(1)
	d.Spec.Replicas = &reps
	d.Spec.Template.Labels = map[string]string{"app": "foo"}
	d.Spec.Template.Spec.NodeSelector = map[string]string{}
	d.Spec.Template.Name = "foo"
	d.Spec.Selector = &metav1.LabelSelector{}
	d.Spec.Selector.MatchLabels = map[string]string{"app": "foo"}
	r.Spec.Selector = d.Spec.Selector
	r.Spec.Replicas = &reps
	r.Status.ReadyReplicas = 1
	r.Name = "replicaset-" + "foo"
	r.Namespace = DefaultK8sNamespace
	r.Labels = d.Spec.Template.Labels
	d.Labels = d.Spec.Template.Labels
	d.ObjectMeta.UID = mtypes.UID("foo" + "-deployment")
	iscontroller := true
	r.ObjectMeta.OwnerReferences = []metav1.OwnerReference{metav1.OwnerReference{UID: d.ObjectMeta.UID, Controller: &iscontroller}}
	d.Name = "foo"
	d.ObjectMeta.Name = "foo"
	d.Namespace = DefaultK8sNamespace
	r.Spec.Template = d.Spec.Template
	kc := fake.NewSimpleClientset(r, d, j, p, pj)
	ce := NewChartError(errors.New("some helm error"))
	err := ce.PopulateFromRelease(rls, kc, 100)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if i := len(ce.FailedDeployments); i != 1 {
		t.Fatalf("unexpected length for failed deployments: %v", i)
	}
	if i := len(ce.FailedJobs); i != 0 {
		t.Fatalf("unexpected length for failed jobs: %v", i)
	}
	fp, ok := ce.FailedDeployments["foo"]
	if !ok {
		t.Fatalf("foo missing")
	}
	if i := len(fp); i != 1 {
		t.Fatalf("unexpected length for failed pod: %v", i)
	}
	if ec := fp[0].ContainerStatuses[0].State.Terminated.ExitCode; ec != 128 {
		t.Fatalf("bad exit code: %v", ec)
	}
}

func TestErrorPopulateFromDeployment(t *testing.T) {
	r := &appsv1.ReplicaSet{}
	d := &appsv1.Deployment{}
	p := &corev1.Pod{}
	p.ObjectMeta.Name = "foo-1234"
	p.Namespace = DefaultK8sNamespace
	p.ObjectMeta.Labels = map[string]string{"app": "foo"}
	p.Status = corev1.PodStatus{
		Phase: corev1.PodFailed,
		Conditions: []corev1.PodCondition{
			corev1.PodCondition{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			corev1.ContainerStatus{
				Name: "foo",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 128,
					},
				},
			},
		},
	}
	reps := int32(1)
	d.Spec.Replicas = &reps
	d.Spec.Template.Labels = map[string]string{"app": "foo"}
	d.Spec.Template.Spec.NodeSelector = map[string]string{}
	d.Spec.Template.Name = "foo"
	d.Spec.Selector = &metav1.LabelSelector{}
	d.Spec.Selector.MatchLabels = map[string]string{"app": "foo"}
	r.Spec.Selector = d.Spec.Selector
	r.Spec.Replicas = &reps
	r.Status.ReadyReplicas = 1
	r.Name = "replicaset-" + "foo"
	r.Namespace = DefaultK8sNamespace
	r.Labels = d.Spec.Template.Labels
	d.Labels = d.Spec.Template.Labels
	d.ObjectMeta.UID = mtypes.UID("foo" + "-deployment")
	iscontroller := true
	r.ObjectMeta.OwnerReferences = []metav1.OwnerReference{metav1.OwnerReference{UID: d.ObjectMeta.UID, Controller: &iscontroller}}
	d.Name = "foo"
	d.ObjectMeta.Name = "foo"
	d.Namespace = DefaultK8sNamespace
	r.Spec.Template = d.Spec.Template
	kc := fake.NewSimpleClientset(r, d, p)
	ce := NewChartError(errors.New("some helm error"))
	err := ce.PopulateFromDeployment(DefaultK8sNamespace, "foo", kc, 500)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if i := len(ce.FailedDeployments); i != 1 {
		t.Fatalf("unexpected length for failed deployments: %v", i)
	}
	fp, ok := ce.FailedDeployments["foo"]
	if !ok {
		t.Fatalf("foo missing")
	}
	if i := len(fp); i != 1 {
		t.Fatalf("unexpected length for failed pod: %v", i)
	}
	if ec := fp[0].ContainerStatuses[0].State.Terminated.ExitCode; ec != 128 {
		t.Fatalf("bad exit code: %v", ec)
	}
}

func TestUnmarshalError(t *testing.T) {
	r := &appsv1.ReplicaSet{}
	d := &appsv1.Deployment{}
	p := &corev1.Pod{}
	p.ObjectMeta.Name = "foo-1234"
	p.Namespace = DefaultK8sNamespace
	p.ObjectMeta.Labels = map[string]string{"app": "foo"}
	p.Status = corev1.PodStatus{
		Phase: corev1.PodFailed,
		Conditions: []corev1.PodCondition{
			corev1.PodCondition{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			corev1.ContainerStatus{
				Name: "foo",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 128,
					},
				},
			},
		},
	}
	reps := int32(1)
	d.Spec.Replicas = &reps
	d.Spec.Template.Labels = map[string]string{"app": "foo"}
	d.Spec.Template.Spec.NodeSelector = map[string]string{}
	d.Spec.Template.Name = "foo"
	d.Spec.Selector = &metav1.LabelSelector{}
	d.Spec.Selector.MatchLabels = map[string]string{"app": "foo"}
	r.Spec.Selector = d.Spec.Selector
	r.Spec.Replicas = &reps
	r.Status.ReadyReplicas = 1
	r.Name = "replicaset-" + "foo"
	r.Namespace = DefaultK8sNamespace
	r.Labels = d.Spec.Template.Labels
	d.Labels = d.Spec.Template.Labels
	d.ObjectMeta.UID = mtypes.UID("foo" + "-deployment")
	iscontroller := true
	r.ObjectMeta.OwnerReferences = []metav1.OwnerReference{metav1.OwnerReference{UID: d.ObjectMeta.UID, Controller: &iscontroller}}
	d.Name = "foo"
	d.ObjectMeta.Name = "foo"
	d.Namespace = DefaultK8sNamespace
	r.Spec.Template = d.Spec.Template
	kc := fake.NewSimpleClientset(r, d, p)
	ce := NewChartError(errors.New("some helm error"))
	err := ce.PopulateFromDeployment(DefaultK8sNamespace, "foo", kc, 500)
	if err != nil {
		t.Fatalf("could not populate chart error from deployment: %v", err)
	}

	encodedCE, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("unable to marshal error: %v", err)
	}

	var outputCE ChartError
	if err := json.Unmarshal(encodedCE, &outputCE); err != nil {
		t.Fatalf("error unmarshaling chart error: %v", err)
	}
}
