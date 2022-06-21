package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// DockerHub implements the Searcher interface and searches for images with a specific owner
type DockerHub struct {
	Owner string
}

type imageResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
type searchAnswer struct {
	Results []imageResult `json:"results"`
}

func (dh *DockerHub) Search(term string, handleError ErrorHandler) <-chan string {
	out := make(chan string)
	if handleError == nil {
		handleError = defaultErrorHandler
	}
	go func() {
		params := url.Values{}
		params.Set("page_size", "100")
		params.Set("ordering", "last_updated")
		params.Set("name", term)
		u := url.URL{
			Scheme:   "https",
			Host:     "hub.docker.com",
			Path:     fmt.Sprintf("/v2/repositories/%s/", dh.Owner),
			RawQuery: params.Encode(),
		}
		r, err := http.Get(u.String())
		if err != nil {
			if handleError(err) {
				close(out)
				return
			}
		}
		answer := searchAnswer{}
		err = json.NewDecoder(r.Body).Decode(&answer)
		if err != nil {
			if handleError(err) {
				close(out)
				return
			}
		}
		for _, image := range answer.Results {
			out <- fmt.Sprintf("%s/%s", image.Namespace, image.Name)
		}
		close(out)
	}()
	return out
}
