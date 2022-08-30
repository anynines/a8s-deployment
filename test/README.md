# a8s End-to-End Tests

## Prerequisites

- Ensure you have completed the
  [Platform Operator Documentation][Platform Operator Documentation]
  instructions as the testing framework assumes that the a8s Control Plane is
  deployed.
- The testing framework is configured using environmental variables. Please
  ensure that the following environmental variables are set:

  - `NAMESPACE`: The target namespace for deploying test objects to. *If not
    provided a unique namespace will be generated*
  - `KUBECONFIGPATH`: The kubeconfig corresponding to the cluster in which
    tests should be run against.
  - `DSI_NAME_PREFIX`: Provides name for the DSI and auxiliary resources
    required for running tests. A unique suffix will be provided for each
    resource to avoid conflict when running tests in parallel.
  - `DATASERVICE`: Provides the data service type tests will be run against.
     Currently supported dataservices:

    - PostgreSQL

## How to use

### Running the Tests

Tests are organized in go packages, each package holds one test suite whose
test cases test the same coarse-grained functionality.

- To run *all* the test suites currently available run `go test ./...` from
  inside the test directory.
- To run the end-to-end tests use `go test ./e2e/..`
- To run only the chaos tests use `go test ./chaos-tests`
- To run a *single* suite/piece of functionality, for example the backup
  end-to-end tests, run `go test ./e2e/backup` from inside the test directory.
- `go test` can also be replaced by `ginkgo` for more informative output.

If your run includes the `chaos-tests`, you will have to install
[ChaosMesh](https://chaos-mesh.org/). As the installation is specific to the
container runtime used in your cluster, refer to the [official installation
guide](https://chaos-mesh.org/docs/production-installation-using-helm/). 

### Adding or Modifying Tests

- To add tests that test the end-to-end (e2e) behavior of a8s,
  create a package under [e2e/][e2e package]. This package will
  import from the package [framework/][Framework package] which provides helper
  functionality in order to simplify the process of writing new tests and help
  make tests for different components more consistent. The framework packages
  can be extended to provide more features but you should try not to break
  existing tests where possible.
- The framework consists of functionality for creating new Kubernetes resources
  for our custom resource definitions included in the a8s Control Plane. It
  includes factory design patterns for generalizing the creation of new data
  service instances and their associated clients to open up connections for
  data manipulation. It also provides helper utilities such as access to the
  database from outside the cluster via port forwards and logic to parse
  environmental variable configuration.
- Tests for each framework components will exist inside packages at the same
  level as [framework/][Framework package]. For example the
  [backup][Backup package] package includes tests for testing backup and
  restore functionality of the [a8s-backup-manager][a8s-backup-manager] against
  supported data service types.

## Directory structure

```text
.
├── README.md
├── go.mod
├── go.sum
└── chaos-tests
│   ├── postgresql_chaos_suite_test.go
│   └── postgresql_chaos_test.go
├── e2e
│   ├── backup
│   │   ├── backup_suite_test.go
│   │   └── backup_test.go
│   ├── patroni
│   │   ├── patroni_suite_test.go
│   │   └── patroni_test.go
│   ├── postgresql
│   │   ├── postgresql_suite_test.go
│   │   └── postgresql_test.go
│   ├── servicebinding
│   │   ├── servicebinding_suite_test.go
│   │   └── servicebinding_test.go
│   ├── topology_awareness
│       ├── topology_awareness_suite_test.go
│       └── topology_awareness_test.go
└── framework
      ├── backup
      │   └── backup.go
      ├── chaos
      │   └── chaos.go
      ├── dsi
      │   ├── client.go
      │   ├── dsi.go
      │   └── dsiclient.go
      ├── parse.go
      ├── portforward.go
      ├── postgresql
      │   ├── dsiclient.go
      │   └── postgresql.go
      ├── restore
      │   └── restore.go
      ├── secret
      │   └── secret.go
      ├── servicebinding
      │   └── servicebinding.go
      └── util.go
    
```

Note: for packages that contain end-to-end test suites, only the test files are shown above. There
might be other files containing helper functions, etc... that aren't shown.

[a8s-backup-manager]: https://github.com/anynines/a8s-backup-manager/
[Platform Operator Documentation]: ../docs/platform-operators/installing_framework.md
[Framework package]: e2e/framework/
[Backup package]: e2e/backup
[e2e package]: e2e
