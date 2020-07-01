package recipe_test

import (
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/imagine/pkg/recipe"
)

type FakeImageTagger struct{}

func (*FakeImageTagger) MakeTag() string { return "t1" }

func TestWithRootDirScope(t *testing.T) {
	g := NewGomegaWithT(t)

	ir := &ImagineRecipe{
		Name: "image-1",
		Scope: &ImageScopeRootDir{
			RootDir:                "/go/src/github.com/errordeveloper/imagine",
			RelativeDockerfilePath: "examples/image-1/Dockerfile",
			Tagger:                 &FakeImageTagger{},
		},
	}

	g.Expect(ir.Scope.ContextPath()).To(Equal("/go/src/github.com/errordeveloper/imagine"))

	g.Expect(ir.Scope.DockerfilePath()).To(Equal("/go/src/github.com/errordeveloper/imagine/examples/image-1/Dockerfile"))

	{
		ir.HasTests = false

		m := ir.ToBakeManifest()

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(HaveLen(0))
	}

	{
		ir.HasTests = false

		m := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1"))

		g.Expect(m.Target).To(HaveLen(1))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:t1",
			"reg2.example.org/imagine/image-1:t1",
		))
	}

	{
		ir.HasTests = true
		ir.Platforms = []string{"linux/amd64", "linux/arm64"}

		m := ir.ToBakeManifest("reg1.example.com/imagine", "reg2.example.org/imagine")

		g.Expect(m.Group).To(HaveKey("default"))
		g.Expect(m.Group["default"].Targets).To(ConsistOf("image-1-test", "image-1"))

		g.Expect(m.Target).To(HaveLen(2))
		g.Expect(m.Target).To(HaveKey("image-1"))
		g.Expect(m.Target).To(HaveKey("image-1-test"))

		g.Expect(m.Target["image-1"].Tags).To(ConsistOf(
			"reg1.example.com/imagine/image-1:t1",
			"reg2.example.org/imagine/image-1:t1",
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
				  "reg1.example.com/imagine/image-1:t1",
				  "reg2.example.org/imagine/image-1:t1"
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
