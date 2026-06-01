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
	graphQLOnce     sync.Once
	cachedGraphQL   graphQLDoer
	graphQLInitErr  error

	restOnce       sync.Once
	cachedREST     restDoer
	restInitErr    error
)

func graphQLClient() (graphQLDoer, error) {
	graphQLOnce.Do(func() {
		cachedGraphQL, graphQLInitErr = ghapi.NewGraphQLClient(ghapi.ClientOptions{})
	})
	return cachedGraphQL, graphQLInitErr
}

func restClient() (restDoer, error) {
	restOnce.Do(func() {
		cachedREST, restInitErr = ghapi.DefaultRESTClient()
	})
	return cachedREST, restInitErr
}
