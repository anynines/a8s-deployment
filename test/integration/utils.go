package integration

import (
	"os/exec"

	corev1 "k8s.io/api/core/v1"

	"github.com/anynines/postgresql-operator/api/v1alpha1"
)

func kubectlExec(namespace, podname, containerName string, args ...string) ([]byte, error) {
	kubectlArgs := append([]string{
		"-n",
		namespace,
		"exec",
		podname,
		"-c",
		containerName,
		"--",
	}, args...)

	return kubectl(kubectlArgs...)
}

func kubectl(args ...string) ([]byte, error) {
	cmd := exec.Command("kubectl", args...)
	return cmd.CombinedOutput()
}

func newDSI(namespace, instanceName string, replicas int32) *v1alpha1.Postgresql {
	return &v1alpha1.Postgresql{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "postgresql.anynines.com/v1alpha1",
			Kind:       "Postgresql",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgresqlSpec{
			Replicas:         pointer.Int32Ptr(replicas),
			BackupAgentImage: "c4aeb71c-dd2a-4e6e-9385-1c3bb839307c.de.a9s.eu/demo/backup-agent:0.3.0",
			Image:            "registry.opensource.zalan.do/acid/spilo-13:2.0-p2",
			Resources: &corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse("100m"),
					corev1.ResourceMemory: k8sresource.MustParse("100Mi"),
				},
				Requests: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse("100m"),
					corev1.ResourceMemory: k8sresource.MustParse("100Mi"),
				},
			},
		},
	}
}

