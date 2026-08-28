package determiner

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	cliresource "k8s.io/cli-runtime/pkg/resource"

	"github.com/vivektech/kubectl-harvest/pkg/resource"
)

func Test_determiner_DetermineDeletion(t *testing.T) {
	const (
		fakeNamespace             = "fake-ns"
		fakePod                   = "fake-pod"
		fakeConfigMap             = "fake-cm"
		fakeSecret                = "fake-secret"
		fakePersistentVolumeClaim = "fake-pvc"
		fakeJob                   = "fake-job"
		fakePodDisruptionBudget   = "fake-pdb"
		fakeLabelKey1             = "fake-label1-key"
		fakeLabelValue1           = "fake-label1-value"
	)

	fakeTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	type fields struct {
		usedConfigMaps        map[string]struct{}
		usedSecrets           map[string]struct{}
		usedPersistentVolumes map[string]struct{}
		pods                  []*corev1.Pod
		podSpecs              []podSpecRef
	}
	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Pod should be deleted when it is not running",
			args: args{
				info: &cliresource.Info{
					Name:      fakePod,
					Namespace: fakeNamespace,
					Object: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPod,
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodFailed,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Pod should not be deleted when it is running",
			args: args{
				info: &cliresource.Info{
					Name:      fakePod,
					Namespace: fakeNamespace,
					Object: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPod,
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "ConfigMap should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name:      fakeConfigMap,
					Namespace: fakeNamespace,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ConfigMap should not be deleted when it is used",
			fields: fields{
				usedConfigMaps: map[string]struct{}{
					fakeNamespace + "/" + fakeConfigMap: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakeConfigMap,
					Namespace: fakeNamespace,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "ConfigMap in another namespace should still be deleted",
			fields: fields{
				usedConfigMaps: map[string]struct{}{
					"other-ns/" + fakeConfigMap: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakeConfigMap,
					Namespace: fakeNamespace,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "kube-root-ca.crt ConfigMap should never be deleted",
			args: args{
				info: &cliresource.Info{
					Name:      "kube-root-ca.crt",
					Namespace: fakeNamespace,
					Object: &corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindConfigMap,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Secret should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name:      fakeSecret,
					Namespace: fakeNamespace,
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindSecret,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Secret should not be deleted when it is used",
			fields: fields{
				usedSecrets: map[string]struct{}{
					fakeNamespace + "/" + fakeSecret: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakeSecret,
					Namespace: fakeNamespace,
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindSecret,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "ServiceAccount token Secret should never be deleted",
			args: args{
				info: &cliresource.Info{
					Name:      fakeSecret,
					Namespace: fakeNamespace,
					Object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindSecret,
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PersistentVolumeClaim should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name:      fakePersistentVolumeClaim,
					Namespace: fakeNamespace,
					Object: &corev1.PersistentVolumeClaim{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolumeClaim,
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "PersistentVolumeClaim should not be deleted when it is used",
			fields: fields{
				usedPersistentVolumes: map[string]struct{}{
					fakeNamespace + "/" + fakePersistentVolumeClaim: {},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakePersistentVolumeClaim,
					Namespace: fakeNamespace,
					Object: &corev1.PersistentVolumeClaim{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolumeClaim,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Job should be deleted when it is completed",
			args: args{
				info: &cliresource.Info{
					Name:      fakeJob,
					Namespace: fakeNamespace,
					Object: &batchv1.Job{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindJob,
						},
						Status: batchv1.JobStatus{
							CompletionTime: &metav1.Time{
								Time: fakeTime,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Job should not be deleted when it is not completed",
			args: args{
				info: &cliresource.Info{
					Name:      fakeJob,
					Namespace: fakeNamespace,
					Object: &batchv1.Job{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindJob,
						},
						Status: batchv1.JobStatus{},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name:      fakePodDisruptionBudget,
					Namespace: fakeNamespace,
					Object: &policyv1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						Spec: policyv1.PodDisruptionBudgetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget should not be deleted when it is used",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakePodDisruptionBudget,
					Namespace: fakeNamespace,
					Object: &policyv1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: policyv1.PodDisruptionBudgetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget with empty selector should not be deleted when Pods exist in the namespace",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakePodDisruptionBudget,
					Namespace: fakeNamespace,
					Object: &policyv1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: policyv1.PodDisruptionBudgetSpec{},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget with no live Pods but a matching scaled-to-zero workload template should not be deleted",
			fields: fields{
				podSpecs: []podSpecRef{
					{
						namespace: fakeNamespace,
						labels: map[string]string{
							fakeLabelKey1: fakeLabelValue1,
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      fakePodDisruptionBudget,
					Namespace: fakeNamespace,
					Object: &policyv1.PodDisruptionBudget{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPodDisruptionBudget,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: policyv1.PodDisruptionBudgetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "NetworkPolicy should be deleted when no Pods are selected",
			args: args{
				info: &cliresource.Info{
					Name:      "fake-netpol",
					Namespace: fakeNamespace,
					Object: &networkingv1.NetworkPolicy{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindNetworkPolicy,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: networkingv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "NetworkPolicy should not be deleted when Pods are selected",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      "fake-netpol",
					Namespace: fakeNamespace,
					Object: &networkingv1.NetworkPolicy{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindNetworkPolicy,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: networkingv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "NetworkPolicy should not be deleted when a scaled-to-zero workload template matches",
			fields: fields{
				podSpecs: []podSpecRef{
					{
						namespace: fakeNamespace,
						labels: map[string]string{
							fakeLabelKey1: fakeLabelValue1,
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name:      "fake-netpol",
					Namespace: fakeNamespace,
					Object: &networkingv1.NetworkPolicy{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindNetworkPolicy,
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: networkingv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									fakeLabelKey1: fakeLabelValue1,
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				usedConfigMaps:             tt.fields.usedConfigMaps,
				usedSecrets:                tt.fields.usedSecrets,
				usedPersistentVolumeClaims: tt.fields.usedPersistentVolumes,
				pods:                       tt.fields.pods,
				podSpecs:                   tt.fields.podSpecs,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_ReplicaSet_KeepRevisions(t *testing.T) {
	const (
		fakeNamespace     = "fake-ns"
		fakeDeployment    = "fake-deployment"
		fakeDeploymentUID = "fake-deployment-uid"
	)

	zero := int32(0)

	// Deployment whose live revision is 7.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeDeployment,
			Namespace: fakeNamespace,
			UID:       fakeDeploymentUID,
			Annotations: map[string]string{
				deploymentRevisionAnnotation: "7",
			},
		},
	}

	// newRS returns a ReplicaSet at the given revision, scaled to zero and
	// owned by the Deployment.
	newRS := func(revision int) *appsv1.ReplicaSet {
		controller := true
		return &appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{Kind: resource.KindReplicaSet},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", fakeDeployment, revision),
				Namespace: fakeNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       fakeDeployment,
						UID:        fakeDeploymentUID,
						Controller: &controller,
					},
				},
				Annotations: map[string]string{
					deploymentRevisionAnnotation: strconv.Itoa(revision),
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &zero,
			},
		}
	}

	// ReplicaSets at revisions 1..7, all scaled to zero, all owned by the
	// Deployment.
	allReplicaSets := func() []*appsv1.ReplicaSet {
		var rss []*appsv1.ReplicaSet
		for revision := 1; revision <= 7; revision++ {
			rss = append(rss, newRS(revision))
		}
		return rss
	}()

	type args struct {
		keepRevisions int
		target        *appsv1.ReplicaSet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ReplicaSet within the newest 5 revisions should be kept",
			args: args{
				keepRevisions: 5,
				target:        newRS(3), // revisions 4..7 are newer: 4 siblings < 5
			},
			want: false,
		},
		{
			name: "ReplicaSet older than the newest 5 revisions should be deleted",
			args: args{
				keepRevisions: 5,
				target:        newRS(2), // revisions 3..7 are newer: 5 siblings >= 5
			},
			want: true,
		},
		{
			name: "live revision should be kept with keep-revisions 0",
			args: args{
				keepRevisions: 0,
				target:        newRS(7),
			},
			want: false,
		},
		{
			name: "old revision should be deleted with keep-revisions 0",
			args: args{
				keepRevisions: 0,
				target:        newRS(6),
			},
			want: true,
		},
		{
			name: "keep-revisions larger than the revision count keeps everything",
			args: args{
				keepRevisions: 10,
				target:        newRS(1),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				replicaSets:   allReplicaSets,
				deployments:   []*appsv1.Deployment{deployment},
				keepRevisions: tt.args.keepRevisions,
			}

			got, err := d.DetermineDeletion(context.Background(), &cliresource.Info{
				Name:      tt.args.target.Name,
				Namespace: fakeNamespace,
				Object:    tt.args.target,
			})
			if err != nil {
				t.Errorf("determiner.DetermineDeletion() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_ReplicaSet(t *testing.T) {
	const (
		fakeNamespace      = "fake-ns"
		fakeDeployment     = "fake-deployment"
		fakeReplicaSetOld  = "fake-replicaset-old"
		fakeReplicaSetLive = "fake-replicaset-live"
		fakeReplicaSetSolo = "fake-replicaset-solo"
		fakeDeploymentUID  = "fake-deployment-uid"
		revisionCurrent    = "3"
		revisionOld        = "2"
	)

	zero := int32(0)
	two := int32(2)

	newDeployment := func() *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeDeployment,
				Namespace: fakeNamespace,
				UID:       fakeDeploymentUID,
				Annotations: map[string]string{
					deploymentRevisionAnnotation: revisionCurrent,
				},
			},
		}
	}

	rsOwner := func(name, revision string) *appsv1.ReplicaSet {
		controller := true
		return &appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{Kind: resource.KindReplicaSet},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: fakeNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       fakeDeployment,
						UID:        fakeDeploymentUID,
						Controller: &controller,
					},
				},
				Annotations: map[string]string{
					deploymentRevisionAnnotation: revision,
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &zero,
			},
		}
	}

	tests := []struct {
		name        string
		rs          *appsv1.ReplicaSet
		deployments []*appsv1.Deployment
		want        bool
	}{
		{
			name:        "ReplicaSet with 0 replicas and old revision should be deleted",
			rs:          rsOwner(fakeReplicaSetOld, revisionOld),
			deployments: []*appsv1.Deployment{newDeployment()},
			want:        true,
		},
		{
			name:        "ReplicaSet with 0 replicas but live revision of its Deployment should not be deleted",
			rs:          rsOwner(fakeReplicaSetLive, revisionCurrent),
			deployments: []*appsv1.Deployment{newDeployment()},
			want:        false,
		},
		{
			name: "ReplicaSet with 0 replicas and no owner should be deleted",
			rs: &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{Kind: resource.KindReplicaSet},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeReplicaSetSolo,
					Namespace: fakeNamespace,
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: &zero,
				},
			},
			deployments: []*appsv1.Deployment{newDeployment()},
			want:        true,
		},
		{
			name: "ReplicaSet with non-zero replicas should not be deleted",
			rs: &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{Kind: resource.KindReplicaSet},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeReplicaSetSolo,
					Namespace: fakeNamespace,
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: &two,
				},
			},
			deployments: []*appsv1.Deployment{newDeployment()},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				deployments: tt.deployments,
			}

			got, err := d.DetermineDeletion(context.Background(), &cliresource.Info{
				Name:      tt.rs.Name,
				Namespace: fakeNamespace,
				Object:    tt.rs,
			})
			if err != nil {
				t.Errorf("determiner.DetermineDeletion() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_PersistentVolume(t *testing.T) {
	const (
		fakePersistentVolume       = "fake-pv"
		fakePersistentVolumeClaim1 = "fake-pvc1"
		fakePersistentVolumeClaim2 = "fake-pvc2"
		fakeLabelKey               = "fake-label-key"
		fakeLabelValue             = "fake-label-value"
	)

	var orgCheckVolumeSatisfyClaimFunc func(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool
	orgCheckVolumeSatisfyClaimFunc, checkVolumeSatisfyClaimFunc =
		checkVolumeSatisfyClaimFunc,
		func(volume *corev1.PersistentVolume, claim *corev1.PersistentVolumeClaim) bool {
			return volume.Labels[fakeLabelKey] == claim.Labels[fakeLabelKey]
		}
	t.Cleanup(func() {
		checkVolumeSatisfyClaimFunc = orgCheckVolumeSatisfyClaimFunc
	})

	type fields struct {
		persistentVolumeClaims []*corev1.PersistentVolumeClaim
	}
	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "PersistentVolume should be deleted when it is not used",
			fields: fields{},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolume,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolume,
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:   "PersistentVolume in Bound phase should never be deleted",
			fields: fields{},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolume,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolume,
						},
						Status: corev1.PersistentVolumeStatus{
							Phase: corev1.VolumeBound,
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PersistentVolume should not be deleted when it is used",
			fields: fields{
				persistentVolumeClaims: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fakePersistentVolumeClaim1,
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fakePersistentVolumeClaim2,
						},
					},
				},
			},
			args: args{
				info: &cliresource.Info{
					Name: fakePersistentVolume,
					Object: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindPersistentVolume,
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								fakeLabelKey: fakeLabelValue,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				persistentVolumeClaims: tt.fields.persistentVolumeClaims,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_DetermineDeletion_HorizontalPodAutoscaler(t *testing.T) {
	const (
		fakeNamespace                = "fake-ns"
		fakeHorizontalPodAutoscaler  = "fake-hpa"
		fakeScaleTargetRefAPIVersion = "apps/v1"
		fakeScaleTargetRefKind       = "Deployment"
		fakeScaleTargetRefName       = "fake-deploy"
	)

	type args struct {
		info *cliresource.Info
	}

	tests := []struct {
		name        string
		args        args
		fakeObjects []runtime.Object
		want        bool
		wantErr     bool
	}{
		{
			name: "HorizontalPodAutoscaler should be deleted when it is not used",
			args: args{
				info: &cliresource.Info{
					Name: fakeHorizontalPodAutoscaler,
					Object: &autoscalingv1.HorizontalPodAutoscaler{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindHorizontalPodAutoscaler,
						},
						Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
								APIVersion: fakeScaleTargetRefAPIVersion,
								Kind:       fakeScaleTargetRefKind,
								Name:       fakeScaleTargetRefName,
							},
						},
					},
				},
			},
			fakeObjects: []runtime.Object{},
			want:        true,
			wantErr:     false,
		},
		{
			name: "HorizontalPodAutoscaler should not be deleted when it is used",
			args: args{
				info: &cliresource.Info{
					Name:      fakeHorizontalPodAutoscaler,
					Namespace: fakeNamespace,
					Object: &autoscalingv1.HorizontalPodAutoscaler{
						TypeMeta: metav1.TypeMeta{
							Kind: resource.KindHorizontalPodAutoscaler,
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      fakeHorizontalPodAutoscaler,
							Namespace: fakeNamespace,
						},
						Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
								APIVersion: fakeScaleTargetRefAPIVersion,
								Kind:       fakeScaleTargetRefKind,
								Name:       fakeScaleTargetRefName,
							},
						},
					},
				},
			},
			fakeObjects: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						APIVersion: fakeScaleTargetRefAPIVersion,
						Kind:       fakeScaleTargetRefKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeScaleTargetRefName,
						Namespace: fakeNamespace,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := resource.NewFakeClient(tt.fakeObjects...)
			if err != nil {
				t.Errorf("failed to construct fake resource client")
				return
			}

			d := &determiner{
				resourceClient: c,
			}

			got, err := d.DetermineDeletion(context.Background(), tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("determiner.DetermineDeletion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determiner.DetermineDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determiner_determineUsedPodDisruptionBudget(t *testing.T) {
	const (
		fakeNamespace           = "fake-ns"
		fakePodDisruptionBudget = "fake-pdb"
		fakeLabelKey1           = "fake-label1-key"
		fakeLabelValue1         = "fake-label1-value"
		fakeLabelKey2           = "fake-label2-key"
		fakeLabelValue2         = "fake-label2-value"
	)

	type fields struct {
		pods []*corev1.Pod
	}
	type args struct {
		pdb *policyv1.PodDisruptionBudget
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "used PodDisruptionBudget should be determined with MatchLabels",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePodDisruptionBudget,
						Namespace: fakeNamespace,
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "used PodDisruptionBudget should be determined with MatchExpressions",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePodDisruptionBudget,
						Namespace: fakeNamespace,
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      fakeLabelKey1,
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{fakeLabelValue1, fakeLabelValue2},
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "used PodDisruptionBudget should not be determined when no Pods with corresponding label exist",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Labels: map[string]string{
								fakeLabelKey2: fakeLabelValue2,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePodDisruptionBudget,
						Namespace: fakeNamespace,
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "PodDisruptionBudget should not match Pods from another namespace",
			fields: fields{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "other-ns",
							Labels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			args: args{
				pdb: &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakePodDisruptionBudget,
						Namespace: fakeNamespace,
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								fakeLabelKey1: fakeLabelValue1,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				pods: tt.fields.pods,
			}

			got, err := d.determineUsedPodDisruptionBudget(tt.args.pdb)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineUsedPodDisruptionBudget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func Test_determiner_determineUsedSecret(t *testing.T) {
	const (
		fakeNamespace = "fake-ns"
		fakeSecret    = "fake-secret"
	)

	type fields struct {
		podSpecs  []podSpecRef
		sas       []*corev1.ServiceAccount
		ingresses []*networkingv1.Ingress
	}

	tests := []struct {
		name   string
		fields fields
		want   map[string]struct{}
	}{
		{
			name: "secrets used in ImagePullSecret should be determined as used",
			fields: fields{
				podSpecs: []podSpecRef{
					{
						namespace: fakeNamespace,
						spec: corev1.PodSpec{
							ImagePullSecrets: []corev1.LocalObjectReference{{Name: fakeSecret}},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
		{
			name: "secrets used in EnvFrom should be determined as used",
			fields: fields{
				podSpecs: []podSpecRef{
					{
						namespace: fakeNamespace,
						spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								EnvFrom: []corev1.EnvFromSource{
									{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: fakeSecret}}},
								},
							}},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
		{
			name: "secrets used in init containers should be determined as used",
			fields: fields{
				podSpecs: []podSpecRef{
					{
						namespace: fakeNamespace,
						spec: corev1.PodSpec{
							InitContainers: []corev1.Container{{
								EnvFrom: []corev1.EnvFromSource{
									{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: fakeSecret}}},
								},
							}},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
		{
			name: "secrets in ServiceAccount imagePullSecrets should be determined as used",
			fields: fields{
				sas: []*corev1.ServiceAccount{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						ImagePullSecrets: []corev1.LocalObjectReference{{Name: fakeSecret}},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
		{
			name: "secrets referenced by Ingress TLS should be determined as used",
			fields: fields{
				ingresses: []*networkingv1.Ingress{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
						},
						Spec: networkingv1.IngressSpec{
							TLS: []networkingv1.IngressTLS{
								{SecretName: fakeSecret},
							},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
		{
			name: "secrets referenced by nginx ingress auth annotations should be determined as used",
			fields: fields{
				ingresses: []*networkingv1.Ingress{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fakeNamespace,
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/auth-secret": fakeSecret,
							},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecret: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				podSpecs: tt.fields.podSpecs,
			}

			got := d.detectUsedSecrets(tt.fields.sas, tt.fields.ingresses, nil)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func Test_determiner_detectUsedConfigMaps(t *testing.T) {
	const (
		fakeNamespace = "fake-ns"
		fakeConfigMap = "fake-cm"
	)

	tests := []struct {
		name     string
		podSpecs []podSpecRef
		want     map[string]struct{}
	}{
		{
			name: "ConfigMaps referenced by CronJob templates should be determined as used",
			podSpecs: []podSpecRef{
				{
					namespace: fakeNamespace,
					spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Env: []corev1.EnvVar{
								{
									Name: "TEST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: fakeConfigMap},
											Key:                  "TEST",
										},
									},
								},
							},
						}},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeConfigMap: {}},
		},
		{
			name: "ConfigMaps referenced by init container volumes should be determined as used",
			podSpecs: []podSpecRef{
				{
					namespace: fakeNamespace,
					spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "cm-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: fakeConfigMap},
									},
								},
							},
						},
					},
				},
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeConfigMap: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{
				podSpecs: tt.podSpecs,
			}

			got := d.detectUsedConfigMaps()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func Test_determiner_detectUsedSecrets_ExternalSecret(t *testing.T) {
	const (
		fakeNamespace = "fake-ns"
		// Fake test fixture, not a real credential.
		fakeSecretExplicit = "fake-secret-explicit" //nolint:gosec
		fakeExternalSecret = "fake-external-secret"
	)

	// newExternalSecret builds an ExternalSecret CR; when targetName is set
	// the managed Secret is named explicitly, otherwise it defaults to the
	// object name (kubernetes-client.io behavior).
	newExternalSecret := func(namespace, name, targetName string) *unstructured.Unstructured {
		es := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "external-secrets.io/v1",
				"kind":       "ExternalSecret",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": namespace,
				},
				"spec": map[string]interface{}{},
			},
		}
		if targetName != "" {
			es.Object["spec"].(map[string]interface{})["target"] = map[string]interface{}{
				"name": targetName,
			}
		}
		return es
	}

	tests := []struct {
		name            string
		externalSecrets []*unstructured.Unstructured
		want            map[string]struct{}
	}{
		{
			name: "Secret with explicit spec.target.name should be determined as used",
			externalSecrets: []*unstructured.Unstructured{
				newExternalSecret(fakeNamespace, fakeExternalSecret, fakeSecretExplicit),
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeSecretExplicit: {}},
		},
		{
			name: "Secret bound by kubernetes-client.io ExternalSecret defaults to the object name",
			externalSecrets: []*unstructured.Unstructured{
				newExternalSecret(fakeNamespace, fakeExternalSecret, ""),
			},
			want: map[string]struct{}{fakeNamespace + "/" + fakeExternalSecret: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &determiner{}

			got := d.detectUsedSecrets(nil, nil, tt.externalSecrets)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
