package preflight

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/vivektech/kubectl-harvest/pkg/resource"
)

func permissionSet(perms ...Permission) map[string]bool {
	set := map[string]bool{}
	for _, p := range perms {
		set[p.key()] = true
	}
	return set
}

func TestPermissionsForKinds(t *testing.T) {
	allGroupsServed := func(string) bool { return true }
	noGroupsServed := func(string) bool { return false }

	tests := []struct {
		name          string
		kinds         []string
		includeDelete bool
		groupServed   func(string) bool
		want          map[string]bool
	}{
		{
			name:          "Pod needs only pod list and delete",
			kinds:         []string{resource.KindPod},
			includeDelete: true,
			groupServed:   allGroupsServed,
			want:          permissionSet(Permission{Verb: "list", Resource: "pods"}, Permission{Verb: "delete", Resource: "pods"}),
		},
		{
			name:          "Pod dry-run needs no delete",
			kinds:         []string{resource.KindPod},
			includeDelete: false,
			groupServed:   allGroupsServed,
			want:          permissionSet(Permission{Verb: "list", Resource: "pods"}),
		},
		{
			name:          "ConfigMap needs workload lists but no secret sources",
			kinds:         []string{resource.KindConfigMap},
			includeDelete: true,
			groupServed:   allGroupsServed,
			want: permissionSet(
				Permission{Verb: "list", Resource: "configmaps"},
				Permission{Verb: "delete", Resource: "configmaps"},
				Permission{Verb: "list", Resource: "pods"},
				Permission{Verb: "list", Group: "apps", Resource: "replicasets"},
				Permission{Verb: "list", Group: "apps", Resource: "deployments"},
				Permission{Verb: "list", Group: "apps", Resource: "statefulsets"},
				Permission{Verb: "list", Group: "apps", Resource: "daemonsets"},
				Permission{Verb: "list", Group: "batch", Resource: "jobs"},
				Permission{Verb: "list", Group: "batch", Resource: "cronjobs"},
			),
		},
		{
			name:          "Secret needs the full reference surface",
			kinds:         []string{resource.KindSecret},
			includeDelete: true,
			groupServed:   allGroupsServed,
			want: permissionSet(
				Permission{Verb: "list", Resource: "secrets"},
				Permission{Verb: "delete", Resource: "secrets"},
				Permission{Verb: "list", Resource: "pods"},
				Permission{Verb: "list", Group: "apps", Resource: "replicasets"},
				Permission{Verb: "list", Group: "apps", Resource: "deployments"},
				Permission{Verb: "list", Group: "apps", Resource: "statefulsets"},
				Permission{Verb: "list", Group: "apps", Resource: "daemonsets"},
				Permission{Verb: "list", Group: "batch", Resource: "jobs"},
				Permission{Verb: "list", Group: "batch", Resource: "cronjobs"},
				Permission{Verb: "list", Resource: "serviceaccounts"},
				Permission{Verb: "list", Group: "networking.k8s.io", Resource: "ingresses"},
				Permission{Verb: "list", Group: "external-secrets.io", Resource: "externalsecrets"},
				Permission{Verb: "list", Group: "kubernetes-client.io", Resource: "externalsecrets"},
			),
		},
		{
			name:          "Secret on a cluster without ExternalSecret operators skips their checks",
			kinds:         []string{resource.KindSecret},
			includeDelete: false,
			groupServed:   noGroupsServed,
			want: permissionSet(
				Permission{Verb: "list", Resource: "secrets"},
				Permission{Verb: "list", Resource: "pods"},
				Permission{Verb: "list", Group: "apps", Resource: "replicasets"},
				Permission{Verb: "list", Group: "apps", Resource: "deployments"},
				Permission{Verb: "list", Group: "apps", Resource: "statefulsets"},
				Permission{Verb: "list", Group: "apps", Resource: "daemonsets"},
				Permission{Verb: "list", Group: "batch", Resource: "jobs"},
				Permission{Verb: "list", Group: "batch", Resource: "cronjobs"},
				Permission{Verb: "list", Resource: "serviceaccounts"},
				Permission{Verb: "list", Group: "networking.k8s.io", Resource: "ingresses"},
			),
		},
		{
			name:          "PersistentVolume is cluster-scoped and needs PVC list",
			kinds:         []string{resource.KindPersistentVolume},
			includeDelete: false,
			groupServed:   allGroupsServed,
			want: permissionSet(
				Permission{Verb: "list", Resource: "persistentvolumes", ClusterScoped: true},
				Permission{Verb: "list", Resource: "persistentvolumeclaims"},
			),
		},
		{
			name:          "Combined kinds are deduplicated",
			kinds:         []string{resource.KindConfigMap, resource.KindPersistentVolumeClaim},
			includeDelete: true,
			groupServed:   allGroupsServed,
			want: permissionSet(
				Permission{Verb: "list", Resource: "configmaps"},
				Permission{Verb: "delete", Resource: "configmaps"},
				Permission{Verb: "list", Resource: "persistentvolumeclaims"},
				Permission{Verb: "delete", Resource: "persistentvolumeclaims"},
				Permission{Verb: "list", Resource: "pods"},
				Permission{Verb: "list", Group: "apps", Resource: "replicasets"},
				Permission{Verb: "list", Group: "apps", Resource: "deployments"},
				Permission{Verb: "list", Group: "apps", Resource: "statefulsets"},
				Permission{Verb: "list", Group: "apps", Resource: "daemonsets"},
				Permission{Verb: "list", Group: "batch", Resource: "jobs"},
				Permission{Verb: "list", Group: "batch", Resource: "cronjobs"},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			perms := PermissionsForKinds(tt.kinds, tt.includeDelete, tt.groupServed)
			got := permissionSet(perms...)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestKindsForResourceTypes(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		want         []string
	}{
		{
			name:         "short names and mixed case resolve",
			resourceType: "cm,Secret",
			want:         []string{resource.KindConfigMap, resource.KindSecret},
		},
		{
			name:         "group-qualified names resolve",
			resourceType: "poddisruptionbudgets.policy",
			want:         []string{resource.KindPodDisruptionBudget},
		},
		{
			name:         "unknown tokens are skipped",
			resourceType: "cm,frobnicators,po",
			want:         []string{resource.KindConfigMap, resource.KindPod},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := KindsForResourceTypes(tt.resourceType)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

// newFakeAuthClient builds a fake clientset whose SelfSubjectAccessReview
// answers deny exactly the resources in denied.
func newFakeAuthClient(denied map[string]bool) *fakeclientset.Clientset {
	clientset := fakeclientset.NewSimpleClientset()
	clientset.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
		attrs := review.Spec.ResourceAttributes
		key := attrs.Verb + "|" + attrs.Group + "|" + attrs.Resource
		review.Status.Allowed = !denied[key]
		return true, review, nil
	})
	return clientset
}

func TestRun(t *testing.T) {
	noGroupsServed := func(string) bool { return false }
	allGroupsServed := func(string) bool { return true }

	tests := []struct {
		name        string
		groupServed func(string) bool
		denied      map[string]bool
		want        []string
	}{
		{
			name:        "fully permitted user gets no missing permissions",
			groupServed: noGroupsServed,
			denied:      map[string]bool{},
			want:        nil,
		},
		{
			name:        "denied verb and group are reported exactly",
			groupServed: allGroupsServed,
			denied: map[string]bool{
				"list|networking.k8s.io|ingresses":         true,
				"list|external-secrets.io|externalsecrets": true,
			},
			want: []string{
				"list externalsecrets.external-secrets.io",
				"list ingresses.networking.k8s.io",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientset := newFakeAuthClient(tt.denied)
			missing, err := Run(context.Background(), clientset, Options{
				Kinds:         []string{resource.KindSecret},
				Namespace:     "fake-ns",
				IncludeDelete: true,
				GroupServed:   tt.groupServed,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			var got []string
			for _, p := range missing {
				got = append(got, p.String())
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatMissing(t *testing.T) {
	message := FormatMissing([]Permission{
		{Verb: "list", Group: "networking.k8s.io", Resource: "ingresses"},
		{Verb: "list", Resource: "persistentvolumes", ClusterScoped: true},
	}, "rpay-dev-rpay-services", false)

	for _, want := range []string{
		"RBAC pre-flight check failed",
		`- list ingresses.networking.k8s.io (namespace "rpay-dev-rpay-services")`,
		"- list persistentvolumes (cluster-wide)",
		"kubectl auth can-i --list",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("FormatMissing() output missing %q:\n%s", want, message)
		}
	}
}
