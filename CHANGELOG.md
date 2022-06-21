# CHANGELOG

All notable changes to the a9s Dataservices on Kubernetes will be documented
here, the format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/). 

## [0.1.0] - 2022-06-22

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