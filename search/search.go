package search

import (
	"fmt"

	"github.com/whalebrew/whalebrew/config"
	"github.com/whalebrew/whalebrew/dockerregistry"
)

// ErrorHandler handles the logic when an error occurs and returns whether to continue or stop
type ErrorHandler func(error) (abort bool)

// Searcher searches a registry for images matching a term
type Searcher interface {
	Search(term string, errorHandler ErrorHandler) <-chan string
}

// ForRegistries initialises searchers from whalebrew configuration
func ForRegistries(registries []config.Registry, handleError ErrorHandler) <-chan Searcher {
	if len(registries) == 0 {
		registries = []config.Registry{
			{
				DockerHub: &config.DockerHubRegistry{
					Owner: "whalebrew",
				},
			},
		}
	}
	out := make(chan Searcher)
	if handleError == nil {
		handleError = defaultErrorHandler
	}
	go func() {
		for idx, registry := range registries {
			if registry.DockerHub != nil {
				out <- &DockerHub{Owner: registry.DockerHub.Owner}
				continue
			} else if registry.DockerRegistry != nil {
				out <- &DockerRegistry{
					Owner: registry.DockerRegistry.Owner,
					Registry: &dockerregistry.Registry{
						Host:    registry.DockerRegistry.Host,
						UseHTTP: registry.DockerRegistry.UseHTTP,
					},
				}
				continue
			} else {
				if handleError(fmt.Errorf("unsupported configuration at index %d: %v", idx, registry)) {
					close(out)
					return
				}
			}
		}
		close(out)
	}()
	return out
}

func defaultErrorHandler(err error) (abort bool) {
	fmt.Println(err.Error())
	return true
}
