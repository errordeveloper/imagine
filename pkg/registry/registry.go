package registry

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	_ "github.com/google/go-containerregistry/pkg/crane"
)

type RegistryAPI interface {
	Digest(string) (string, error)
}

type Registry struct {
}

func (r *Registry) Digest(ref string) (string, error) {
	return crane.Digest(ref)
}

type FakeRegistry struct {
	DigestValues map[string]string
}

func (f *FakeRegistry) Digest(ref string) (string, error) {
	v, ok := f.DigestValues[ref]
	if !ok {
		return "", fmt.Errorf("%s is not in fake registry", ref)
	}
	return v, nil
}
