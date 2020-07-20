package registry

import (
	"github.com/google/go-containerregistry/pkg/crane"
)

type RegistryAPI interface {
	Digest(string) (string, error)
}

type Registry struct {
}

func (r *Registry) Digest(ref string) (string, error) {
	return crane.Digest(ref)
}
