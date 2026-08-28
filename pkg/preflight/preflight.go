// Package preflight verifies that the user's RBAC permissions cover
// everything a harvest run needs before any work (and especially any
// deletion) starts. It uses SelfSubjectAccessReview — the same mechanism
// behind `kubectl auth can-i`, which every authenticated user is allowed
// to call — so the check itself needs zero extra permissions.
package preflight

import (
	"context"
	"fmt"
	"sort"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/vivektech/kubectl-harvest/pkg/resource"
)

// Permission is a verb/resource pair that gets checked against the API
// server's authorization layer.
type Permission struct {
	Verb          string
	Group         string // "" means the core API group
	Resource      string // plural, lowercase, as used in RBAC rules
	ClusterScoped bool
}

func (p Permission) String() string {
	name := p.Resource
	if p.Group != "" {
		name += "." + p.Group
	}
	return p.Verb + " " + name
}

func (p Permission) key() string {
	return p.Verb + "|" + p.Group + "|" + p.Resource
}

// Options configures a pre-flight run.
type Options struct {
	// Kinds are the resource kinds being reaped (resource.KindSecret, ...).
	Kinds []string
	// Namespace is the target namespace; "" means all namespaces
	// (cluster-wide authorization is checked in that case).
	Namespace string
	// IncludeDelete adds delete permission on the reaped kinds themselves.
	// Dry-runs only need read access, so this is false for them.
	IncludeDelete bool
	// GroupServed optionally reports whether an API group is served by the
	// cluster. It gates checks for optional CRDs (ExternalSecrets) so that
	// clusters without the operator installed do not produce false
	// "missing permission" reports. May be nil.
	GroupServed func(group string) bool
}

// argAliases maps the resource-type tokens users may pass on the command
// line (including short names) to kinds.
var argAliases = map[string]string{
	"pod": resource.KindPod, "pods": resource.KindPod, "po": resource.KindPod,
	"replicaset": resource.KindReplicaSet, "replicasets": resource.KindReplicaSet, "rs": resource.KindReplicaSet,
	"configmap": resource.KindConfigMap, "configmaps": resource.KindConfigMap, "cm": resource.KindConfigMap,
	"secret": resource.KindSecret, "secrets": resource.KindSecret,
	"persistentvolume": resource.KindPersistentVolume, "persistentvolumes": resource.KindPersistentVolume, "pv": resource.KindPersistentVolume,
	"persistentvolumeclaim": resource.KindPersistentVolumeClaim, "persistentvolumeclaims": resource.KindPersistentVolumeClaim, "pvc": resource.KindPersistentVolumeClaim,
	"job": resource.KindJob, "jobs": resource.KindJob,
	"poddisruptionbudget": resource.KindPodDisruptionBudget, "poddisruptionbudgets": resource.KindPodDisruptionBudget, "pdb": resource.KindPodDisruptionBudget,
	"horizontalpodautoscaler": resource.KindHorizontalPodAutoscaler, "horizontalpodautoscalers": resource.KindHorizontalPodAutoscaler, "hpa": resource.KindHorizontalPodAutoscaler,
	"networkpolicy": resource.KindNetworkPolicy, "networkpolicies": resource.KindNetworkPolicy, "netpol": resource.KindNetworkPolicy,
}

// KindForArg resolves a command-line resource-type token to a kind. Group
// qualifiers (e.g. "poddisruptionbudgets.policy") are tolerated.
func KindForArg(arg string) (string, bool) {
	a := strings.ToLower(strings.TrimSpace(arg))
	if i := strings.Index(a, "."); i > 0 {
		a = a[:i]
	}
	kind, ok := argAliases[a]
	return kind, ok
}

// KindsForResourceTypes resolves the comma-separated resource-type argument
// to kinds. Tokens that cannot be resolved are skipped; the run surfaces
// them later through the normal API machinery.
func KindsForResourceTypes(resourceTypes string) []string {
	var kinds []string
	for _, arg := range strings.Split(resourceTypes, ",") {
		if kind, ok := KindForArg(arg); ok {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

// targetResource describes the reaped resource itself.
type targetResource struct {
	group         string
	resource      string
	clusterScoped bool
}

var targetResources = map[string]targetResource{
	resource.KindPod:                     {resource: "pods"},
	resource.KindReplicaSet:              {group: "apps", resource: "replicasets"},
	resource.KindConfigMap:               {resource: "configmaps"},
	resource.KindSecret:                  {resource: "secrets"},
	resource.KindPersistentVolume:        {resource: "persistentvolumes", clusterScoped: true},
	resource.KindPersistentVolumeClaim:   {resource: "persistentvolumeclaims"},
	resource.KindJob:                     {group: "batch", resource: "jobs"},
	resource.KindPodDisruptionBudget:     {group: "policy", resource: "poddisruptionbudgets"},
	resource.KindHorizontalPodAutoscaler: {group: "autoscaling", resource: "horizontalpodautoscalers"},
	resource.KindNetworkPolicy:           {group: "networking.k8s.io", resource: "networkpolicies"},
}

// workloadLists are the read permissions needed to scan every workload
// Pod template for references.
var workloadLists = []Permission{
	{Verb: "list", Resource: "pods"},
	{Verb: "list", Group: "apps", Resource: "replicasets"},
	{Verb: "list", Group: "apps", Resource: "deployments"},
	{Verb: "list", Group: "apps", Resource: "statefulsets"},
	{Verb: "list", Group: "apps", Resource: "daemonsets"},
	{Verb: "list", Group: "batch", Resource: "jobs"},
	{Verb: "list", Group: "batch", Resource: "cronjobs"},
}

// referenceSources maps each reaped kind to the reference sources the
// determiner reads before deciding anything. It must stay in sync with
// determiner.collectReferences.
var referenceSources = map[string][]Permission{
	resource.KindPod:                     nil,
	resource.KindJob:                     nil,
	resource.KindHorizontalPodAutoscaler: nil,
	resource.KindReplicaSet: {
		{Verb: "list", Group: "apps", Resource: "replicasets"},
		{Verb: "list", Group: "apps", Resource: "deployments"},
	},
	resource.KindPodDisruptionBudget: {
		{Verb: "list", Resource: "pods"},
	},
	resource.KindNetworkPolicy: {
		{Verb: "list", Resource: "pods"},
	},
	resource.KindConfigMap:             workloadLists,
	resource.KindPersistentVolumeClaim: workloadLists,
	resource.KindPersistentVolume: {
		{Verb: "list", Resource: "persistentvolumeclaims"},
	},
	resource.KindSecret: append(append([]Permission{}, workloadLists...),
		Permission{Verb: "list", Resource: "serviceaccounts"},
		Permission{Verb: "list", Group: "networking.k8s.io", Resource: "ingresses"},
	),
}

// externalSecretGroups are the API groups of the supported ExternalSecret
// operators. Checks against them are only included when the group is
// actually served by the cluster.
var externalSecretGroups = []string{"external-secrets.io", "kubernetes-client.io"}

// PermissionsForKinds returns the deduplicated set of permissions a run
// reaping the given kinds needs.
func PermissionsForKinds(kinds []string, includeDelete bool, groupServed func(string) bool) []Permission {
	seen := map[string]Permission{}
	add := func(p Permission) {
		seen[p.key()] = p
	}

	for _, kind := range kinds {
		target, ok := targetResources[kind]
		if !ok {
			continue
		}
		add(Permission{Verb: "list", Group: target.group, Resource: target.resource, ClusterScoped: target.clusterScoped})
		if includeDelete {
			add(Permission{Verb: "delete", Group: target.group, Resource: target.resource, ClusterScoped: target.clusterScoped})
		}
		for _, p := range referenceSources[kind] {
			add(p)
		}
		if kind == resource.KindSecret && groupServed != nil {
			for _, group := range externalSecretGroups {
				if groupServed(group) {
					add(Permission{Verb: "list", Group: group, Resource: "externalsecrets"})
				}
			}
		}
	}

	perms := make([]Permission, 0, len(seen))
	for _, p := range seen {
		perms = append(perms, p)
	}
	sort.Slice(perms, func(i, j int) bool {
		if perms[i].Group != perms[j].Group {
			return perms[i].Group < perms[j].Group
		}
		if perms[i].Resource != perms[j].Resource {
			return perms[i].Resource < perms[j].Resource
		}
		return perms[i].Verb < perms[j].Verb
	})
	return perms
}

// Run checks every permission via SelfSubjectAccessReview and returns the
// denied ones. An error means the check itself could not run (callers
// should warn and proceed rather than block).
func Run(ctx context.Context, clientset kubernetes.Interface, opts Options) ([]Permission, error) {
	var missing []Permission
	for _, p := range PermissionsForKinds(opts.Kinds, opts.IncludeDelete, opts.GroupServed) {
		namespace := ""
		if !p.ClusterScoped {
			namespace = opts.Namespace
		}
		review := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      p.Verb,
					Group:     p.Group,
					Resource:  p.Resource,
				},
			},
		}
		resp, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		if !resp.Status.Allowed {
			missing = append(missing, p)
		}
	}
	return missing, nil
}

// MissingPermissionsError reports the permissions a run needs that the
// current user lacks. Its message is rendered by FormatMissing.
type MissingPermissionsError struct {
	Missing       []Permission
	Namespace     string
	AllNamespaces bool
}

// Error implements the error interface.
func (e *MissingPermissionsError) Error() string {
	return FormatMissing(e.Missing, e.Namespace, e.AllNamespaces)
}

// FormatMissing renders the missing permissions as a user-facing error
// message. allNamespaces switches the scope label to cluster-wide.
func FormatMissing(missing []Permission, namespace string, allNamespaces bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RBAC pre-flight check failed: this run needs the following permissions that your user is missing:\n")
	for _, p := range missing {
		scope := fmt.Sprintf("namespace %q", namespace)
		if allNamespaces || p.ClusterScoped {
			scope = "cluster-wide"
		}
		fmt.Fprintf(&b, "  - %s (%s)\n", p.String(), scope)
	}
	b.WriteString("Ask your cluster administrator to grant them, or inspect your current permissions with: kubectl auth can-i --list")
	return b.String()
}
