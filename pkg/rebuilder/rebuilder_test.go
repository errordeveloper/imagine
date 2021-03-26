package rebuilder_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"

	. "github.com/errordeveloper/imagine/pkg/rebuilder"
)

func TestRebuilder(t *testing.T) {
	g := NewGomegaWithT(t)

	newImagineRecipe := func(repo *git.FakeRepo) *recipe.ImagineRecipe {
		dir := ""
		ir := &recipe.ImagineRecipe{
			WorkDir: "/go/src/github.com/errordeveloper/imagine",
			BuildSpec: &config.BuildSpec{
				Name: "image-1",
				WithBuildInstructions: &config.WithBuildInstructions{
					Dir: &dir,
					Dockerfile: &config.DockerfileBuildInstructions{
						Path: "examples/image-1/Dockerfile",
					},
				},
			},
		}

		ir.Config.Path = "dummy.yaml"

		repo.TreeHashForHeadRoot = "16c315243fd31c00b80c188123099501ae2ccf91"
		repo.TreeHashForHeadVal = map[string]string{
			"dummy.yaml": "613919533ebd03d6bafbd538ccad3a4acea9b761",
		}
		repo.IsWIPVal = map[string]bool{
			"dummy.yaml": false,
		}

		ir.Git.Git = repo
		ir.Git.BranchedOffSuffix = "-dev"
		ir.Git.WorkInProgressSuffix = "-wip"

		g.Expect(ir.BuildSpec.ApplyDefaultsAndValidate()).To(Succeed())

		return ir
	}

	newRebuilder := func(present ...string) *Rebuilder {
		fakeRegistry := &registry.FakeRegistry{
			DigestValues: map[string]string{},
		}

		for _, image := range present {
			fakeRegistry.DigestValues[image] = "sha256:test"
		}

		return &Rebuilder{
			RegistryAPI:          fakeRegistry,
			BranchedOffSuffix:    "-dev",
			WorkInProgressSuffix: "-wip",
		}
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: true,
			IsDevVal:  false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder()

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-wip" suffix`))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: false,
			IsDevVal:  true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder()

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-dev" suffix`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: false,
			IsDevVal:  false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder(
			"reg2.example.org/imagine:613919.16c315",
		)

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding as remote image "reg1.example.com/imagine/image-1:613919.16c315" is not present`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: false,
			IsDevVal:  false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder()

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding as remote image "reg1.example.com/imagine/image-1:613919.16c315" is not present`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: true,
			IsDevVal:  true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder(
			"reg2.example.org/imagine/image-1:613919.16c315",
			"reg1.example.com/imagine/image-1:613919.16c315",
		)

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeTrue())
		g.Expect(reason).To(Equal(`rebuilding due to "-dev-wip" suffix`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: false,
			IsDevVal:  false,
		})

		m, err := ir.ToBakeManifest("reg3.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		rb := newRebuilder(
			"reg2.example.org/imagine/image-1:613919.16c315",
			"reg3.example.com/imagine/image-1:613919.16c315",
		)

		rebuild, reason, err := rb.ShouldRebuild(m)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(rebuild).To(BeFalse())
		g.Expect(reason).To(BeEmpty())
	}
}
