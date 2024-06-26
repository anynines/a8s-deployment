package dsi

import (
	"context"
	"fmt"
	"strings"

	"github.com/anynines/a8s-deployment/test/framework/postgresql"
)

// TODO: Create implementations for Data interface to generalize test data input
type DSIClient interface {
	DSIDeleter
	DSIReader
	DSIWriter
	DSIAccountValidator
	DSICollectionValidator
	DSIConfigurationValidator
}

type DSIReader interface {
	Read(ctx context.Context, entity string) (string, error)
}

type DSIWriter interface {
	Write(ctx context.Context, entity, data string) error
}

type DSIDeleter interface {
	Delete(ctx context.Context, entity, data string) error
}

type DSIAccountValidator interface {
	UserExists(ctx context.Context, username string) (bool, error)
}

type DSICollectionValidator interface {
	CollectionExists(ctx context.Context, collection string) bool
}

type DSIConfigurationValidator interface {
	CheckParameter(ctx context.Context, parameter, value string) error
}

func NewClient(ds, port string, sbData map[string]string) (DSIClient, error) {
	switch strings.ToLower(ds) {
	case "postgresql":
		return postgresql.NewClientOverPortForwarding(sbData, port), nil
	}
	return nil, fmt.Errorf(
		"dsi client factory received request to create dsi client for unknown data service %s; only supported data services are %s",
		ds,
		supportedDataServices(),
	)
}

func NewClientForURL(ds, host, port, sslmode string, sbData map[string]string) (DSIClient, error) {
	switch strings.ToLower(ds) {
	case "postgresql":
		return postgresql.NewClient(sbData, host, port, sslmode), nil
	}
	return nil, fmt.Errorf(
		"dsi client factory received request to create dsi client for unknown data service %s; only supported data services are %s",
		ds,
		supportedDataServices(),
	)
}
