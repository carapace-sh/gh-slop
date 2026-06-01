package api

import (
	"io"

	"github.com/cli/go-gh/v2/pkg/api"
)

type GraphQLDoer interface {
	Do(query string, variables map[string]any, response any) error
}

func NewDefaultGraphQLClient() (GraphQLDoer, error) {
	return api.NewGraphQLClient(api.ClientOptions{})
}

type RESTDoer interface {
	Get(path string, resp any) error
	Patch(path string, body io.Reader, resp any) error
}

func NewDefaultRESTClient() (RESTDoer, error) {
	return api.DefaultRESTClient()
}
