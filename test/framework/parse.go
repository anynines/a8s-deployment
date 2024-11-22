package framework

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	k8serrors "k8s.io/apimachinery/pkg/util/errors"
)

const (
	testingNamespacePrefix = "a8s-e2e-tests"
	suffixLength           = 5
	defaultVersion         = 16
)

type TestRunConfig struct {
	// KubeconfigPath is the path to the kube config to be used by the Kubernetes client
	KubeconfigPath string
	// Dataservice is the dataservice the tests are to be performed on
	Dataservice string
	// DSINamePrefix provides a name that the DSI object will take in the cluster
	DSINamePrefix string
	// Namespace provides the target namespace to be used for testing. If not given then a
	// unique namespace is created.
	Namespace string
	// Dataservice version to test
	Version int
}

// TODO: Use marshalling approach to provide more fine grained feedback on missing environment
// variables.
func ParseEnv() (TestRunConfig, error) {
	config := TestRunConfig{
		KubeconfigPath: os.Getenv("KUBECONFIGPATH"),
		Dataservice:    os.Getenv("DATASERVICE"),
		DSINamePrefix:  os.Getenv("DSI_NAME_PREFIX"),
		Namespace:      os.Getenv("NAMESPACE"),
	}
	// Use dynmically generated name for Namespace if none is provided.
	if config.Namespace == "" {
		config.Namespace = UniqueName(testingNamespacePrefix, suffixLength)
	}

	version, err := getIntEnv("VERSION", defaultVersion)
	if err != nil {
		return TestRunConfig{}, err
	}
	config.Version = version
	return config, validateConfig(config)
}

func getIntEnv(key string, fallback int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil // Environment variable not set, use the fallback
	}
	if value == "" {
		return fallback, fmt.Errorf("%s is set but empty", key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", key, value)
	}
	return intValue, nil
}

func validateConfig(c TestRunConfig) error {
	errs := make([]error, 0, 3)
	if c.DSINamePrefix == "" {
		errs = append(errs, errors.New("DSI_NAME_PREFIX env var is not set and MUST be set"))
	}
	if c.KubeconfigPath == "" {
		errs = append(errs, errors.New("KUBECONFIGPATH env var is not set and MUST be set"))
	}
	if c.Dataservice == "" {
		errs = append(errs, errors.New("DATASERVICE env var is not set and MUST be set"))
	}
	return k8serrors.NewAggregate(errs)
}

func ConfigToVars(c TestRunConfig) (kubeconfigPath, dsiNamePrefix, dataservice, namespace string, version int) {
	return c.KubeconfigPath, c.DSINamePrefix, c.Dataservice, c.Namespace, c.Version
}
