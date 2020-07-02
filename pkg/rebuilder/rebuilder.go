package rebuilder

import (
	"fmt"
	"strings"

	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"
)

type Rebuilder struct {
	RegistryAPI registry.RegistryAPI
}

func (r *Rebuilder) ShouldRebuild(manifest *recipe.BakeManifest) (bool, string, error) {
	for _, ref := range manifest.RegistryTags() {
		for _, suffix := range []string{"-dev-wip", "-dev", "-wip"} {
			if strings.HasSuffix(ref, suffix) {
				return true, fmt.Sprintf("rebuilding due to %q suffix", suffix), nil
			}
		}

		if _, err := r.RegistryAPI.Digest(ref); err != nil {
			// TODO: check the error is actually a 404, otherwise if it's to do with auth or network - fail early
			return true, fmt.Sprintf("rebuilding as remote image %q is not present", ref), nil
			break
		}
	}

	return false, "", nil
}
