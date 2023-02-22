# a8s-deployment

a8s is a Kubernetes-native platform for lifecycle automation of data services.

This repo contains:

- a list of the current major limitations of a8s.
- the requirements and prerequisites for the deployment of a8s.
- instructions and YAML manifests that platform operators can use to install and update the a8s
  control plane on a Kubernetes cluster.
- some guidance for application developers on how to use a8s to provision and manage (e.g. take a
backup) PostgreSQL instances that their applications can use.

> WARNING: **a8s is in beta** and the only data service that it currently supports is
PostgreSQL 13 and 14. **Don't use it for production workloads**. Some features may still be fragile
 and some breaking API changes may occur. For known issues and limitations please consult
[Current Limitations](docs/current_limitations.md).

## Main features

- Provision and dynamically configure a high availability PostgreSQL instance
- Bind an application to the PostgreSQL instance
- Backup and restore of the PostgreSQL instance
- Visualize the logs of the PostgreSQL instance and other components
- Visualize the metrics of the PostgreSQL instance and other components

## Index

- [CHANGELOG](CHANGELOG.md)
- [Technical Requirements](docs/technical_requirements.md)
- [Platform Operator Documentation](docs/platform-operators/README.md)
  - [Install the a8s Control Plane](/docs/platform-operators/installing_framework.md#/install-the-a8s-control-plane)
  - [Update the a8s Control Plane](/docs/platform-operators/updating_framework.md)
- [Application Developer Documentation](docs/application-developers/README.md)
  - [Usage Overview](docs/application-developers/usage_overview.md)
  - [Advanced Configuration](/docs/application-developers/advanced_configuration.md)
  - [API Documentation](/docs/application-developers/api-documentation/README.md)
    - [a8s-backup-manager](/docs/application-developers/api-documentation/a8s-backup-manager)
    - [a8s-service-binding-controller](/docs/application-developers/api-documentation/a8s-service-binding-controller)
    - [postgresql-operator](/docs/application-developers/api-documentation/postgresql-operator)
    - [Labels of DSI Secondary Resources](/docs/application-developers/api-documentation/labels_secondary_dsi_objects.md)
- [Current Limitations](docs/current_limitations.md)
- [Broken Link for testing](https://github.com/anynines/a8s-deployment/blob/a8s_1035_testing_broken_link_checking/README.BROKEN)
