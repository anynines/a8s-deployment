# a8s Service Binding Controller Helm Chart

Helm chart for deploying the a8s Service Binding Controller, which manages service binding between applications and data service instances (PostgreSQL databases).

## Overview

The a8s Service Binding Controller is responsible for:

- Discovering data service instances in the cluster
- Creating and managing service bindings from applications to databases
- Injecting connection credentials into application workloads
- Supporting PostgreSQL integration (when enabled)
- Following the Kubernetes Service Binding specification

## Prerequisites

Before installing this chart, ensure:

1. **Kubernetes cluster** (1.19+) is available and accessible
2. **Helm 3.0+** is installed
3. **cert-manager** is installed on the cluster (for webhook TLS certificates)
4. **(Optional) PostgreSQL Operator** is installed if you want PostgreSQL integration

## Quick Start

### Basic Installation

```bash
helm install a8s-service-binding-controller ./deploy/a8s/charts/a8s-service-binding-controller \
  --namespace a8s-system \
  --create-namespace
```

### Enable PostgreSQL Integration

```bash
helm install a8s-service-binding-controller ./deploy/a8s/charts/a8s-service-binding-controller \
  --namespace a8s-system \
  --create-namespace \
  --set controllerConfig.enable-integration=postgresql
```

> **Note:** PostgreSQL integration is enabled by default in both the `Basic Installation` and `Enable PostgreSQL Integration` examples above. They are functionally equivalent. The explicit `--set controllerConfig.enable-integration=postgresql` flag is shown for clarity and future extensibility. To enable a different integration in the future, simply replace `postgresql` with the desired integration name:
>
> ```bash
> --set controllerConfig.enable-integration=<integration-name>
> ```

### Custom Image Tag

```bash
helm install a8s-service-binding-controller ./deploy/a8s/charts/a8s-service-binding-controller \
  --namespace a8s-system \
  --create-namespace \
  --set image.tag=your-tag
```

## Configuration

### Integration Options

Set `postgresql` activate the integration for the service binding controller.

```yaml
controllerConfig:
  enable-integration: "postgresql"  # Enable PostgreSQL integration
```

See [values.yaml](values.yaml) for all available options.

### Controller Configuration

Key configurable values:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `namespace` | `a8s-system` | Kubernetes namespace for the controller |
| `replicaCount` | `1` | Number of controller replicas |
| `image.repository` | ECR public registry | Docker image repository |
| `image.tag` | `a8555999c8c6fd7a1544c599b47f458b172eb4ca` | Image tag/version |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `controllerConfig.enable-integration` | `postgresql` | Data service integration to enable |
| `controllerConfig.healthProbeBindAddress` | `:8081` | Health probe address |
| `controllerConfig.metricsBindAddress` | `127.0.0.1:8080` | Metrics address |
| `controllerConfig.leaderElect` | `true` | Enable leader election |
| `resources` | See values.yaml | CPU/memory limits and requests |
| `nodeSelector` | `{}` | Node selection constraints |
| `affinity` | `{}` | Pod affinity rules |
| `tolerations` | `[]` | Pod tolerations |

## Verification

Verify the controller is running:

```bash
kubectl get deployment -n a8s-system a8s-service-binding-controller-manager
```

Expected output:

```bash
NAME                                                   READY   UP-TO-DATE   AVAILABLE   AGE
a8s-service-binding-controller-manager                 1/1     1            1           2m
```

Check the webhook service:

```bash
kubectl get service -n a8s-system -l app.kubernetes.io/name=a8s-service-binding-controller
```

View logs:

```bash
kubectl logs -n a8s-system -l app.kubernetes.io/name=a8s-service-binding-controller -f
```

## Integration with Other Components

- **PostgreSQL Operator**: When PostgreSQL integration is enabled, the controller discovers PostgreSQL instances created by the a8s-postgresql-operator and enables binding to them.
- **Backup Manager**: Service bindings work alongside the a8s-backup-manager to provide persistent databases with backup capabilities.

## Troubleshooting

### PostgreSQL Integration Not Working

Verify PostgreSQL integration is enabled:

```bash
helm get values a8s-service-binding-controller -n a8s-system | grep enable-integration
```

Ensure the PostgreSQL Operator is installed and running:

```bash
kubectl get deployment -n a8s-system a8s-postgresql-operator-controller-manager
```

### Service Binding Not Creating

Check the ServiceBinding status:

```bash
kubectl describe servicebinding my-app-to-db -n default
```

View controller logs for errors:

```bash
kubectl logs -n a8s-system -l app.kubernetes.io/name=a8s-service-binding-controller --tail=50
```

Verify the application and data service exist:

```bash
kubectl get deployment my-app -n default
kubectl get postgresql my-database -n default
```

## Documentation

For more information, see:

- [Main installation guide](../../../docs/platform-operators/installing_framework.md)
- [Service Binding usage guide](../../../docs/application-developers/advanced_configuration.md)
- [API documentation](../../../docs/application-developers/api-documentation/a8s-service-binding-controller/)
- [Kubernetes Service Binding specification](https://servicebinding.io/)
