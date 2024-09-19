# API Reference

## Packages
- [config.servicebindings.anynines.com/v1beta3](#configservicebindingsanyninescomv1beta3)

## config.servicebindings.anynines.com/v1beta3

Package v1beta3 contains API Schema definitions for the config v1beta3 API group

### Resource Types
- [ControllerManagerConfig](#controllermanagerconfig)

#### ControllerHealth

ControllerHealth defines the health configs.

_Appears in:_
- [ControllerManagerConfig](#controllermanagerconfig)

| Field | Description |
| --- | --- |
| `healthProbeBindAddress` _string_ | HealthProbeBindAddress is the TCP address that the controller should bind to for serving health probes It can be set to "0" or "" to disable serving the health probe. Defaults to ":8081". |
| `readinessEndpointName` _string_ | Readiness endpoint name. Must start with a '/'. |
| `livenessEndpointName` _string_ | Liveness endpoint name. Must start with a '/'. |

#### ControllerManagerConfig

ControllerManagerConfig is the Schema for the manager's configuration API TEST

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `config.servicebindings.anynines.com/v1beta3`
| `kind` _string_ | `ControllerManagerConfig`
| `leaderElection` _[LeaderElection](#leaderelection)_ | LeaderElection contains the leader election configuration |
| `metrics` _[ControllerMetrics](#controllermetrics)_ | Metrics contains the controller metrics configuration |
| `health` _[ControllerHealth](#controllerhealth)_ | Health contains the controller health configuration |

#### ControllerMetrics

ControllerMetrics defines the metrics configs.

_Appears in:_
- [ControllerManagerConfig](#controllermanagerconfig)

| Field | Description |
| --- | --- |
| `bindAddress` _string_ | BindAddress is the TCP address that the controller should bind to for serving prometheus metrics. It can be set to "0" to disable the metrics serving. Defaults to "":8080". |

#### LeaderElection

LeaderElection defines the leader election configs.

_Appears in:_
- [ControllerManagerConfig](#controllermanagerconfig)

| Field | Description |
| --- | --- |
| `leaderElect` _boolean_ | leaderElect enables a leader election client to gain leadership before executing the main loop. Enable this when running replicated components for high availability. |
| `leaderElectionNamespace` _string_ | LeaderElectionNamespace indicates the namespace of resource object that will be used to lock during leader election cycles. |

