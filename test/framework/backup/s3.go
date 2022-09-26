package backup

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-backup-manager/api/v1alpha1"
)

const (
	namespace       = "a8s-system"
	secretName      = "a8s-backup-storage-credentials"
	configMapName   = "a8s-backup-store-config"
	backupConfigKey = "backup-store-config.yaml"
	s3Endpoint      = "s3.amazonaws.com"
	idKey           = "access-key-id"
	secretKey       = "secret-access-key"
)

type backupCfg struct {
	Config struct {
		CloudConfiguration struct {
			Provider  string `yaml:"provider"`
			Container string `yaml:"container"`
			Region    string `yaml:"region"`
		} `yaml:"cloud_configuration"`
	} `yaml:"config"`
}

type S3Client struct {
	client     minio.Client
	bucketName string
}

func (c S3Client) HasPartialBackupData(ctx context.Context, bkp v1alpha1.Backup) (bool, error) {
	for object := range c.client.ListObjects(ctx,
		c.bucketName,
		minio.ListObjectsOptions{}) {
		if object.Err != nil {
			return false, object.Err
		}

		if strings.Contains(object.Key, string(bkp.UID)) {
			return true, nil
		}
	}

	return false, nil
}

// NewS3Client creates a S3Client by taking the existing backup configuration in the cluster being
// tested. As the location of this configuration is not a part of the public API this might break.
func NewS3Client(k8sClient client.Client) (S3Client, error) {
	ctx := context.Background()

	backupStoreCreds := corev1.Secret{}

	err := k8sClient.Get(ctx,
		types.NamespacedName{Namespace: namespace, Name: secretName},
		&backupStoreCreds)
	if err != nil {
		return S3Client{}, fmt.Errorf("unable to get backup store credentials secret: %w", err)
	}

	backupStoreCfg, err := backupConfig(ctx, k8sClient)
	if err != nil {
		return S3Client{}, err
	}

	accessKeyID, err := valueOf(backupStoreCreds.Data, idKey)
	if err != nil {
		return S3Client{}, err
	}

	secretAccessKey, err := valueOf(backupStoreCreds.Data, secretKey)
	if err != nil {
		return S3Client{}, err
	}

	minioClient, err := minio.New(s3Endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Region: backupStoreCfg.Config.CloudConfiguration.Region,
			Secure: true,
		})
	if err != nil {
		return S3Client{}, fmt.Errorf("unable to create new minio client: %w", err)
	}

	return S3Client{
		client:     *minioClient,
		bucketName: backupStoreCfg.Config.CloudConfiguration.Container,
	}, nil
}

func valueOf(m map[string][]byte, key string) (string, error) {
	valBytes, ok := m[key]
	if !ok {
		return "", fmt.Errorf("unable to get value using key: %s", key)
	}

	trimmedValueStr := strings.TrimSpace(string(valBytes))

	return trimmedValueStr, nil
}

func backupConfig(ctx context.Context, k8sClient client.Client) (backupCfg, error) {
	cm := corev1.ConfigMap{}

	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: configMapName}, &cm)
	if err != nil {
		return backupCfg{}, fmt.Errorf("unable to get secret for backup store: %w", err)
	}

	yamlContents, ok := cm.Data[backupConfigKey]
	if !ok {
		return backupCfg{}, fmt.Errorf("unable to get config value from configmap")
	}

	config := backupCfg{}
	if err := yaml.Unmarshal([]byte(yamlContents), &config); err != nil {
		return backupCfg{}, fmt.Errorf("failed to unmarshal backup config: %w", err)
	}

	return config, nil
}
