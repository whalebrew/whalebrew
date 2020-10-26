package dockerregistry

// Catalog lists repositories in a registry.
// See https://docs.docker.com/registry/spec/api/#catalog
type Catalog struct {
	Repositories []string `json:"repositories"`
}

func (r *Registry) Catalog() (Catalog, error) {
	c := Catalog{}
	err := r.Get("/v2/_catalog", &c)
	return c, err
}
