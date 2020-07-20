package rebuilder_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"

	. "github.com/errordeveloper/imagine/pkg/rebuilder"
)

func TestRebuilder(t *testing.T) {
	g := NewGomegaWithT(t)

	newImagineRecipe := func(git git.Git) *recipe.ImagineRecipe {
		return &recipe.ImagineRecipe{
			Name: "image-1",
			Scope: &recipe.ImageScopeRootDir{
				BaseDir:                "/go/src/github.com/errordeveloper/imagine",
				RelativeDockerfilePath: "examples/image-1/Dockerfile",
				Git:                    git,
			},
		}
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            true,
			IsDevVal:             false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-wip" suffix`))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            false,
			IsDevVal:             true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-dev" suffix`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            false,
			IsDevVal:             false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{
				DigestValues: map[string]string{
					"reg2.example.org/imagine:16c315": "sha256:test",
				},
			},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding as remote image "reg1.example.com/imagine/image-1:16c315" is not present`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            false,
			IsDevVal:             false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{
				DigestValues: map[string]string{},
			},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding as remote image "reg1.example.com/imagine/image-1:16c315" is not present`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            true,
			IsDevVal:             true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{
				DigestValues: map[string]string{
					"reg2.example.org/imagine/image-1:16c315": "sha256:test",
					"reg1.example.com/imagine/image-1:16c315": "sha256:test",
				},
			},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-dev-wip" suffix`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            false,
			IsDevVal:             false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := &Rebuilder{
			RegistryAPI: &registry.FakeRegistry{
				DigestValues: map[string]string{
					"reg2.example.org/imagine/image-1:16c315": "sha256:test",
					"reg1.example.com/imagine/image-1:16c315": "sha256:test",
				},
			},
		}

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeFalse())
		g.Expect(reason).To(BeEmpty())
	}
}
