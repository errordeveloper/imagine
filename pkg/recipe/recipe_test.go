package recipe_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/errordeveloper/imagine/pkg/git"
	. "github.com/errordeveloper/imagine/pkg/recipe"
)

func TestWithRootDirScope(t *testing.T) {
	g := NewGomegaWithT(t)

	ir := &ImagineRecipe{
		Name: "image-1",
		Scope: &ImageScopeRootDir{
			RootDir:                "/go/src/github.com/errordeveloper/imagine",
			RelativeDockerfilePath: "examples/image-1/Dockerfile",
			Git: &git.FakeRepo{
				CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			},
		},
	}

	g.Expect(ir.Scope.ContextPath()).To(Equal("/go/src/github.com/errordeveloper/imagine"))

	g.Expect(ir.Scope.DockerfilePath()).To(Equal("/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile"))

	{
		ir.HasTests = false

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(HaveLen(0))
	}

	{
		ir.HasTests = false

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315",
			"reg2.example.org/imagine/image-1:16c315",
		))
	}

	{
		ir.HasTests = true
		ir.Platforms = []string{"linux/amd64", "linux/arm64"}

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1-test", "image-1"))

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target).To(HaveKey("image-1-test"))

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315",
			"reg2.example.org/imagine/image-1:16c315",
		))
		g.Expect(m.Target["image-1-test"].Tags).To(HaveLen(0))

		g.Expect(m.Target["image-1"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))
		g.Expect(m.Target["image-1-test"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))

		js, err := ir.ToBakeManifestAsJSON("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		expected := `
		  {
			"group": {
			  "default": {
				"targets": [
				  "image-1-test",
				  "image-1"
				]
			  }
			},
			"target": {
			  "image-1": {
				"context": "/go/src/github.com/errordeveloper/imagine",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"tags": [
				  "reg1.example.com/imagine/image-1:16c315",
				  "reg2.example.org/imagine/image-1:16c315"
				],
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				]
			  },
			  "image-1-test": {
				"context": "/go/src/github.com/errordeveloper/imagine",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"target": "test",
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				]
			  }
			}
		  }
		`
		g.Expect(js).To(MatchJSON(expected))
	}
}

func TestWithRootDirScopeGit(t *testing.T) {
	g := NewGomegaWithT(t)

	newImagineRecipe := func(git git.Git) *ImagineRecipe {
		return &ImagineRecipe{
			Name: "image-1",
			Scope: &ImageScopeRootDir{
				RootDir:                "/go/src/github.com/errordeveloper/imagine",
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

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315-wip",
			"reg2.example.org/imagine/image-1:16c315-wip",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            false,
			IsDevVal:             true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315-dev",
			"reg2.example.org/imagine/image-1:16c315-dev",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            true,
			IsDevVal:             true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315-dev-wip",
			"reg2.example.org/imagine/image-1:16c315-dev-wip",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			IsWIPRoot:            true,
			IsDevVal:             true,
		})

		ir.Scope.(*ImageScopeRootDir).WithoutSuffix = true
		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315",
			"reg2.example.org/imagine/image-1:16c315",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			TagsForHeadVal:       []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
			IsWIPRoot:            true,
			IsDevVal:             true,
		})

		_, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(Equal("unable make image tag: tree is not clean to use a tag"))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			TagsForHeadVal:       []string{"foobar"},
		})

		m, _ := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315",
			"reg2.example.org/imagine/image-1:16c315",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			TagsForHeadVal:       []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:v1.23.1",
			"reg2.example.org/imagine/image-1:v1.23.1",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			CommitHashForHeadVal: "16c315243fd31c00b80c188123099501ae2ccf91",
			TagsForHeadVal:       []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
			IsWIPRoot:            true,
			IsDevVal:             true,
		})

		ir.Scope.(*ImageScopeRootDir).WithoutSuffix = true
		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315",
			"reg2.example.org/imagine/image-1:16c315",
		))
	}
}

func TestWithSubDirScope(t *testing.T) {
	g := NewGomegaWithT(t)

	ir := &ImagineRecipe{
		Name: "image-1",
		Scope: &ImageScopeSubDir{
			RootDir:              "/go/src/github.com/errordeveloper/imagine",
			RelativeImageDirPath: "examples/image-1",
			Dockerfile:           "Dockerfile",
			WithoutSuffix:        true,
			Git: &git.FakeRepo{
				TreeHashForHeadVal: map[string]string{
					"examples/image-1": "16c315243fd31c00b80c188123099501ae2ccf91",
				},
			},
		},
	}

	g.Expect(ir.Scope.ContextPath()).To(Equal("/go/src/github.com/errordeveloper/imagine/examples/image-1"))

	g.Expect(ir.Scope.DockerfilePath()).To(Equal("/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile"))

	{
		ir.HasTests = false

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(HaveLen(0))
	}

	{
		ir.HasTests = false

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91",
			"reg2.example.org/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91",
		))
	}

	{
		ir.HasTests = true
		ir.Platforms = []string{"linux/amd64", "linux/arm64"}

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1-test", "image-1"))

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target).To(HaveKey("image-1-test"))

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91",
			"reg2.example.org/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91",
		))
		g.Expect(m.Target["image-1-test"].Tags).To(HaveLen(0))

		g.Expect(m.Target["image-1"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))
		g.Expect(m.Target["image-1-test"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))

		js, err := ir.ToBakeManifestAsJSON("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		expected := `
		  {
			"group": {
			  "default": {
				"targets": [
				  "image-1-test",
				  "image-1"
				]
			  }
			},
			"target": {
			  "image-1": {
				"context": "/go/src/github.com/errordeveloper/imagine/examples/image-1",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"tags": [
				  "reg1.example.com/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91",
				  "reg2.example.org/imagine/image-1:16c315243fd31c00b80c188123099501ae2ccf91"
				],
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				]
			  },
			  "image-1-test": {
				"context": "/go/src/github.com/errordeveloper/imagine/examples/image-1",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"target": "test",
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				]
			  }
			}
		  }
		`
		g.Expect(js).To(MatchJSON(expected))
	}
}
