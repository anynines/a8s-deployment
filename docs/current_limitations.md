# Current Limitations

## General Limitations

- The a8s control plane components (PostgreSQL-Operator, a8s-Backup-Manager, a8s-
  Service-Binding-Controller) currently do not support HA setups.
- Currently almost no component of a8s supports TLS. The only communication that uses TLS is the one
  between the Kubernetes API server and an a8s [validating webhook][k8s-validating-webhook] that
  performs basic syntactic validation on the PostgreSQL instances API objects.
  Communication between:

  - DSI replicas
  - applications and DSIs
  - DSIs and a8s control plane components
  - a8s control plane components
  - Kubernetes API servers and a8s control plane components

  doesn't use TLS at the moment, with the exception mentioned above. We plan to add that very soon.

## PostgreSQL Instances

- The PostgreSQL server port is hardcoded to `5432`.
- Only PostgreSQL version 13 and 14 are supported.
- Horizontal down scaling is not supported, i.e. you can not scale down the
  number of replicas.
- Each instance stores its data in a dedicated PersistentVolumeClaim of the
  default StorageClass; Currently only this PersistentVolumeClaims size can be configured
- Instance names have to be shorter than 53 characters. That is, their maximum length is 52
  characters. Attempts to create an instance with a longer name will fail immediately.
- Currently a8s doesn't enforce any multi-tenancy/access control regarding the
  Instances it manages. This means that unless you or the Kubernetes cluster
  administrator explicitly set up [RBAC rules][k8s-rbac] and
  [Network Policies][k8s-network-policies] to prevent that, every user of the
  Kubernetes cluster can interact in any possible way with any PostgreSQL
  instance (e.g. provision, deprovision, take backups, create a service binding
  and use it to write/read data to/from the instance, etc...).
- Given an instance, there's no multi-tenancy: all service bindings to
  it will share the same database.
- Instances cannot be used from outside the cluster.
- At the moment there's no way to configure anti-affinity rules to ensure that
  the different replicas of a HA instance run on different Kubernetes
  cluster nodes (or availability zones). This means that it can happen that two
  or more replicas of the same instance end up running on the same
  Kubernetes cluster node.

## Backup and Restore

- Only AWS S3 buckets are supported as backup storage.
- Open connections (idle or active) to the PostgreSQL server during a restore can
  lead to silent failure of the restore. More specifically, the data in the
  backup used for the restore is appended to the data already in the database,
  rather than being used as a replacement for it.
- The backup encryption key is stored in plain text as part of the backup manager
  configuration.
- Point-In-Time-Recovery (PITR) is not supported.
- We currently do not support the creation of a periodic backup schedule.
- The deletion of Recovery API objects can hang indefinitely in some rare cases.
  If you encounter this issue, run the command:

  ```shell
  kubectl patch -n <namespace of the recovery> \
    recoveries.backups.anynines.com/<name of the recovery> \
    -p '{"metadata":{"finalizers":[]}}' --type=merge
  ```

  to force the deletion.

## Service Bindings

- Custom parameters for configuring the permissions are not supported.
- All Service Bindings of a single DSI share the `a9s_apps_default_db` database.
- Service Bindings can only be used in the namespace they are created in, the
  reason behind that is that the secrets, where the password and username are
  stored, are limited to a single namespace (see [Kubernetes Secrets][k8s-secrets])

## Logging

- OpenDashboards has the authentication disabled, this means that the dashboard
  can be accessed from anyone that can reach its URL.
- We currently do not support multiple logging destinations or a separation of
  logs for different users. All logs will be shipped to a single instance
  OpenSearch and are accessible from the dashboard.

## Metrics

- Here we also do not support multi-tenancy, analogous to Logging this implies
  that we do not support multiple destinations or metrics for different users.

[k8s-secrets]:https://kubernetes.io/docs/concepts/configuration/secret/#restrictions
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[k8s-network-policies]: https://kubernetes.io/docs/concepts/services-networking/network-policies/
[k8s-validating-webhook]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook
