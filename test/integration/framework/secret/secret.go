package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const pgAdminSecretPrefix = "postgres.credentials."

type SecretData map[string]string

func parseRawSecretData(raw map[string][]byte) SecretData {
	s := make(SecretData, len(raw))
	for key, bytes := range raw {
		s[key] = string(bytes)
	}
	return s
}

func Data(ctx context.Context,
	k8sClient client.Client,
	secretName,
	secretNamespace string) (SecretData, error) {

	var s corev1.Secret
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace}, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to get service binding secret %w", err)
	}
	return parseRawSecretData(s.Data), nil
}

func AdminSecretData(
	ctx context.Context,
	k8sClient client.Client,
	dsiName, dsiNamespace string,
) (SecretData, error) {

	namespacedName := types.NamespacedName{
		Name:      pgAdminSecretPrefix + dsiName,
		Namespace: dsiNamespace}

	var s corev1.Secret
	err := k8sClient.Get(ctx, namespacedName, &s)
	if err != nil {
		return nil, fmt.Errorf("unable to get service binding secret %s",
			namespacedName)
	}

	return parseRawSecretData(s.Data), nil
}
