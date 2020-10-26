package search

import (
	"strings"

	"github.com/whalebrew/whalebrew/dockerregistry"
)

// DockerRegistry implements the Searcher interface searching over a docekr registry
type DockerRegistry struct {
	Owner    string
	Registry Cataloger
}

type Cataloger interface {
	Catalog() (dockerregistry.Catalog, error)
	ImageName(path string) string
}

func (dr *DockerRegistry) Search(term string, handleError ErrorHandler) <-chan string {
	out := make(chan string)
	if handleError == nil {
		handleError = defaultErrorHandler
	}
	go func() {
		catalog, err := dr.Registry.Catalog()
		if err != nil && handleError(err) {
			close(out)
			return
		}
		for _, repo := range catalog.Repositories {
			if strings.HasPrefix(repo, dr.Owner+"/") {
				if strings.Contains(strings.TrimPrefix(repo, dr.Owner+"/"), term) {
					out <- dr.Registry.ImageName(repo)
				}
			}
		}
		close(out)
	}()
	return out
}
