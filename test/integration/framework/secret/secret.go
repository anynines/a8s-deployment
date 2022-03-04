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

func ParseRawSecretData(raw map[string][]byte) SecretData {
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

	s, err := Get(ctx, k8sClient, secretName, secretNamespace)
	if err != nil {
		return nil, err
	}

	return ParseRawSecretData(s.Data), nil
}

func Get(ctx context.Context,
	k8sClient client.Client,
	secretName,
	secretNamespace string) (corev1.Secret, error) {

	var s corev1.Secret
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace}, &s)
	if err != nil {
		return s, fmt.Errorf("failed to get service binding secret %w", err)
	}

	return s, nil
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

	return ParseRawSecretData(s.Data), nil
}
