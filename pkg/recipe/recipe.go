package recipe

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/docker/buildx/bake"
)

type ImageScope interface {
	DockerfilePath() string
	ContextPath() string
	MakeTag() string
}
type ImageTagger interface {
	MakeTag() string
}

var (
	_ ImageScope = &ImageScopeRootDir{}
	//_ ImageScope = &ImageScopeSubDir{}
)

type ImageScopeRootDir struct {
	RootDir                string
	RelativeDockerfilePath string
	Tagger                 ImageTagger
}

func (i *ImageScopeRootDir) DockerfilePath() string {
	return filepath.Join(i.RootDir, i.RelativeDockerfilePath)
}

func (i *ImageScopeRootDir) ContextPath() string {
	return i.RootDir
}

func (i *ImageScopeRootDir) MakeTag() string {
	return i.Tagger.MakeTag()
}

type ImageScopeSubDir struct {
}

type ContextType int

const (
	ContextTypeRootDir ContextType = iota
	ContextTypeImageDir
)

const (
	TestBakeTargetNameSuffix = "-test"
	TestImageBuildTargetName = "test"
)

type ImagineRecipe struct {
	Name      string
	Scope     ImageScope
	Platforms []string
	HasTests  bool
}

type bakeGroupMap map[string]*bake.Group
type bakeTargetMap map[string]*bake.Target

type BakeManifest struct {
	Group  bakeGroupMap  `json:"group"`
	Target bakeTargetMap `json:"target"`
}

func (r *ImagineRecipe) newBakeTarget() *bake.Target {
	target := &bake.Target{
		Context:    new(string),
		Dockerfile: new(string),
		Platforms:  r.Platforms,
	}
	*target.Context = r.Scope.ContextPath()
	*target.Dockerfile = r.Scope.DockerfilePath()
	return target
}

func (r *ImagineRecipe) ToBakeManifest(registries ...string) *BakeManifest {
	group := &bake.Group{
		Targets: []string{r.Name},
	}

	mainTarget := r.newBakeTarget()

	targets := bakeTargetMap{
		r.Name: mainTarget,
	}

	tag := r.Scope.MakeTag()
	for _, registry := range registries {
		registryTag := fmt.Sprintf("%s/%s:%s", registry, r.Name, tag)
		mainTarget.Tags = append(mainTarget.Tags, registryTag)
	}

	if r.HasTests {
		testTarget := r.newBakeTarget()
		testTarget.Target = new(string)
		*testTarget.Target = TestImageBuildTargetName
		targets[r.Name+TestBakeTargetNameSuffix] = testTarget
		group.Targets = []string{r.Name + TestBakeTargetNameSuffix, r.Name}
	}

	return &BakeManifest{
		Group: bakeGroupMap{
			"default": group,
		},
		Target: targets,
	}
}

func (r *ImagineRecipe) ToBakeManifestAsJSON(registries ...string) ([]byte, error) {
	return json.Marshal(r.ToBakeManifest(registries...))
}
