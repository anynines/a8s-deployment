# CHANGELOG

All notable changes to the a9s Dataservices on Kubernetes will be documented
here, the format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/).

## [unreleased]

### Updated

* Bump Operator-SDK version to v1.34.2 for postgresql-operator

## [1.1.0] - 2024-06-05

### Added

* Added support for installing/uninstalling PostgreSQL extensions via the extensions field,
  including postgis, uuid-ossp, ltree, pgcrypto, pg_stat_statements, fuzzystrmatch, pg_trgm, and
  hstore.

### Fixed

* Fix PostgreSQL restart logic after user parameter updates.

### Updated

* The backup_agent has been bumped to the latest version and now includes buildx support.

## [1.0.0] - 2024-03-15

### Updated

* Control plane images are now compatible with both ARM and AMD architectures.
* Updated cert-manager to v1.12.0.
* Documentation has been revised, removing outdated instructions.

## [0.3.0] - 2023-05-15
### Migration Instructions
* If you use the extensions feature, delete the stateful set objects (**not the PostgreSQL objects**)
for instances that use extensions. Your data will be preserved, and the stateful sets will
automatically be recreated.
* Remove old finalizer `postgresql.operator.a8s.anynines.com` from all PostgreSQL objects.
* Migrate all `v1alpha1` PostgreSQL custom resources to `v1beta3` before migrating to v0.3.0, as PostgreSQL
version `v1alpha1` will not be available in v0.3.0.

### Added

* Protect PostgreSQL secrets against accidental or unwanted deletion by adding a finalizer.
* Protect ServiceBinding secret against accidental or unwanted deletion by adding a finalizer.
* Add optional read-only service to the PostgreSQL-Operator. The read-only service can be used
  to distribute the load of read operations across the PostgreSQL instance. It can be enabled via
  the optional `enableReadOnlyService` field on the Custom Resource.
* Add field `spec.resources.claims` to the PostgreSQL CRD to allow configuring
  dynamic resources for the PostgreSQL instance Pods. This feature is in alpha,
  is implemented only on K8s clusters with version 1.26 or higher and the
  feature gate DynamicResourceAllocation enabled.
* Add field `spec.expose` to expose a PostgreSQL instance to outside the K8s cluster where it runs.

### Changed

* Updated description of `NamespacedName` in servicebinding.
* Deletion of PostgreSQL instance pods now runs in parallel, improving the deletion time in high
  availability setups.
* In the Postgresql CRD, the description of the `namespaceSelector` field (which is one of the many
  fields that control pod affinity and anti-affinity) has been updated to reflect the fact that the
  field has graduated from beta to stable (in Kubernetes v1.24).
* **Breaking change:** PostgreSQL CRD has been updated with the following:
  * `spec.parameters.maxConnections` has a minimum value of 1 and a maximum value
    of 262143 enforced.
  * `spec.parameters.maxReplicationSlots` has a minimum value of 0 and a maximum
    value of 262143 enforced.
  * `spec.parameters.maxWALSenders` has a minimum value of 0 and a maximum value
    of 262143 enforced.
  * `spec.parameters.sharedBuffers` has a minimum value of 16 and a maximum value
    of 1073741823 enforced.
  * `spec.parameters.statementTimeoutMillis` has a minimum value of 0 and a
    maximum value of 2147483647 enforced.
  > Attempts to create a PostgreSQL API object with one or more values not
    compliant with the min and max listed above will be rejected with an error by
    the K8s API server.
* **Breaking change**: Postgresql-controllers finalizer has been updated from 
  `postgresql.operator.a8s.anynines.com` to `a8s.anynines.com/postgresql.operator`.
* **Breaking change**: Postgresql-operator now uses an emptyDir instead of a persistent volume.
* **Breaking change**: The field `postgresConfiguration` has been renamed to
  `parameters` in API Version `v1beta3`.
* Make the secrets that store the credentials of the admin and replication roles
  of each PostgreSQL instance immutable. Preventing changes to credentials
  protects against accidental (or unwanted) updates that could cause service
  outages and improves performance by significantly reducing load on
  kube-apiserver for clusters that make extensive use of secrets.
* Make service binding secrets immutable. Preventing changes to credentials
  protects against accidental (or unwanted) updates that could cause service
  outages and improves performance by significantly reducing load on
  kube-apiserver for clusters that make extensive use of secrets.
* The backup\_agent has been updated, and is now using a smaller image. The new version of the
  backup\_agent logs to stderr instead of the previously used stdout.
* Defaulting webhooks have been moved to API version `v1beta3`.

### Removed

* **Breaking change:** PostgreSQL version `v1alpha1` has been removed and is replaced by version 
`v1beta3`.

## [0.2.0] - 2022-11-08

### Added

- Publish API version v1beta3
- max\_locks\_per\_transaction PostgreSQL configuration property has been added
- end-to-end tests on PostgreSQL tolerations to node taints.
- service-binding controller emits events for change of state
- add field `extensions` to  `Postgresql.spec`, that allows installation of supported
  PostgreSQL extensions.
- support for MobilityDB PostgreSQL extension
- backup custom resources now have a `maxRetries` field that specifies how often a backup
  will be retried before entering a failed state
- Add chaos test for crashing backup agent
- Add chaos test for ensuring interrupted backup data is cleaned up from S3

### Updated

- Due to some internal code clean-up some error messages have changed in the logs
- Rename tests from "integration tests" to "end-to-end tests", as they are end-to-end tests
- Upgrade ginkgo from v1 to v2 in the end-to-end tests
- Upgrade version of PostgreSQL-Operator to v0.39.0

### Fixed

- **breaking change** Fix bug that caused the event for the successful deletion of
  a DSI to be emitted multiple times and before the deletion had actually
  completed successfully
- **breaking change** Fix issue where only a single event was emitted for two secrets
  of a PostgreSQL instance
- Apply fix to PostgreSQL-Operator end-to-end tests to reduce flakiness
- backup manager now handles crashes of the backup agent gracefully by restarting the failed backup

### Changed

- backup-manager now uses a dedicated ServiceAccount, instead of the default one
- service-binding controller now uses a dedicated ServiceAccount, instead of the default one
- postgresql-operator now used a dedicated ServiceAccount, instead of the default one
- **breaking change** backup custom resources now use a list of Conditions instead of a single enum
as the status
- **breaking change** `Recovery` objects have been renamed to `Restore`.
  The new version of the operator does no longer watch for objects of the `Recovery` type. Do not
  upgrade while a `Recovery` object is in progress.

## [0.1.0] - 2022-06-27

### Added

- Changelog has been added
- Support for PostgreSQL 14 has been added
- Instructions on how to update the framework have been added
- Installation with the help of OLM bundles is supported
- Validation webhooks are introduced, with the cert-manager to obtain
  certificates for them
- Volume size of the PersistentVolume of a PostgreSQL instance is configurable

### Fixed

- Backup encryption password has been moved to a secret
- Memory and CPU resource requests on operator pods have been increased to
  prevent issues on ARM

### Changed

- The PostgreSQL operator has been moved to the `a8s-system` namespace
- Backup agent image and Spilo image are no longer configurable by the
  Application Developer, but rather are set by the Platform Operator
- Fluentbit is updated to v1.9.4
- Fluentd is updated to v1.14.6-1.1, additionally the OpenSearch plugin has been
  added and the deprecated ES plugin was removed
- Update OpenSearch, OpenSearchDashboards to v2.0.0
- Prometheus v2.32.1
- Grafana 8.3.3, including updates to the documentation
