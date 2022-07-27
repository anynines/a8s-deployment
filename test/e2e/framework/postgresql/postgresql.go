package postgresql

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/anynines/a8s-backup-manager/api/v1alpha1"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
	pgv1alpha1 "github.com/anynines/postgresql-operator/api/v1alpha1"
)

const (
	resourceCPU = "500m"
	resourceMem = "500Mi"
	volumeSize  = "1G"
	version     = 14

	kind = "Postgresql"
)

type Postgresql struct {
	*pgv1alpha1.Postgresql
}

func (pg Postgresql) ClusterStatus() string {
	return pg.Status.ClusterStatus
}

// TODO: make the K8s client a field of pg rather than something to pass to its functions.
func (pg Postgresql) StatefulSet(ctx context.Context,
	k8sClient runtimeClient.Client) (*appsv1.StatefulSet, error) {

	nsn := types.NamespacedName{Namespace: pg.Namespace, Name: pg.Name}
	ss := &appsv1.StatefulSet{}

	if err := k8sClient.Get(ctx, nsn, ss); err != nil {
		return nil, fmt.Errorf("failed to get statefulset for instance %s: %w", nsn, err)
	}

	return ss, nil
}

// Pods uses `k8sClient` to retrieve all the pods that belong to `pg`. The retrieval is based
// entirely on metadata labels, and doesn't check whether the pods actually belong to `pg` or to an
// older DSI with same namespace, name and kind.
// TODO: Improve robustness by checking that the pods actually belong to DSI.
func (pg Postgresql) Pods(ctx context.Context,
	k8sClient runtimeClient.Client) ([]corev1.Pod, error) {

	podsLabels := labels.Set{
		pgv1alpha1.DSINameLabelKey:  pg.Name,
		pgv1alpha1.DSIGroupLabelKey: "postgresql.anynines.com",
		pgv1alpha1.DSIKindLabelKey:  "Postgresql",
	}
	podsSelector, err := podsLabels.AsValidatedSelector()
	if err != nil {
		return nil, fmt.Errorf("failed to generate label selector for pods of %#+v: %w",
			pg.Postgresql, err)
	}

	listOpts := &runtimeClient.ListOptions{
		LabelSelector: podsSelector,
		Namespace:     pg.Namespace,
	}

	pods := &corev1.PodList{}
	if err := k8sClient.List(ctx, pods, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list pods for %#+v: %w", pg.Postgresql, err)
	}

	return pods.Items, nil
}

func (pg Postgresql) SetTolerations(ts ...corev1.Toleration) {
	if pg.Postgresql.Spec.SchedulingConstraints == nil {
		pg.Postgresql.Spec.SchedulingConstraints = &pgv1alpha1.PostgresqlSchedulingConstraints{}
	}
	pg.Postgresql.Spec.SchedulingConstraints.Tolerations = ts
}

func (pg Postgresql) AddRequiredPodAntiAffinityTerm(at corev1.PodAffinityTerm) {
	pg.initPodAntiAffinity()
	paa := pg.Postgresql.Spec.SchedulingConstraints.Affinity.PodAntiAffinity
	paa.RequiredDuringSchedulingIgnoredDuringExecution =
		append(paa.RequiredDuringSchedulingIgnoredDuringExecution, at)
}

func (pg Postgresql) AddPreferredPodAntiAffinityTerm(weight int, at corev1.PodAffinityTerm) {
	pg.initPodAntiAffinity()
	paa := pg.Postgresql.Spec.SchedulingConstraints.Affinity.PodAntiAffinity
	paa.PreferredDuringSchedulingIgnoredDuringExecution =
		append(paa.PreferredDuringSchedulingIgnoredDuringExecution, corev1.WeightedPodAffinityTerm{
			Weight:          int32(weight),
			PodAffinityTerm: at,
		})
}

func (pg Postgresql) initPodAntiAffinity() {
	if pg.Postgresql.Spec.SchedulingConstraints == nil {
		pg.Postgresql.Spec.SchedulingConstraints = &pgv1alpha1.PostgresqlSchedulingConstraints{}
	}
	if pg.Postgresql.Spec.SchedulingConstraints.Affinity == nil {
		pg.Postgresql.Spec.SchedulingConstraints.Affinity = &corev1.Affinity{}
	}
	if pg.Postgresql.Spec.SchedulingConstraints.Affinity.PodAntiAffinity == nil {
		pg.Postgresql.Spec.SchedulingConstraints.Affinity.
			PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
}

// GetClientObject exposes the embedded PostgreSQL object to methods/functions that expect that
// type. We also need to set the APIVersion and Kind since Kubernetes will remove these fields when
// marshalling API objects.
func (pg Postgresql) GetClientObject() runtimeClient.Object {
	pg.Postgresql.APIVersion = pgv1alpha1.GroupVersion.String()
	pg.Postgresql.Kind = kind
	return pg.Postgresql
}

func NewK8sClient(kubeconfig string) (runtimeClient.Client, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build config from kubeconfig path: %w", err)
	}

	pgv1alpha1.AddToScheme(scheme.Scheme)
	sbv1alpha1.AddToScheme(scheme.Scheme)
	backupv1alpha1.AddToScheme(scheme.Scheme)

	k8sClient, err := runtimeClient.New(cfg, runtimeClient.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, fmt.Errorf("unable to create new Kubernetes client for tests: %w", err)
	}
	return k8sClient, nil
}

func New(namespace, name string, replicas int32) *Postgresql {
	return &Postgresql{&pgv1alpha1.Postgresql{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: pgv1alpha1.GroupVersion.String(),
		},
		Spec: pgv1alpha1.PostgresqlSpec{
			Replicas:   pointer.Int32Ptr(replicas),
			VolumeSize: k8sresource.MustParse(volumeSize),
			Version:    version,
			Resources: &corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse(resourceCPU),
					corev1.ResourceMemory: k8sresource.MustParse(resourceMem),
				},
				Requests: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse(resourceCPU),
					corev1.ResourceMemory: k8sresource.MustParse(resourceMem),
				},
			},
		},
	}}
}

func NewEmpty() Postgresql {
	return Postgresql{&pgv1alpha1.Postgresql{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: pgv1alpha1.GroupVersion.String(),
		},
	}}
}

func MasterService(instanceName string) string {
	return fmt.Sprintf("%s-%s", instanceName, "master")
}

func AdminRoleSecretName(instanceName string) string {
	return fmt.Sprintf("%s.%s", "postgres.credentials", instanceName)
}

func StandbyRoleSecretName(instanceName string) string {
	return fmt.Sprintf("%s.%s", "standby.credentials", instanceName)
}

func PvcName(instanceName string, index int) string {
	return fmt.Sprintf("%s-%s-%d", "pgdata", instanceName, index)
}
