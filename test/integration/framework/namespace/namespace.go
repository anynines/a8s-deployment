package namespace

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateIfNotExists(ctx context.Context,
	testingNamespace string,
	c runtimeClient.Client) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingNamespace,
		},
	}

	err := c.Create(ctx, ns)
	if k8serrors.IsAlreadyExists(err) {
		log.Println("The namespace already exists. Skipping namespace creation for: ",
			testingNamespace)
		return nil
	}
	return err
}

func DeleteIfAllowed(ctx context.Context,
	testingNamespace string,
	c runtimeClient.Client) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingNamespace,
		},
	}

	err := c.Delete(ctx, ns)
	if err != nil && k8serrors.IsForbidden(err) {
		// TODO: Use structured logging using context where stdlib log or fmt is used for
		// logging.
		log.Println("The namespace is forbidden from deletion: ", testingNamespace)

		return nil
	}
	return err
}
