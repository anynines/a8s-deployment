package postgresql

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/anynines/a8s-backup-manager/api/v1alpha1"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
	pgv1alpha1 "github.com/anynines/postgresql-operator/api/v1alpha1"
)

const (
	resourceCPU  = "500m"
	resourceMem  = "500Mi"
	volumeSizeGB = 1
	version      = 14

	kind = "Postgresql"
)

type Postgresql struct {
	*pgv1alpha1.Postgresql
}

func (pg Postgresql) ClusterStatus() string {
	return pg.Status.ClusterStatus
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
			Replicas:      pointer.Int32Ptr(replicas),
			VolumeSizeGiB: volumeSizeGB,
			Version:       version,
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
