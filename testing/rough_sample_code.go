package test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// config.yaml provided by the user
//```yaml
//TestEnvironments:
//  - TestEnvironment1:
//      shareDSIPerCluster: false # If true then sequential
//      processes: 4 # Number of go tests processors to run tests in parallel. 1 Means sequencial
//      kubeConfig: "~/.kube/config" # Add one or many kubeconfigs representing each cluster
//      dataservice: "PostgreSQL:MongoDB" # How to specify version?
//      failFast: true # When a test fails stop running the rest of the tests
//      cleanup: onFailure # onFailure/always/never
//```

// Testing framework
func main() {
	// We also need to parse and provide the config.yml to set options for the TestEnvironment
	testEnv := NewTestEnvironment(SetShareDataservice(false)) // And other options not mentioned here.
	for i, spec := range(testEnv.Specs()) {
		clients := testEnv.Clients()
		go func(runSpec Spec, client TestClient, needsCleanup bool) {
			runSpec(client, client, needsCleanup)
		}(spec, clients[i % len(clients)], true)
	}
}

// TestEnvironment - We need a better name than even this.
// TestEnvironment? How do we avoid making this to Kubernetes specific
type TestEnvironment interface {
	// Methods to expose
	Specs() []Spec
	Clients() []TestClient
}

type DataserviceType string

const (
	PostgreSQL DataserviceType = "PostgreSQL"
	MongoDB DataserviceType = "MongoDB"
	Redis DataserviceType = "Redis"
)

// TestEnvironment
// Kubernetes infrastructure config
// TestConfig
// We need a better name. Its not a KuberneteEnv
type testEnvironment struct { // TestingEnv?
	clients []TestClient

	// TestConfig Struct
	Nodes int
	ShareDataservice bool
	DataserviceTypes []DataserviceType
	Kubeconfigs []string
	TestCategories []string // Default all but option to select individual tests or subset of tests
	specs []Spec
}

func NewTestEnvironment(options ...ConfigOption) TestEnvironment { //return TestEnvironment
	// Provide defaults
	// Apply all options to the object.
	return &testEnvironment{}
}

type ConfigOption func(testEnvironment)

func SetShareDataservice(share bool) ConfigOption {
	return func(env testEnvironment) {
		env.ShareDataservice = share
	}
}

func SetNodes(nodes int) ConfigOption {
	return func(env testEnvironment) {
		env.Nodes = nodes
	}
}

// Clients
// KubernetesClients? But this could be namespaces? But we need to speak with Kubernetes clients?
// TestClients?
// Clients
// Clients?
// TestClient?
// We could use different packages to make this clear
// This is not needed given we can use runtimeClient.Client
// Express how this would wor
type TestClient interface {
	// Expose private fields of instances of TestClient
	Namespace() string
	Client() runtimeClient.Client
}

// This needs a better name.
// How do we differentiate4 a naked client from a managed client. They need better and more
// desciptive names? On what basis do we differentiate?
// We could use a single client
// DSIClient or CustomResourceClient or DataserviceClient
type DataserviceToDSIClient map[string]*runtimeClient.Client

// How we communicate with each cluster's kubernetes API
// We need ClientSet for interacting with standard Kubernetetes APIs in a direct cacheless way.
// We need the managerClient to conveniently perform operations on the various CR that represent
// our DSIs.
type testClient struct {
	Client *kubernetes.Clientset // Client for accessing standard Kubernetes APIs.
	//clientSet *runtimeClient.Client // Manager client
	Dataservices []Dataservice // Only if needed
	DataserviceToDSIClient // For handling many ManagerClients for each DSI
	// Kubeconfig
}

func NewClients(...ClientsOption) TestClient {
	// Apply ClientsOptions before return object
	return &testClient{}
}

type Kubeconfig string

type ClientsOption func(TestClient)

func WithKubeConfig(...Kubeconfig) ClientsOption {
	return func(TestClient) {
		// Create clients
	}
}

// Dataservice
type Dataservice interface {
	Client() DatabaseClient
}

// Rename to dataserviceInstance
type dataservice struct {
	dbClient DatabaseClient
	PortForwardChan chan struct{} // Used for closing portforward when cleaning up DSI
	obj *runtimeClient.Object
}

func (dsi *dataservice) Client() DatabaseClient {
	return dsi.dbClient
}

func NewDataservice() Dataservice {
	// We need to always create a DatabaseClient for each dataservice to be used for
	// accessing the database.
	return &dataservice{}
}

// DatabaseClient
// Pick a better name. Like remove tester.
// Comment on what this and other interfaces are.
type DatabaseClient interface {
	DatabaseIsUp(ctx context.Context) error
	// We still need to determine the payload mechanism so we can control data being inputted
	// in a general way
	InsertData(ctx context.Context) error // Payload input needs to be thought of
	GetData(ctx context.Context) error
	// And more
}


// SpecOptions for expressing and managing Tests to be interated upon in main loop.
const asyncOpsTimeoutMins = time.Minute * 5

type Spec func(ctx context.Context, client TestClient)

func TestInsertion(ctx context.Context, client TestClient) Spec {
	// Some assertions may need to be eventuallys.
	return func(ctx context.Context, client TestClient) {
		// Conditional on shareDataservice
		customResourceClient := client.CustomResourceClient()
		dsi := NewDataservice()
		customResourceClient.Create(ctx, dsi)
		Eventually(func () error {
			return dsi.Client().DatabaseIsUp(ctx)
		}, asyncOpsTimeoutMins, ).Should(Equal(BeNil()))
		Expect(dsi.Client().InsertData(ctx)).To(Succeed())
		Expect(dsi.Client().GetData(ctx)).To(Succeed())
		// We need to a record of actions taken against the database so that we can
		// unwind those actions in order to remove all side effects. This will enable us to
		// safely shareDataservice across tests.
		// Conditional on whether sharedDataservice
		customResourceClient.Delete(ctx, dsi)
	}
}

// Convenience function for grouping SpecFuncs
func BasicTests(ctx context.Context, dbClient DatabaseClient, dsiClient runtimeClient.Client, dsi runtimeClient.Object) []Spec {
	return []Spec{TestInsertion(ctx, dbClient, dsiClient, dsi)}
}
