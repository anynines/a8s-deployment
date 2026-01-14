# a8s-backup-manager Helm Chart

Helm chart for deploying the a8s backup-manager component, which manages backup and recovery operations for PostgreSQL databases in Kubernetes.

## Configuration

### Backup Storage Configuration

The backup-manager requires configuration for S3/cloud storage backup destinations. This is done through two Kubernetes resources:

1. **ConfigMap** (`a8s-backup-store-config`): Stores the backup storage coordinates (provider, region, container)
2. **Secret** (`a8s-backup-storage-credentials`): Stores credentials for accessing the backup storage

### Using Existing Secret

If you already have a Secret with backup storage credentials:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.secret.create=false \
  --set backupStorageConfig.secret.name=a8s-backup-storage-credentials
```

### Creating Secret via Helm Values

Provide credentials directly via Helm values to have the chart create the Secret:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.secret.create=true \
  --set backupStorageConfig.secret.accessKeyId="YOUR_AWS_KEY" \
  --set backupStorageConfig.secret.secretAccessKey="YOUR_AWS_SECRET" \
  --set backupStorageConfig.secret.encryptionPassword="YOUR_ENCRYPTION_PASSWORD"
```

### Customizing Backup Storage Configuration

To configure the S3 backup storage coordinates:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.configMap.config.cloud_configuration.provider="AWS" \
  --set backupStorageConfig.configMap.config.cloud_configuration.container="my-backup-bucket" \
  --set backupStorageConfig.configMap.config.cloud_configuration.region="us-east-1"
```

Or using a values file:

```yaml
# values-override.yaml
backupStorageConfig:
  configMap:
    config:
      cloud_configuration:
        provider: "AWS"
        container: "my-backup-bucket"
        region: "us-east-1"
  secret:
    create: true
    accessKeyId: "YOUR_AWS_KEY"
    secretAccessKey: "YOUR_AWS_SECRET"
    encryptionPassword: "YOUR_ENCRYPTION_PASSWORD"
```

Then install:

```bash
helm install a8s-backup-manager ./a8s-backup-manager -f values-override.yaml
```

## Required Kubernetes Credentials

The backup-manager requires credentials stored in the `a8s-backup-storage-credentials` Secret with the following keys:

- `access-key-id`: AWS access key ID
- `secret-access-key`: AWS secret access key
- `encryption-password`: Password for encrypting backups

These can be created separately before installing the chart:

```bash
kubectl create secret generic a8s-backup-storage-credentials \
  --from-literal=access-key-id="YOUR_KEY" \
  --from-literal=secret-access-key="YOUR_SECRET" \
  --from-literal=encryption-password="YOUR_PASSWORD" \
  -n a8s-system
```

## Chart Values

Key configurable values:

- `image.repository`: Docker image repository
- `image.tag`: Image tag/version
- `replicaCount`: Number of backup-manager replicas
- `backupStorageConfig.configMap.name`: ConfigMap name for storage coordinates
- `backupStorageConfig.secret.name`: Secret name for credentials
- `backupStorageConfig.secret.create`: Whether to create the secret (default: true)
- `resources`: Resource limits and requests for containers
- `nodeSelector`: Node selection constraints
- `affinity`: Pod affinity rules
- `tolerations`: Pod tolerations

See `values.yaml` for all available options.

## Integration with backup-config

This chart replaces the static `deploy/a8s/backup-config/` directory approach. Instead of using Kustomize to generate the ConfigMap and Secret, this chart templates them dynamically with Helm, providing:

- Dynamic configuration via values
- Easy credential management
- Namespace-aware resources
- Consistent labeling and naming
