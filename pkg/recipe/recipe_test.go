package recipe_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
	. "github.com/errordeveloper/imagine/pkg/recipe"
)

const commonWD = "/go/src/github.com/errordeveloper/imagine"

func TestManifestsBasic(t *testing.T) {
	g := NewGomegaWithT(t)

	ir := &ImagineRecipe{
		BuildSpec: &config.BuildSpec{
			Name: "image-1",
			WithBuildInstructions: &config.WithBuildInstructions{
				Dockerfile: &config.DockerfileBuildInstructions{},
			},
		},
	}

	ir.Config.Path = "examples/image-1.yaml"

	ir.Dir = new(string)
	*ir.Dir = ""

	ir.Dockerfile.Path = "examples/image-1/Dockerfile"

	g.Expect(ir.BuildSpec.ApplyDefaultsAndValidate()).To(Succeed())

	ir.Git.Git = &git.FakeRepo{
		TreeHashForHeadRoot: "16c315243fd31c00b80c188123099501ae2ccf91",
		TreeHashForHeadVal: map[string]string{
			ir.Config.Path: "0c108230c1b6c0032ccf8199315243fd1ae81591",
		},
	}

	ir.Test = new(bool)

	g.Expect(ir.BuildSpec.WithBuildInstructions.ContextPath(commonWD)).To(Equal(commonWD))

	g.Expect(ir.BuildSpec.WithBuildInstructions.DockerfilePath(commonWD)).To(Equal(commonWD + "/examples/image-1/Dockerfile"))

	ir.WorkDir = commonWD
	ir.Config.Data = "W3sgInRlc3QiOiB0cnVlIH1dCg=="

	{
		*ir.Test = false

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(HaveLen(0))
	}

	{
		*ir.Test = false

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:0c1082.16c315",
			"reg2.example.org/imagine/image-1:0c1082.16c315",
		))
	}

	{
		*ir.Test = true
		ir.Platforms = []string{"linux/amd64", "linux/arm64"}

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1-test", "image-1"))

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target).To(HaveKey("image-1-test"))

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:0c1082.16c315",
			"reg2.example.org/imagine/image-1:0c1082.16c315",
		))
		g.Expect(m.Target["image-1-test"].Tags).To(HaveLen(0))

		g.Expect(m.Target["image-1"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))
		g.Expect(m.Target["image-1-test"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))

		js, err := m.ToJSON()
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
				"labels": {
				  "com.github.errordeveloper.imagine.buildConfig.Data": "W3sgInRlc3QiOiB0cnVlIH1dCg==",
				  "com.github.errordeveloper.imagine.buildConfig.TreeHash": "0c108230c1b6c0032ccf8199315243fd1ae81591",
				  "com.github.errordeveloper.imagine.context.TreeHash": "16c315243fd31c00b80c188123099501ae2ccf91",
				  "com.github.errordeveloper.imagine.schemaVersion": "v1alpha1"
				},
				"tags": [
				  "reg1.example.com/imagine/image-1:0c1082.16c315",
				  "reg2.example.org/imagine/image-1:0c1082.16c315"
				],
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				],
				"output": [
                  "type=image,push=false"
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

func TestTagging(t *testing.T) {
	g := NewGomegaWithT(t)

	newImagineRecipe := func(repo *git.FakeRepo) *ImagineRecipe {
		dir := ""
		ir := &recipe.ImagineRecipe{
			WorkDir: commonWD,
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

		repo.CommitHashForHeadVal = "15b881c016c1d81f924cc0c1ae002333253f0991"
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

	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: true,
			IsDevVal:  false,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315-wip",
			"reg2.example.org/imagine/image-1:613919.16c315-wip",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: false,
			IsDevVal:  true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315-dev",
			"reg2.example.org/imagine/image-1:613919.16c315-dev",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: true,
			IsDevVal:  true,
		})

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315-dev-wip",
			"reg2.example.org/imagine/image-1:613919.16c315-dev-wip",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			IsWIPRoot: true,
			IsDevVal:  true,
		})

		ir.Git.BranchedOffSuffix = ""
		ir.Git.WorkInProgressSuffix = ""
		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315",
			"reg2.example.org/imagine/image-1:613919.16c315",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			TagsForHeadVal: []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
			IsWIPRoot:      true,
			IsDevVal:       true,
		})

		ir.TagMode = "GitTagSemVer"

		_, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(Equal(`unable make image tag for image "image-1": cannot use tag because of "-dev-wip" suffix`))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{
			TagsForHeadVal: []string{"foobar"},
		})

		m, _ := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315",
			"reg2.example.org/imagine/image-1:613919.16c315",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			TagsForHeadVal: []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
		})

		ir.TagMode = "GitTagSemVer"

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:v1.23.1",
			"reg2.example.org/imagine/image-1:v1.23.1",
		))
	}
	{
		ir := newImagineRecipe(&git.FakeRepo{
			TagsForHeadVal: []string{"v1.22.9", "v1.23.1", "1.23.1", "1.20.9"},
			IsWIPRoot:      true,
			IsDevVal:       true,
		})

		ir.Git.BranchedOffSuffix = ""
		ir.Git.WorkInProgressSuffix = ""
		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:613919.16c315",
			"reg2.example.org/imagine/image-1:613919.16c315",
		))
	}
}

func TestManifestsWithVariants(t *testing.T) {
	g := NewGomegaWithT(t)

	newImagineRecipe := func(repo *git.FakeRepo) *ImagineRecipe {
		dir := "examples/image-1"
		ir := &recipe.ImagineRecipe{
			WorkDir: commonWD,
			BuildSpec: &config.BuildSpec{
				Name: "image-1",
				WithBuildInstructions: &config.WithBuildInstructions{
					Dir:        &dir,
					Test:       new(bool),
					Dockerfile: &config.DockerfileBuildInstructions{},
				},
				Variants: []config.BuildVariant{
					{
						Name: "foo",
					},
				},
			},
		}

		ir.Config.Path = "dummy.yaml"

		repo.TreeHashForHeadRoot = "16c315243fd31c00b80c188123099501ae2ccf91"
		repo.TreeHashForHeadVal = map[string]string{
			"dummy.yaml":       "613919533ebd03d6bafbd538ccad3a4acea9b761",
			"examples/image-1": "16c315243fd31c00b80c188123099501ae2ccf91",
		}
		repo.IsWIPVal = map[string]bool{
			"dummy.yaml":       false,
			"examples/image-1": false,
		}

		ir.Git.Git = repo
		ir.Git.BranchedOffSuffix = "-dev"
		ir.Git.WorkInProgressSuffix = "-wip"

		g.Expect(ir.BuildSpec.ApplyDefaultsAndValidate()).To(Succeed())

		return ir
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{})

		g.Expect(ir.BuildSpec.WithBuildInstructions.ContextPath(commonWD)).To(Equal(commonWD + "/examples/image-1"))

		g.Expect(ir.BuildSpec.WithBuildInstructions.DockerfilePath(commonWD)).To(Equal(commonWD + "/examples/image-1/Dockerfile"))

		*ir.BuildSpec.Test = false

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("index", "image-1-foo"))

		g.Expect(m.Target).To(HaveLen(2))

		g.Expect(m.Target).To(HaveKey("index"))
		g.Expect(m.Target["index"].Tags).To(HaveLen(0))

		g.Expect(m.Target).To(HaveKey("image-1-foo"))
		g.Expect(m.Target["image-1-foo"].Tags).To(HaveLen(0))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{})

		*ir.BuildSpec.Test = false

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("index", "image-1-foo"))

		g.Expect(m.Target).To(HaveLen(2))

		g.Expect(m.Target).To(HaveKey("index"))
		g.Expect(m.Target["index"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:index.15b881",
			"reg2.example.org/imagine/image-1:index.15b881",
		))

		g.Expect(m.Target).To(HaveKey("image-1-foo"))
		g.Expect(m.Target["image-1-foo"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:foo.613919.16c315",
			"reg2.example.org/imagine/image-1:foo.613919.16c315",
		))
	}

	{
		ir := newImagineRecipe(&git.FakeRepo{})

		*ir.BuildSpec.Test = true
		ir.Platforms = []string{"linux/amd64", "linux/arm64"}

		m, err := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1-foo-test", "image-1-foo", "index"))

		g.Expect(m.Target).To(HaveLen(3))
		g.Expect(m.Target).To(HaveKey("image-1-foo"))
		g.Expect(m.Target).To(HaveKey("image-1-foo-test"))
		g.Expect(m.Target).To(HaveKey("index"))

		g.Expect(m.Target["image-1-foo"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:foo.613919.16c315",
			"reg2.example.org/imagine/image-1:foo.613919.16c315",
		))
		g.Expect(m.Target["image-1-foo-test"].Tags).To(HaveLen(0))

		g.Expect(m.Target["image-1-foo"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))
		g.Expect(m.Target["image-1-foo-test"].Platforms).To(ConsistOf("linux/amd64", "linux/arm64"))

		js, err := m.ToJSON()
		g.Expect(err).ToNot(HaveOccurred())

		expected := `
		  {
			"group": {
			  "default": {
				"targets": [
				  "index",
				  "image-1-foo-test",
				  "image-1-foo"
				]
			  }
			},
			"target": {
			  "image-1-foo": {
				"context": "/go/src/github.com/errordeveloper/imagine/examples/image-1",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"labels": {
				  "com.github.errordeveloper.imagine.buildConfig.Data": "",
				  "com.github.errordeveloper.imagine.buildConfig.TreeHash": "613919533ebd03d6bafbd538ccad3a4acea9b761",
				  "com.github.errordeveloper.imagine.context.TreeHash": "16c315243fd31c00b80c188123099501ae2ccf91",
				  "com.github.errordeveloper.imagine.schemaVersion": "v1alpha1"
				},
				"tags": [
				  "reg1.example.com/imagine/image-1:foo.613919.16c315",
				  "reg2.example.org/imagine/image-1:foo.613919.16c315"
				],
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				],
				"output": [
				  "type=image,push=false"
				]
			  },
			  "image-1-foo-test": {
				"context": "/go/src/github.com/errordeveloper/imagine/examples/image-1",
				"dockerfile": "/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile",
				"target": "test",
				"platforms": [
				  "linux/amd64",
				  "linux/arm64"
				]
			  },
			  "index": {
				"context": "",
				"dockerfile-inline": "FROM scratch\nCOPY index-image-1.json /index.json\n",
				"labels": {
				  "com.github.errordeveloper.imagine.buildConfig.Data": "",
				  "com.github.errordeveloper.imagine.indexSchemaVersion": "v1alpha1",
				  "com.github.errordeveloper.imagine.schemaVersion": "v1alpha1"
				},
				"output": [
				  "type=image,push=false"
				]
			  }
			}
		  }
		`
		g.Expect(js).To(MatchJSON(expected))
	}
}

func TestOutputModes(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := "examples/image-1"
	ir := &ImagineRecipe{
		WorkDir: commonWD,
		BuildSpec: &config.BuildSpec{
			Name: "image-2",
			WithBuildInstructions: &config.WithBuildInstructions{
				Dir:  &dir,
				Test: new(bool),
			},
		},
	}

	ir.Config.Path = "dummy.yaml"

	ir.Git.Git = &git.FakeRepo{
		CommitHashForHeadVal: "15b881c016c1d81f924cc0c1ae002333253f0991",
		TreeHashForHeadVal: map[string]string{
			"examples/image-1": "16c315243f8123099501ae2ccd31c00b80c18f91",
			"dummy.yaml":       "613919533ebd03d6bafbd538ccad3a4acea9b761",
		},
		IsWIPVal: map[string]bool{
			"examples/image-1": false,
			"dummy.yaml":       false,
		},
	}
	ir.Git.BranchedOffSuffix = "-dev"
	ir.Git.WorkInProgressSuffix = "-wip"

	g.Expect(ir.BuildSpec.ApplyDefaultsAndValidate()).To(Succeed())

	{
		*ir.BuildSpec.Test = false

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-2"))
		g.Expect(m.Target["image-2"].Outputs).To(ConsistOf("type=image,push=false"))
		g.Expect(m.Target["image-2"].Tags).To(HaveLen(0))
	}

	{
		*ir.BuildSpec.Test = true

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-2"))
		g.Expect(m.Target).To(HaveKey("image-2-test"))

		g.Expect(m.Target["image-2"].Outputs).To(ConsistOf("type=image,push=false"))
		g.Expect(m.Target["image-2"].Tags).To(HaveLen(0))
		g.Expect(m.Target["image-2-test"].Outputs).To(HaveLen(0))
		g.Expect(m.Target["image-2-test"].Tags).To(HaveLen(0))
	}

	{
		*ir.BuildSpec.Test = true

		m, err := ir.ToBakeManifest("example.com/reg", "example.org/reg")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-2"))
		g.Expect(m.Target).To(HaveKey("image-2-test"))

		g.Expect(m.Target["image-2"].Outputs).To(ConsistOf("type=image,push=false"))

		g.Expect(m.Target["image-2"].Tags).To(ConsistOf(
			"example.com/reg/image-2:613919.16c315",
			"example.org/reg/image-2:613919.16c315",
		))

		g.Expect(m.Target["image-2-test"].Outputs).To(HaveLen(0))
		g.Expect(m.Target["image-2-test"].Tags).To(HaveLen(0))

	}

	{
		ir.Export = true
		ir.ExportDir = "/tmp"

		m, err := ir.ToBakeManifest()
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target["image-2"].Outputs).To(ConsistOf("type=docker,dest=/tmp/image-image-2.oci"))
		g.Expect(m.Target["image-2"].Tags).To(HaveLen(0))
		g.Expect(m.Target["image-2-test"].Outputs).To(HaveLen(0))
		g.Expect(m.Target["image-2-test"].Tags).To(HaveLen(0))
	}

	{
		*ir.BuildSpec.Test = true

		m, err := ir.ToBakeManifest("example.com/reg", "example.org/reg")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-2"))
		g.Expect(m.Target).To(HaveKey("image-2-test"))

		g.Expect(m.Target["image-2"].Tags).To(ConsistOf(
			"example.com/reg/image-2:613919.16c315",
			"example.org/reg/image-2:613919.16c315",
		))
	}
}
