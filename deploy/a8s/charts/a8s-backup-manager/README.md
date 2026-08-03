# a8s-backup-manager Helm Chart

Helm chart for deploying the a8s backup-manager component, which manages backup and recovery operations for PostgreSQL databases in Kubernetes.

## Configuration

### Backup Storage Configuration

The backup-manager requires configuration for S3/cloud storage backup destinations. This is done through two Kubernetes resources:

1. **ConfigMap** (`a8s-backup-store-config`): Stores the backup storage coordinates (provider, region, container)
2. **Secret** (`a8s-backup-storage-credentials`): Stores credentials for accessing the backup storage

### Using an Existing Secret

If you already manage a Secret with backup storage credentials, reference it
with `existingSecret`. This takes precedence over inline credentials:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.secret.existingSecret=a8s-backup-storage-credentials
```

By default the chart expects the keys `access-key-id`, `secret-access-key` and
`encryption-password`. If your Secret uses different keys, override
`backupStorageConfig.secret.accessKeyIdKey`, `.secretAccessKeyKey` and
`.encryptionPasswordKey`.

### Creating Secret via Helm Values

Provide credentials directly via Helm values to have the chart create the Secret:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.secret.accessKeyId="YOUR_AWS_KEY" \
  --set backupStorageConfig.secret.secretAccessKey="YOUR_AWS_SECRET" \
  --set backupStorageConfig.secret.encryptionPassword="YOUR_ENCRYPTION_PASSWORD"
```

> **Not recommended for production.** For production, manage the Secret yourself and reference it
> with `existingSecret` — see [Using an Existing Secret](#using-an-existing-secret).

### Customizing Backup Storage Configuration

To configure the S3 backup storage coordinates:

```bash
helm install a8s-backup-manager ./a8s-backup-manager \
  --set backupStorageConfig.configMap.provider="AWS" \
  --set backupStorageConfig.configMap.container="my-backup-bucket" \
  --set backupStorageConfig.configMap.region="us-east-1"
  --set backupStorageConfig.configMap.endpoint="https://<custom-s3-endpoint>"
```

Or using a values file:

```yaml
# values-override.yaml
backupStorageConfig:
  configMap:
    provider: "AWS"
    container: "my-backup-bucket"
    region: "us-east-1"
  secret:
    accessKeyId: "YOUR_AWS_KEY"
    secretAccessKey: "YOUR_AWS_SECRET"
    encryptionPassword: "YOUR_ENCRYPTION_PASSWORD"
```

Then install:

```bash
helm install a8s-backup-manager ./a8s-backup-manager -f values-override.yaml
```

## Required Kubernetes Credentials

To create the backup storage Secret yourself before installing the chart,
then reference it via `backupStorageConfig.secret.existingSecret` (see
[Using an Existing Secret](#using-an-existing-secret) for the key details):

```bash
kubectl create secret generic a8s-backup-storage-credentials \
  --from-literal=access-key-id="YOUR_KEY" \
  --from-literal=secret-access-key="YOUR_SECRET" \
  --from-literal=encryption-password="YOUR_PASSWORD" \
  -n a8s-system
```

## Chart Values

Key configurable values:

- `crd.enable`: Install CRDs via Helm
- `crd.keep`: Keep CRDs on uninstall (avoids potential data loss)
- `image.repository`: Docker image repository
- `image.tag`: Image tag/version
- `replicaCount`: Number of backup-manager replicas
- `backupStorageConfig.configMap.name`: ConfigMap name for storage coordinates
- `backupStorageConfig.configMap.provider`: Storage provider (e.g. `AWS`)
- `backupStorageConfig.configMap.container`: Bucket/container name (required)
- `backupStorageConfig.configMap.region`: Bucket region (required)
- `backupStorageConfig.configMap.endpoint`: Endpoint URL for S3-compatible storage (optional)
- `backupStorageConfig.configMap.pathStyle`: Use path-style addressing for S3-compatible storage (optional)
- `backupStorageConfig.secret.name`: Secret name the chart creates for inline credentials
- `backupStorageConfig.secret.existingSecret`: Reference an externally managed credentials Secret (takes precedence over inline credentials)
- `backupStorageConfig.secret.accessKeyIdKey` / `.secretAccessKeyKey` / `.encryptionPasswordKey`: Key names within `existingSecret`
- `umbrellaValuePathPrefix`: Set by a parent umbrella chart so validation error messages point at the right value path (leave empty for standalone installs)
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
