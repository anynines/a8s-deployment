# a8s PostgreSQL Operator Helm Chart

Helm chart for deploying the a8s PostgreSQL Operator, which manages PostgreSQL database instances (DSIs) in Kubernetes.

## Overview

The a8s PostgreSQL Operator is responsible for:

- Creating and managing PostgreSQL instances from custom resources
- Configuring PostgreSQL with Patroni for high availability
- Managing PostgreSQL image versions and extensions
- Integrating with the backup-manager for automated backups
- Providing webhook validation for PostgreSQL resources

## Prerequisites

Before installing this chart, ensure:

1. **Kubernetes cluster** (1.19+) is available and accessible
2. **Helm 3.0+** is installed
3. **cert-manager** is installed on the cluster (for webhook TLS certificates)

## Quick Start

### Basic Installation

```bash
helm install a8s-postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
  --namespace a8s-system \
  --create-namespace
```

### Custom Image Tag

```bash
helm install a8s-postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
  --namespace a8s-system \
  --create-namespace \
  --set image.tag=your-tag
```

### PostgreSQL Image Configuration

Configure the images used for PostgreSQL instances (Spilo) and backup agents:

```bash
helm install a8s-postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
  --namespace a8s-system \
  --create-namespace \
  --set postgresqlImages.spiloImage=your-registry.com/spilo:14-latest \
  --set postgresqlImages.backupAgentImage=your-registry.com/backup-agent:latest
```

Or using a values file:

```yaml
# values-override.yaml
postgresqlImages:
  spiloImage: "your-registry.com/spilo:14-latest"
  backupAgentImage: "your-registry.com/backup-agent:latest"
```

Then install:

```bash
helm install a8s-postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
  --namespace a8s-system \
  --create-namespace \
  -f values-override.yaml
```

## Configuration

### Operator Configuration

Key configurable values:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `namespace` | `a8s-system` | Kubernetes namespace for the operator |
| `replicaCount` | `1` | Number of operator replicas |
| `image.repository` | ECR public registry | Docker image repository |
| `image.tag` | `a8555999c8c6fd7a1544c599b47f458b172eb4ca` | Image tag/version |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `resources` | See values.yaml | CPU/memory limits and requests |
| `nodeSelector` | `{}` | Node selection constraints |
| `affinity` | `{}` | Pod affinity rules |
| `tolerations` | `[]` | Pod tolerations |

### Image Configuration

Configure the images used when creating PostgreSQL instances:

```yaml
postgresqlImages:
  spiloImage: "public.ecr.aws/w5n9a2g2/a9s-ds-for-k8s/dev/spilo:14-latest"
  backupAgentImage: "public.ecr.aws/w5n9a2g2/a9s-ds-for-k8s/dev/backup-agent:latest"
```

These images can be:

- Mirrored to your internal registry
- Updated when a8s releases new versions
- Pinned to specific versions for reproducibility

See [values.yaml](values.yaml) for all available options.

## Integration with Other Components

- **Backup Manager**: The PostgreSQL Operator integrates with a8s-backup-manager to enable automated backups. Install the backup-manager separately or alongside the operator.
- **Service Binding Controller**: The a8s-service-binding-controller discovers PostgreSQL instances and creates service bindings.

## Troubleshooting

### Webhook Certificate Issues

If you see webhook errors, ensure cert-manager is installed and running:

```bash
kubectl get pods -n cert-manager
```

The operator creates self-signed certificates automatically. For production, configure cert-manager for proper CA management.

### Operator Not Starting

Check operator logs:

```bash
kubectl logs -n a8s-system deployment/a8s-postgresql-operator-controller-manager
kubectl describe pod -n a8s-system -l app.kubernetes.io/name=a8s-postgresql-operator
```

Ensure the namespace exists:

```bash
kubectl get namespace a8s-system
```

## Documentation

For more information, see:

- [Main installation guide](../../../docs/platform-operators/installing_framework.md)
- [API documentation](../../../docs/application-developers/api-documentation/a8s-postgresql-operator/)
- [PostgreSQL usage overview](../../../docs/application-developers/usage_overview.md)
