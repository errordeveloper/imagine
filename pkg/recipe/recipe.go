package recipe

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/docker/buildx/bake"

	"github.com/errordeveloper/imagine/pkg/git"
)

type ImageScope interface {
	DockerfilePath() string
	ContextPath() string
	MakeTag() (string, error)
}

var (
	_ ImageScope = &ImageScopeRootDir{}
	_ ImageScope = &ImageScopeSubDir{}
)

type ImageScopeRootDir struct {
	RootDir                string
	RelativeDockerfilePath string

	BaseBranch    string
	WithoutSuffix bool
	Git           git.Git
}

func (i *ImageScopeRootDir) DockerfilePath() string {
	return filepath.Join(i.RootDir, i.RelativeDockerfilePath)
}

func (i *ImageScopeRootDir) ContextPath() string {
	return i.RootDir
}

func (i *ImageScopeRootDir) MakeTag() (string, error) {
	commitHash, err := i.Git.CommitHashForHead(true)
	if err != nil {
		return "", err
	}

	if i.WithoutSuffix {
		return commitHash, nil
	}

	isDev, err := i.Git.IsDev(i.BaseBranch)
	if err != nil {
		return "", err
	}
	if isDev {
		commitHash += "-dev"
	}

	isWIP, err := i.Git.IsWIP("")
	if err != nil {
		return "", err
	}
	if isWIP {
		commitHash += "-wip"
	}

	// it doens't make sense to use a tag when tree is not clean, or
	// it is a development branch
	if semVerTag, _ := i.Git.SemVerTagForHead(false); semVerTag != nil {
		if !isDev && !isWIP {
			return "v" + semVerTag.String(), nil
		}
		return "", fmt.Errorf("tree is not clean to use a tag")
	}

	return commitHash, nil
}

type ImageScopeSubDir struct {
	RootDir              string
	RelativeImageDirPath string
	Dockerfile           string

	BaseBranch    string
	WithoutSuffix bool
	Git           git.Git
}

func (i *ImageScopeSubDir) DockerfilePath() string {
	return filepath.Join(i.RootDir, i.RelativeImageDirPath, i.Dockerfile)
}
func (i *ImageScopeSubDir) ContextPath() string {
	return filepath.Join(i.RootDir, i.RelativeImageDirPath)
}

func (i *ImageScopeSubDir) MakeTag() (string, error) {
	treeHash, err := i.Git.TreeHashForHead(i.RelativeImageDirPath)
	if err != nil {
		return "", err
	}

	if i.WithoutSuffix {
		return treeHash, nil
	}

	isDev, err := i.Git.IsDev(i.BaseBranch)
	if err != nil {
		return "", err
	}
	if isDev {
		treeHash += "-dev"
	}

	isWIP, err := i.Git.IsWIP(i.RelativeImageDirPath)
	if err != nil {
		return "", err
	}
	if isWIP {
		treeHash += "-wip"
	}

	return treeHash, nil
}

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

func (r *ImagineRecipe) ToBakeManifest(registries ...string) (*BakeManifest, error) {
	group := &bake.Group{
		Targets: []string{r.Name},
	}

	mainTarget := r.newBakeTarget()

	targets := bakeTargetMap{
		r.Name: mainTarget,
	}

	tag, err := r.Scope.MakeTag()
	if err != nil {
		return nil, fmt.Errorf("unable make image tag: %w", err)
	}

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
	}, nil
}

func (r *ImagineRecipe) ToBakeManifestAsJSON(registries ...string) ([]byte, error) {
	m, err := r.ToBakeManifest(registries...)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
