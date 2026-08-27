package determiner

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/vivektech/kubectl-harvest/pkg/resource"
)

const (
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"

	// rootCAConfigMapName is the ConfigMap the root CA publisher injects into
	// every namespace. Pods consume it implicitly, so reference scanning can
	// never see it. It must never be reaped.
	rootCAConfigMapName = "kube-root-ca.crt"
)

// ingressSecretAnnotations are the nginx ingress controller annotations that
// reference Secrets by name. They are invisible to Pod-based reference
// scanning, so they are checked explicitly.
var ingressSecretAnnotations = []string{
	"nginx.ingress.kubernetes.io/auth-secret",
	"nginx.ingress.kubernetes.io/auth-tls-secret",
}

var checkVolumeSatisfyClaimFunc = resource.CheckVolumeSatisfyClaim

type Determiner interface {
	DetermineDeletion(ctx context.Context, info *cliresource.Info) (bool, error)
}

// podSpecRef carries a PodSpec together with the namespace and the labels of
// the object it was extracted from, so that references can be tracked per
// namespace even when running with --all-namespaces, and label selectors can
// be matched against workload Pod templates and not only live Pods.
type podSpecRef struct {
	namespace string
	labels    map[string]string
	spec      corev1.PodSpec
}

// determiner determines whether a resource should be deleted.
type determiner struct {
	resourceClient resource.Client

	// used* maps are keyed by "<namespace>/<name>".
	usedConfigMaps             map[string]struct{}
	usedSecrets                map[string]struct{}
	usedPersistentVolumeClaims map[string]struct{}

	pods                   []*corev1.Pod
	podSpecs               []podSpecRef
	replicaSets            []*appsv1.ReplicaSet
	deployments            []*appsv1.Deployment
	persistentVolumeClaims []*corev1.PersistentVolumeClaim

	// keepRevisions is the number of newest ReplicaSet revisions of each
	// Deployment that are always kept when reaping ReplicaSets, so that
	// rollback via `kubectl rollout undo` remains possible. 0 means only
	// the live revision is kept.
	keepRevisions int
}

// Guarantee *determiner implements Determiner.
var _ Determiner = (*determiner)(nil)

// Options configures the determiner.
type Options struct {
	// KeepRevisions always keeps the newest N ReplicaSet revisions of each
	// Deployment when reaping ReplicaSets (the live revision is always kept
	// regardless). 0 keeps only the live revision.
	KeepRevisions int
}

func New(resourceClient resource.Client, r *cliresource.Result, namespace string, opts Options) (Determiner, error) {
	d := &determiner{
		resourceClient: resourceClient,
		keepRevisions:  opts.KeepRevisions,
	}

	var (
		reapConfigMaps             bool
		reapSecrets                bool
		reapPersistentVolumes      bool
		reapPersistentVolumeClaims bool
		reapPodDisruptionBudgets   bool
		reapReplicaSets            bool
		reapNetworkPolicies        bool
	)

	if err := r.Visit(func(info *cliresource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case resource.KindConfigMap:
			reapConfigMaps = true
		case resource.KindSecret:
			reapSecrets = true
		case resource.KindPersistentVolume:
			reapPersistentVolumes = true
		case resource.KindPersistentVolumeClaim:
			reapPersistentVolumeClaims = true
		case resource.KindPodDisruptionBudget:
			reapPodDisruptionBudgets = true
		case resource.KindReplicaSet:
			reapReplicaSets = true
		case resource.KindNetworkPolicy:
			reapNetworkPolicies = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	ctx := context.Background()

	needsPodTemplates := reapConfigMaps || reapSecrets
	needsPods := needsPodTemplates || reapPersistentVolumeClaims || reapPodDisruptionBudgets || reapNetworkPolicies

	if needsPods {
		var err error
		d.pods, err = d.resourceClient.ListPods(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, pod := range d.pods {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: pod.Namespace, labels: pod.Labels, spec: pod.Spec})
		}
	}

	if needsPodTemplates || reapReplicaSets {
		var err error
		d.replicaSets, err = d.resourceClient.ListReplicaSets(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, rs := range d.replicaSets {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: rs.Namespace, labels: rs.Spec.Template.Labels, spec: rs.Spec.Template.Spec})
		}
	}

	if needsPodTemplates || reapReplicaSets {
		var err error
		d.deployments, err = d.resourceClient.ListDeployments(ctx, namespace)
		if err != nil {
			return nil, err
		}
		if needsPodTemplates {
			for _, deploy := range d.deployments {
				d.podSpecs = append(d.podSpecs, podSpecRef{namespace: deploy.Namespace, labels: deploy.Spec.Template.Labels, spec: deploy.Spec.Template.Spec})
			}
		}
	}

	if needsPodTemplates {
		stss, err := d.resourceClient.ListStatefulSets(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, sts := range stss {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: sts.Namespace, labels: sts.Spec.Template.Labels, spec: sts.Spec.Template.Spec})
		}

		dss, err := d.resourceClient.ListDaemonSets(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, ds := range dss {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: ds.Namespace, labels: ds.Spec.Template.Labels, spec: ds.Spec.Template.Spec})
		}

		jobs, err := d.resourceClient.ListJobs(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: job.Namespace, labels: job.Spec.Template.Labels, spec: job.Spec.Template.Spec})
		}

		cronJobs, err := d.resourceClient.ListCronJobs(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, cronJob := range cronJobs {
			d.podSpecs = append(d.podSpecs, podSpecRef{namespace: cronJob.Namespace, labels: cronJob.Spec.JobTemplate.Spec.Template.Labels, spec: cronJob.Spec.JobTemplate.Spec.Template.Spec})
		}
	}

	if reapPersistentVolumes {
		var err error
		d.persistentVolumeClaims, err = d.resourceClient.ListPersistentVolumeClaims(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	var (
		sas             []*corev1.ServiceAccount
		ingresses       []*networkingv1.Ingress
		externalSecrets []*unstructured.Unstructured
	)
	if reapSecrets {
		var err error
		sas, err = d.resourceClient.ListServiceAccounts(ctx, namespace)
		if err != nil {
			return nil, err
		}
		ingresses, err = d.resourceClient.ListIngresses(ctx, namespace)
		if err != nil {
			return nil, err
		}
		externalSecrets, err = d.resourceClient.ListExternalSecrets(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}

	if reapConfigMaps {
		d.usedConfigMaps = d.detectUsedConfigMaps()
	}

	if reapSecrets {
		d.usedSecrets = d.detectUsedSecrets(sas, ingresses, externalSecrets)
	}

	if reapPersistentVolumeClaims {
		d.usedPersistentVolumeClaims = d.detectUsedPersistentVolumeClaims()
	}

	return d, nil
}

// DetermineDeletion determines whether a resource should be deleted.
func (d *determiner) DetermineDeletion(ctx context.Context, info *cliresource.Info) (bool, error) {
	switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
	case resource.KindPod:
		return d.determineDeletionPod(info)

	case resource.KindReplicaSet:
		return d.determineDeletionReplicaSet(info)

	case resource.KindConfigMap:
		return d.determineDeletionConfigMap(info)

	case resource.KindSecret:
		return d.determineDeletionSecret(info)

	case resource.KindPersistentVolume:
		return d.determineDeletionPersistentVolume(info)

	case resource.KindPersistentVolumeClaim:
		return d.determineDeletionPersistentVolumeClaim(info)

	case resource.KindJob:
		return d.determineDeletionJob(info)

	case resource.KindPodDisruptionBudget:
		return d.determineDeletionPodDisruptionBudget(info)

	case resource.KindHorizontalPodAutoscaler:
		return d.determineDeletionHorizontalPodAutoscaler(ctx, info)

	case resource.KindNetworkPolicy:
		return d.determineDeletionNetworkPolicy(info)

	default:
		return false, fmt.Errorf("unsupported kind: %s/%s", kind, info.Name)
	}
}

func (d *determiner) determineDeletionPod(info *cliresource.Info) (bool, error) {
	pod, err := resource.ObjectToPod(info.Object)
	if err != nil {
		return false, err
	}

	return pod.Status.Phase != corev1.PodRunning, nil
}

// determineDeletionReplicaSet deletes ReplicaSets that are scaled down to
// zero replicas, unless they are the live revision of their owning
// Deployment (the Deployment controller would just recreate them), or they
// are among the newest Options.KeepRevisions revisions of the Deployment so
// that `kubectl rollout undo` remains possible.
func (d *determiner) determineDeletionReplicaSet(info *cliresource.Info) (bool, error) {
	rs, err := resource.ObjectToReplicaSet(info.Object)
	if err != nil {
		return false, err
	}

	if rs.Spec.Replicas == nil || *rs.Spec.Replicas != 0 {
		return false, nil
	}

	controller := metav1.GetControllerOf(rs)
	if controller == nil {
		return true, nil
	}

	deployment := d.owningDeployment(rs, controller)
	if deployment == nil {
		return true, nil // owning Deployment is gone; this is an orphan
	}

	if d.keepRevisions > 0 && d.withinNewestRevisions(rs, controller, d.keepRevisions) {
		return false, nil // kept for rollback
	}

	if deployment.Annotations[deploymentRevisionAnnotation] == rs.Annotations[deploymentRevisionAnnotation] {
		// Current revision of a live Deployment; keep it.
		return false, nil
	}

	return true, nil
}

// owningDeployment returns the Deployment that owns the ReplicaSet via the
// given controller owner reference, or nil when it cannot be found.
func (d *determiner) owningDeployment(rs *appsv1.ReplicaSet, controller *metav1.OwnerReference) *appsv1.Deployment {
	for _, deploy := range d.deployments {
		// Prefer UID matching; fall back to name when either side has no UID
		// (e.g. objects constructed in tests).
		if deploy.UID != "" && controller.UID != "" {
			if deploy.UID != controller.UID {
				continue
			}
		} else if deploy.Name != controller.Name {
			continue
		}
		return deploy
	}
	return nil
}

// withinNewestRevisions reports whether the ReplicaSet is among the newest
// `keep` revisions of its owning Deployment: fewer than `keep` sibling
// ReplicaSets of the same Deployment carry a strictly higher revision
// number. ReplicaSets whose revision cannot be parsed are conservatively
// treated as protected.
func (d *determiner) withinNewestRevisions(rs *appsv1.ReplicaSet, controller *metav1.OwnerReference, keep int) bool {
	revision, ok := parseRevision(rs.Annotations[deploymentRevisionAnnotation])
	if !ok {
		return true
	}

	newer := 0
	for _, sibling := range d.replicaSets {
		if sibling == rs {
			continue
		}
		if !sameOwner(sibling, controller) {
			continue
		}
		if siblingRevision, ok := parseRevision(sibling.Annotations[deploymentRevisionAnnotation]); ok && siblingRevision > revision {
			newer++
		}
	}

	return newer < keep
}

// sameOwner reports whether the ReplicaSet is controlled by the given owner
// reference (by UID, falling back to name).
func sameOwner(rs *appsv1.ReplicaSet, controller *metav1.OwnerReference) bool {
	rsController := metav1.GetControllerOf(rs)
	if rsController == nil {
		return false
	}
	if rsController.UID != "" && controller.UID != "" {
		return rsController.UID == controller.UID
	}
	return rsController.Name == controller.Name
}

func parseRevision(revision string) (int, bool) {
	n, err := strconv.Atoi(revision)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (d *determiner) determineDeletionConfigMap(info *cliresource.Info) (bool, error) {
	if info.Name == rootCAConfigMapName {
		return false, nil // consumed implicitly by every Pod; never reap
	}
	_, ok := d.usedConfigMaps[namespacedName(info.Namespace, info.Name)]
	return !ok, nil
}

func (d *determiner) determineDeletionSecret(info *cliresource.Info) (bool, error) {
	secret, err := resource.ObjectToSecret(info.Object)
	if err != nil {
		return false, err
	}
	if secret.Type == corev1.SecretTypeServiceAccountToken {
		// Tokens referenced by a ServiceAccount are already protected by
		// reference scanning; manually created tokens used by external
		// integrations are invisible to it, so never reap token Secrets.
		return false, nil
	}
	_, ok := d.usedSecrets[namespacedName(info.Namespace, info.Name)]
	return !ok, nil
}

func (d *determiner) determineDeletionPersistentVolume(info *cliresource.Info) (bool, error) {
	volume, err := resource.ObjectToPersistentVolume(info.Object)
	if err != nil {
		return false, err
	}

	if volume.Status.Phase == corev1.VolumeBound {
		return false, nil // a Bound volume is serving a claim; never reap it
	}

	for _, claim := range d.persistentVolumeClaims {
		if ok := checkVolumeSatisfyClaimFunc(volume, claim); ok {
			return false, nil
		}
	}
	return true, nil // should delete PV if it doesn't satisfy any PVCs
}

func (d *determiner) determineDeletionPersistentVolumeClaim(info *cliresource.Info) (bool, error) {
	_, ok := d.usedPersistentVolumeClaims[namespacedName(info.Namespace, info.Name)]
	return !ok, nil
}

func (d *determiner) determineDeletionJob(info *cliresource.Info) (bool, error) {
	job, err := resource.ObjectToJob(info.Object)
	if err != nil {
		return false, err
	}

	return job.Status.CompletionTime != nil, nil
}

func (d *determiner) determineDeletionPodDisruptionBudget(info *cliresource.Info) (bool, error) {
	pdb, err := resource.ObjectToPodDisruptionBudget(info.Object)
	if err != nil {
		return false, err
	}

	used, err := d.determineUsedPodDisruptionBudget(pdb)
	if err != nil {
		return false, err
	}
	return !used, nil
}

func (d *determiner) determineDeletionHorizontalPodAutoscaler(ctx context.Context, info *cliresource.Info) (bool, error) {
	hpa, err := resource.ObjectToHorizontalPodAutoscaler(info.Object)
	if err != nil {
		return false, err
	}

	ref := hpa.Spec.ScaleTargetRef
	u, err := d.resourceClient.GetUnstructured(ctx, ref.APIVersion, ref.Kind, ref.Name, info.Namespace)
	if err != nil {
		return false, err
	}
	return u == nil, nil // should delete HPA if ScaleTargetRef's target object is not found
}

func (d *determiner) determineDeletionNetworkPolicy(info *cliresource.Info) (bool, error) {
	np, err := resource.ObjectToNetworkPolicy(info.Object)
	if err != nil {
		return false, err
	}

	// An empty podSelector selects all Pods in the namespace.
	selector := labels.Everything()
	if !isEmptyLabelSelector(np.Spec.PodSelector) {
		selector, err = metav1.LabelSelectorAsSelector(&np.Spec.PodSelector)
		if err != nil {
			return false, fmt.Errorf("invalid pod selector (%s): %w", np.Name, err)
		}
	}

	// Keep the policy when any live Pod, or the Pod template of any workload
	// that could create such Pods later (e.g. a scaled-to-zero Deployment),
	// is selected.
	for _, pod := range d.pods {
		if pod.Namespace == np.Namespace && selector.Matches(labels.Set(pod.Labels)) {
			return false, nil
		}
	}
	for _, ref := range d.podSpecs {
		if ref.namespace == np.Namespace && selector.Matches(labels.Set(ref.labels)) {
			return false, nil
		}
	}

	return true, nil // no Pods are selected by this NetworkPolicy
}

func (d *determiner) detectUsedConfigMaps() map[string]struct{} {
	usedConfigMaps := make(map[string]struct{})

	for _, ref := range d.podSpecs {
		eachContainer(ref.spec, func(container corev1.Container) {
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					usedConfigMaps[namespacedName(ref.namespace, envFrom.ConfigMapRef.Name)] = struct{}{}
				}
			}

			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					usedConfigMaps[namespacedName(ref.namespace, env.ValueFrom.ConfigMapKeyRef.Name)] = struct{}{}
				}
			}
		})

		for _, volume := range ref.spec.Volumes {
			if volume.ConfigMap != nil {
				usedConfigMaps[namespacedName(ref.namespace, volume.ConfigMap.Name)] = struct{}{}
			}

			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						usedConfigMaps[namespacedName(ref.namespace, source.ConfigMap.Name)] = struct{}{}
					}
				}
			}
		}
	}

	return usedConfigMaps
}

func (d *determiner) detectUsedSecrets(sas []*corev1.ServiceAccount, ingresses []*networkingv1.Ingress, externalSecrets []*unstructured.Unstructured) map[string]struct{} {
	usedSecrets := make(map[string]struct{})

	// Add Secrets used by Pods and workload Pod templates
	// (ReplicaSets, Deployments, StatefulSets, DaemonSets, Jobs, CronJobs)
	for _, ref := range d.podSpecs {
		for _, imagePullSecret := range ref.spec.ImagePullSecrets {
			usedSecrets[namespacedName(ref.namespace, imagePullSecret.Name)] = struct{}{}
		}

		eachContainer(ref.spec, func(container corev1.Container) {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil {
					usedSecrets[namespacedName(ref.namespace, envFrom.SecretRef.Name)] = struct{}{}
				}
			}

			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					usedSecrets[namespacedName(ref.namespace, env.ValueFrom.SecretKeyRef.Name)] = struct{}{}
				}
			}
		})

		for _, volume := range ref.spec.Volumes {
			if volume.Secret != nil {
				usedSecrets[namespacedName(ref.namespace, volume.Secret.SecretName)] = struct{}{}
			}

			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil {
						usedSecrets[namespacedName(ref.namespace, source.Secret.Name)] = struct{}{}
					}
				}
			}
		}
	}

	// Add Secrets used by ServiceAccounts: token references and image pull
	// secrets (the latter are attached to every Pod that uses the account).
	for _, sa := range sas {
		for _, secret := range sa.Secrets {
			usedSecrets[namespacedName(sa.Namespace, secret.Name)] = struct{}{}
		}
		for _, imagePullSecret := range sa.ImagePullSecrets {
			usedSecrets[namespacedName(sa.Namespace, imagePullSecret.Name)] = struct{}{}
		}
	}

	// Add Secrets referenced as Ingress TLS certificates and via nginx
	// ingress auth annotations
	for _, ingress := range ingresses {
		for _, tls := range ingress.Spec.TLS {
			if tls.SecretName != "" {
				usedSecrets[namespacedName(ingress.Namespace, tls.SecretName)] = struct{}{}
			}
		}
		for _, annotation := range ingressSecretAnnotations {
			if name := ingress.Annotations[annotation]; name != "" {
				usedSecrets[namespacedName(ingress.Namespace, name)] = struct{}{}
			}
		}
	}

	// Add Secrets bound by ExternalSecret custom resources
	// (external-secrets.io and kubernetes-client.io operators)
	for _, es := range externalSecrets {
		usedSecrets[namespacedName(es.GetNamespace(), externalSecretTargetSecretName(es))] = struct{}{}
	}

	return usedSecrets
}

func (d *determiner) detectUsedPersistentVolumeClaims() map[string]struct{} {
	usedPersistentVolumeClaims := make(map[string]struct{})

	for _, ref := range d.podSpecs {
		for _, volume := range ref.spec.Volumes {
			if volume.PersistentVolumeClaim == nil {
				continue
			}
			usedPersistentVolumeClaims[namespacedName(ref.namespace, volume.PersistentVolumeClaim.ClaimName)] = struct{}{}
		}
	}

	return usedPersistentVolumeClaims
}

func (d *determiner) determineUsedPodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) (bool, error) {
	// In policy/v1 an empty selector matches all Pods in the namespace.
	selector := labels.Everything()
	if pdb.Spec.Selector != nil && !isEmptyLabelSelector(*pdb.Spec.Selector) {
		var err error
		selector, err = metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil {
			return false, fmt.Errorf("invalid label selector (%s): %w", pdb.Name, err)
		}
	}

	// The budget is in use when any live Pod, or the Pod template of any
	// workload that could create such Pods later (e.g. a scaled-to-zero
	// Deployment), is selected.
	for _, pod := range d.pods {
		if pod.Namespace == pdb.Namespace && selector.Matches(labels.Set(pod.Labels)) {
			return true, nil
		}
	}
	for _, ref := range d.podSpecs {
		if ref.namespace == pdb.Namespace && selector.Matches(labels.Set(ref.labels)) {
			return true, nil
		}
	}

	return false, nil
}

// externalSecretTargetSecretName returns the name of the Secret an
// ExternalSecret manages. The external-secrets.io operator allows an explicit
// spec.target.name; otherwise both major operators bind to a Secret named
// after the ExternalSecret itself.
func externalSecretTargetSecretName(es *unstructured.Unstructured) string {
	if name, found, _ := unstructured.NestedString(es.Object, "spec", "target", "name"); found && name != "" {
		return name
	}
	return es.GetName()
}

func namespacedName(namespace, name string) string {
	return namespace + "/" + name
}

// eachContainer invokes fn for every regular and init container of a PodSpec.
func eachContainer(spec corev1.PodSpec, fn func(container corev1.Container)) {
	for _, container := range spec.Containers {
		fn(container)
	}
	for _, container := range spec.InitContainers {
		fn(container)
	}
}

func isEmptyLabelSelector(selector metav1.LabelSelector) bool {
	return len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0
}
