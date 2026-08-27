package resource

import (
	"context"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type FakeClient struct {
	fakeObjects map[fakeObjectKey]runtime.Object

	fakePods                   []*corev1.Pod
	fakeReplicaSets            []*appsv1.ReplicaSet
	fakeDeployments            []*appsv1.Deployment
	fakeStatefulSets           []*appsv1.StatefulSet
	fakeDaemonSets             []*appsv1.DaemonSet
	fakeJobs                   []*batchv1.Job
	fakeCronJobs               []*batchv1.CronJob
	fakeServiceAccounts        []*corev1.ServiceAccount
	fakePersistentVolumeClaims []*corev1.PersistentVolumeClaim
	fakeIngresses              []*networkingv1.Ingress
	fakeExternalSecrets        []*unstructured.Unstructured

	mu sync.RWMutex
}

type fakeObjectKey struct {
	apiVersion string
	kind       string
	name       string
	namespace  string
}

func NewFakeClient(objects ...runtime.Object) (*FakeClient, error) {
	c := &FakeClient{
		fakeObjects: make(map[fakeObjectKey]runtime.Object),
	}

	accessor := apimeta.NewAccessor()

	for _, obj := range objects {
		kind, err := accessor.Kind(obj)
		if err != nil {
			return nil, err
		}

		switch o := obj.(type) {
		case *corev1.Pod:
			c.fakePods = append(c.fakePods, o)
		case *appsv1.ReplicaSet:
			c.fakeReplicaSets = append(c.fakeReplicaSets, o)
		case *appsv1.Deployment:
			c.fakeDeployments = append(c.fakeDeployments, o)
		case *appsv1.StatefulSet:
			c.fakeStatefulSets = append(c.fakeStatefulSets, o)
		case *appsv1.DaemonSet:
			c.fakeDaemonSets = append(c.fakeDaemonSets, o)
		case *batchv1.Job:
			c.fakeJobs = append(c.fakeJobs, o)
		case *batchv1.CronJob:
			c.fakeCronJobs = append(c.fakeCronJobs, o)
		case *corev1.ServiceAccount:
			c.fakeServiceAccounts = append(c.fakeServiceAccounts, o)
		case *corev1.PersistentVolumeClaim:
			c.fakePersistentVolumeClaims = append(c.fakePersistentVolumeClaims, o)
		case *networkingv1.Ingress:
			c.fakeIngresses = append(c.fakeIngresses, o)
		case *unstructured.Unstructured:
			if o.GetKind() == "ExternalSecret" {
				c.fakeExternalSecrets = append(c.fakeExternalSecrets, o)
			}
		}

		apiVersion, err := accessor.APIVersion(obj)
		if err != nil {
			return nil, err
		}
		name, err := accessor.Name(obj)
		if err != nil {
			return nil, err
		}
		namespace, err := accessor.Namespace(obj)
		if err != nil {
			return nil, err
		}

		key := fakeObjectKey{
			apiVersion: apiVersion,
			kind:       kind,
			name:       name,
			namespace:  namespace,
		}
		c.fakeObjects[key] = obj
	}

	return c, nil
}

// Guarantee *FakeClient implements Client.
var _ Client = (*FakeClient)(nil)

func (c *FakeClient) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakePods, nil
}

func (c *FakeClient) ListReplicaSets(ctx context.Context, namespace string) ([]*appsv1.ReplicaSet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeReplicaSets, nil
}

func (c *FakeClient) ListDeployments(ctx context.Context, namespace string) ([]*appsv1.Deployment, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeDeployments, nil
}

func (c *FakeClient) ListStatefulSets(ctx context.Context, namespace string) ([]*appsv1.StatefulSet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeStatefulSets, nil
}

func (c *FakeClient) ListDaemonSets(ctx context.Context, namespace string) ([]*appsv1.DaemonSet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeDaemonSets, nil
}

func (c *FakeClient) ListJobs(ctx context.Context, namespace string) ([]*batchv1.Job, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeJobs, nil
}

func (c *FakeClient) ListCronJobs(ctx context.Context, namespace string) ([]*batchv1.CronJob, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeCronJobs, nil
}

func (c *FakeClient) ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeServiceAccounts, nil
}

func (c *FakeClient) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakePersistentVolumeClaims, nil
}

func (c *FakeClient) ListIngresses(ctx context.Context, namespace string) ([]*networkingv1.Ingress, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeIngresses, nil
}

func (c *FakeClient) ListExternalSecrets(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fakeExternalSecrets, nil
}

func (c *FakeClient) GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error) {
	key := fakeObjectKey{
		apiVersion: apiVersion,
		kind:       kind,
		name:       name,
		namespace:  namespace,
	}

	c.mu.RLock()
	obj, ok := c.fakeObjects[key]
	c.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	u, err := unstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: u,
	}, nil
}
