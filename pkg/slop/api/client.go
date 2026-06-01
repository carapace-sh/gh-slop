package api

import (
	"io"
	"sync"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

type graphQLDoer interface {
	Do(query string, variables map[string]any, response any) error
}

type restDoer interface {
	Get(path string, resp any) error
	Patch(path string, body io.Reader, resp any) error
}

var (
	graphqlOnce    sync.Once
	graphqlClient  graphQLDoer
	graphqlInitErr error

	restOnce    sync.Once
	restClient  restDoer
	restInitErr error
)

func GraphQLClient() (graphQLDoer, error) {
	graphqlOnce.Do(func() {
		graphqlClient, graphqlInitErr = ghapi.NewGraphQLClient(ghapi.ClientOptions{})
	})
	return graphqlClient, graphqlInitErr
}

func RESTClient() (restDoer, error) {
	restOnce.Do(func() {
		restClient, restInitErr = ghapi.DefaultRESTClient()
	})
	return restClient, restInitErr
}
