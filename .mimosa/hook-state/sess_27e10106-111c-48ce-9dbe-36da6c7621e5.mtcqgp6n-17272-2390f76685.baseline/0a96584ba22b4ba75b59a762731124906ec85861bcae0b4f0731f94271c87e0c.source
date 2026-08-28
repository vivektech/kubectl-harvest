package resource

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// externalSecretGVRs lists the GroupVersionResources of the ExternalSecret
// custom resources of the major external-secret operators. Listing is
// best-effort: when an operator is not installed the CRD does not exist and
// the list call is skipped.
var externalSecretGVRs = []schema.GroupVersionResource{
	{Group: "external-secrets.io", Version: "v1", Resource: "externalsecrets"},
	{Group: "external-secrets.io", Version: "v1beta1", Resource: "externalsecrets"},
	{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"},
}

type Client interface {
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]*appsv1.ReplicaSet, error)
	ListDeployments(ctx context.Context, namespace string) ([]*appsv1.Deployment, error)
	ListStatefulSets(ctx context.Context, namespace string) ([]*appsv1.StatefulSet, error)
	ListDaemonSets(ctx context.Context, namespace string) ([]*appsv1.DaemonSet, error)
	ListJobs(ctx context.Context, namespace string) ([]*batchv1.Job, error)
	ListCronJobs(ctx context.Context, namespace string) ([]*batchv1.CronJob, error)
	ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error)
	ListIngresses(ctx context.Context, namespace string) ([]*networkingv1.Ingress, error)
	ListExternalSecrets(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error)
	GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error)
}

type client struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// Guarantee *client implements Client.
var _ Client = (*client)(nil)

func NewClient(clientset kubernetes.Interface, dynamicClient dynamic.Interface) Client {
	return &client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

func (c *client) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	return pods, nil
}

func (c *client) ListReplicaSets(ctx context.Context, namespace string) ([]*appsv1.ReplicaSet, error) {
	rsList, err := c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rss := make([]*appsv1.ReplicaSet, 0, len(rsList.Items))
	for i := range rsList.Items {
		rss = append(rss, &rsList.Items[i])
	}

	return rss, nil
}

func (c *client) ListDeployments(ctx context.Context, namespace string) ([]*appsv1.Deployment, error) {
	deployList, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]*appsv1.Deployment, 0, len(deployList.Items))
	for i := range deployList.Items {
		deployments = append(deployments, &deployList.Items[i])
	}

	return deployments, nil
}

func (c *client) ListStatefulSets(ctx context.Context, namespace string) ([]*appsv1.StatefulSet, error) {
	stsList, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	stss := make([]*appsv1.StatefulSet, 0, len(stsList.Items))
	for i := range stsList.Items {
		stss = append(stss, &stsList.Items[i])
	}

	return stss, nil
}

func (c *client) ListDaemonSets(ctx context.Context, namespace string) ([]*appsv1.DaemonSet, error) {
	dsList, err := c.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	dss := make([]*appsv1.DaemonSet, 0, len(dsList.Items))
	for i := range dsList.Items {
		dss = append(dss, &dsList.Items[i])
	}

	return dss, nil
}

func (c *client) ListJobs(ctx context.Context, namespace string) ([]*batchv1.Job, error) {
	jobList, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	jobs := make([]*batchv1.Job, 0, len(jobList.Items))
	for i := range jobList.Items {
		jobs = append(jobs, &jobList.Items[i])
	}

	return jobs, nil
}

func (c *client) ListCronJobs(ctx context.Context, namespace string) ([]*batchv1.CronJob, error) {
	cronJobList, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// batch/v1 CronJobs are served since Kubernetes 1.21. On older
		// clusters degrade gracefully instead of failing the whole command.
		if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
			return nil, nil
		}
		return nil, err
	}

	cronJobs := make([]*batchv1.CronJob, 0, len(cronJobList.Items))
	for i := range cronJobList.Items {
		cronJobs = append(cronJobs, &cronJobList.Items[i])
	}

	return cronJobs, nil
}

func (c *client) ListServiceAccounts(ctx context.Context, namespace string) ([]*corev1.ServiceAccount, error) {
	saList, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	sas := make([]*corev1.ServiceAccount, 0, len(saList.Items))
	for i := range saList.Items {
		sas = append(sas, &saList.Items[i])
	}

	return sas, nil
}

func (c *client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	pvcList, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pvcs := make([]*corev1.PersistentVolumeClaim, 0, len(pvcList.Items))
	for i := range pvcList.Items {
		pvcs = append(pvcs, &pvcList.Items[i])
	}

	return pvcs, nil
}

func (c *client) ListIngresses(ctx context.Context, namespace string) ([]*networkingv1.Ingress, error) {
	ingressList, err := c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// networking.k8s.io/v1 Ingresses are served since Kubernetes 1.19.
		// On older clusters degrade gracefully instead of failing the whole
		// command.
		if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
			return nil, nil
		}
		return nil, err
	}

	ingresses := make([]*networkingv1.Ingress, 0, len(ingressList.Items))
	for i := range ingressList.Items {
		ingresses = append(ingresses, &ingressList.Items[i])
	}

	return ingresses, nil
}

func (c *client) ListExternalSecrets(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error) {
	seen := make(map[string]struct{})
	var externalSecrets []*unstructured.Unstructured

	for _, gvr := range externalSecretGVRs {
		list, err := c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// The CRD is not served when the operator is not installed.
			// Anything else (e.g. RBAC) is not fatal either: reaping must not
			// break just because one reference source cannot be inspected.
			if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
				continue
			}
			return nil, err
		}

		for i := range list.Items {
			es := &list.Items[i]
			// The same object can be served under multiple versions of the
			// same group; keep each object once.
			key := es.GroupVersionKind().Group + "/" + es.GetNamespace() + "/" + es.GetName()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			externalSecrets = append(externalSecrets, es)
		}
	}

	return externalSecrets, nil
}

func (c *client) GetUnstructured(ctx context.Context, apiVersion, kind, name, namespace string) (*unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	gvk := gv.WithKind(kind)
	gvr, _ := apimeta.UnsafeGuessKindToResource(gvk)

	u, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		return u, nil
	case apierrors.IsNotFound(err):
		return nil, nil
	default:
		return nil, err
	}
}
