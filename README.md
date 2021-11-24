# a8s-deployment

a8s is a Kubernetes-native platform for lifecycle automation of data services.

This repo contains:
- a list of the current major limitations of a8s.
- the requirements and prerequisites for the deployment of a8s.
- instructions and YAML manifests that platform operators can use to install the a8s control plane
on a Kubernetes cluster.
- some guidance for application developers on how to use a8s to provision and manage (e.g. take a
backup) PostgreSQL instances that their applications can use.

> WARNING: **a8s is in alpha** and the only data service that it currently supports is
PostgreSQL 13. **Don't use it for production workloads**. All features are still very fragile and all
APIs can change at any time. For known issues and limitations please consult 
[Current Limitations](docs/current_limitations.md).

## Main features
- Provision and dynamically configure a high availability PostgreSQL instance
- Bind an application to the PostgreSQL instance
- Backup and restore of the PostgreSQL instance
- Visualize the logs of the PostgreSQL instance and other components
- Visualize the metrics of the PostgreSQL instance and other components

## Index

- [Current Limitations](docs/current_limitations.md)
- [Technical Requirements](docs/technical_requirements.md)
- [Platform Operator Documentation](docs/platform_operators.md)
- [Application Developer Documentation](docs/application_developers.md)
