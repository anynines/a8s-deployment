package testing

import (
	"time"
	//. "github.com/onsi/ginkgo"
	"context"

	"github.com/docker/docker/testutil/environment"
	. "github.com/onsi/gomega"
	. "github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Testing framework
func main() {
	// We also need to parse and provide the config.yml to set options of the Infrastructure
	// We need to be to be thread safe with sync.Mutex
	infraEnv := NewInfrastructure(SetShareDataservice(false)) // And other options not mentioned here.
	for i, spec := range(infraEnv.Specs()) {
		clusters := infraEnv.Clusters()
		// We may need to branch to run sequentially if `ShareDataServiceIsOff` or
		// processes=1.
		go func(spec SpecFunc, cluster Cluster) {
			// Create dataservice using cluster.Client
			// Call spec with spec(dataservice.DatabaseClientTester)
			// Cleanup if successful or leave up depending on configuration
		}(spec, clusters[i % len(clusters)])
	}
}

// Infra
type Infrastructure interface {
	// Methods to expose
	Specs() []SpecFunc
	Clusters() []Cluster
}

type DeploymentComponent string

const (
	Core DeploymentComponent = "core"
	All DeploymentComponent = "all"
	Logging DeploymentComponent = "logging"
	Metrics DeploymentComponent = "metrics"
)

type DataserviceType string

const (
	PostgreSQL DataserviceType = "PostgreSQL"
	MongoDB DataserviceType = "MongoDB"
	Redis DataserviceType = "Redis"
)

type KubernetesEnv struct { // TestingEnv
	clusters []Cluster
	Nodes int
	ShareDataservice bool
	DeploymentComponents []DeploymentComponent
	DataserviceTypes []DataserviceType
	Kubeconfigs []string
	TestCategories []string // Default all but option to select individual tests or subset of tests
	specs []SpecFunc
}


func NewInfrastructure(options ...ConfigOption) Infrastructure { //return Infrastructure
	return &KubernetesEnv{}
}

type ConfigOption func(KubernetesEnv)

func SetShareDataservice(share bool) ConfigOption {
	return func(env KubernetesEnv) {
		env.ShareDataservice = share
	}
}

func SetNodes(nodes int) ConfigOption {
	return func(env KubernetesEnv) {
		env.Nodes = nodes
	}
}

// Cluster
type Cluster interface {
}

type DataserviceToManagerClientMap map[string]*runtimeClient.Client

type cluster struct {
	Client *kubernetes.Clientset // We may want a k8sclient too
	clientSet *runtimeClient.Client // Manager client
	Dataservices []Dataservice
	DataserviceToManagerClientMap // For handling many ManagerClients for each DSI
	// Kubeconfig
}

func NewCluster(...ClusterOption) Cluster {
	// Apply ClusterOptions before return object
	return &cluster{}
}

type Kubeconfig string

type ClusterOption func(Cluster)

func WithKubeConfig(...Kubeconfig) ClusterOption {
	return func(Cluster) {
		// Create clients
	}
}

// The cluster will hold ManagerClients for setuping DSI CRs for each spec.
type ManagerClient interface {
	Create(ctx context.Context, obj runtimeClient.Object) error
	Delete(ctx context.Context, obj runtimeClient.Object) error
	Update(ctx context.Context, obj runtimeClient.Object) error
}

// Dataservice
type Dataservice interface {
	Client() DatabaseClientTester
}

type dataservice struct {
	NamespacedName types.NamespacedName
	dbClient DatabaseClientTester
	PortForwardChan chan struct{} // Used for closing portforward when cleaning up DSI
	obj *runtimeClient.Object
}

func (dsi *dataservice) Client() DatabaseClientTester {
	return dsi.dbClient
}

func NewDataservice() Dataservice {
	// We need to always create a DatabaseClientTester for each dataservice to be used for
	// accessing the database.
	return &dataservice{}
}

// DatabaseClientTester
type DatabaseClientTester interface {
	DatabaseIsUp(ctx context.Context) error
	// We still need to determine the payload mechanism so we can control data being inputted
	// in a general way
	InsertData(ctx context.Context) error
	GetData(ctx context.Context) error
}

// SpecOptions for expressing and managing Tests to be interated upon in main loop.
const asyncOpsTimeoutMins = time.Minute * 5
type SpecFunc func()
func TestData(ctx context.Context, dbClient DatabaseClientTester) SpecFunc {
	// Some assertions may need to be eventuallys.
	return func() {
		Eventually(func () error {
			return dbClient.DatabaseIsUp(ctx)
		}, asyncOpsTimeoutMins, ).Should(Equal(BeNil()))
		Expect(dbClient.InsertData(ctx)).To(Succeed())
		Expect(dbClient.GetData(ctx)).To(Succeed())
	}
}

// Convenience function for grouping SpecFuncs
func BasicTests(ctx context.Context, dbClient DatabaseClientTester) []SpecFunc {
	return []SpecFunc{TestData(ctx, dbClient)}
}

