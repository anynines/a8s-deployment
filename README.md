# a8s-deployment

a8s is a Kubernetes-native platform for lifecycle automation of data services.

This repo contains:

- instructions and YAML manifests that platform operators can use to install the a8s control plane
on a Kubernetes cluster.
- some guidance for application developers on how to use a8s to provision and manage (e.g. take a
backup) PostgreSQL instances that their applications can use.

> WARNING: **a8s is in pre-alpha** and the only data service that it currently supports is
PostgreSQL. **Don't use it for production workloads**. All features are still very fragile and all
APIs can change at any time.

## Main features
- Provision and dynamically configure a high availability PostgreSQL instance
- Bind an application to the PostgreSQL instance
- Backup and restore of the PostgreSQL instance
- Visualize the logs of the PostgreSQL instance and other components
- Visualize the metrics of the PostgreSQL instance and other components

## Index

- [Platform Operator Documentation](docs/platform_operators.md)
- [Application Developer Documentation](docs/application_developers.md)
