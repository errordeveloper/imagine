package registry

import (
	"fmt"
)

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
