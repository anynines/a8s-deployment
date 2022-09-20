# CHANGELOG

All notable changes to the a9s Dataservices on Kubernetes will be documented
here, the format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- max\_locks\_per\_transaction PostgreSQL configuration property has been added
- end-to-end tests on PostgreSQL tolerations to node taints.
- service-binding controller emits events for change of state
- add field `extensions` to  `Postgresql.spec`, that allows installation of supported
  PostgreSQL extensions.
- support for MobilityDB PostgreSQL extension
- backup custom resources now have a `maxRetries` field that specifies how often a backup
  will be retried before entering a failed state

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
