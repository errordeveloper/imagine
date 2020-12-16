package recipe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/buildx/bake"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
)

const (
	schemaVersion            = "v1alpha1"
	labelPrefix              = "com.github.imagine."
	schemaVersionLabel       = labelPrefix + "schemaVersion"
	buildConfigDataLabel     = labelPrefix + "buildConfig.Data"
	buildConfigTreeHashLabel = labelPrefix + "buildConfig.TreeHash"
	ContextTreeHashLabel     = labelPrefix + "context.TreeHash"

	imagineDir = ".imagine"
)

type ImagineRecipe struct {
	WorkDir string

	Platforms []string

	Config struct {
		Path, Data string
	}

	Push, Export bool

	*config.BuildSpec

	Git struct {
		git.Git

		BaseBranch           string
		BranchedOffSuffix    string
		WorkInProgressSuffix string
	}
}

// type RepoManifest struct {
// 	Images []ImageManifest `json:"images"`
// }

// type ImageManifest struct {
// 	Name       string `json:"name"`
// 	FullRefs   []string
// 	SourceInfo ImageManifestSourceInfo
// }

// type ImageManifestSourceInfo struct {
// 	Path                  string
// 	Commit                string
// 	BaseBranch            string
// 	BuildBranch           string
// 	BaseBranchOriginURL   string
// 	CommitWasOnBaseBranch bool
// 	CommitURL             string
// }

// type FromImage struct {
// 	Name               string `json:"name"`
// 	FullRef            string `json:"fullRef"`
// 	PreferRegistry     string `json:"preferRegistry"`
// 	SourceRepoManifest string `json:"sourceRepoManifest"`
// }

// type ImagineRecipeVariants struct {
// 	FromImages []FromImage `json:"fromImages"`
// 	Variants   []Variants  `json:"args"`
// }

type bakeGroupMap map[string]*bake.Group
type bakeTargetMap map[string]*bake.Target

type BakeManifest struct {
	Group  bakeGroupMap  `json:"group"`
	Target bakeTargetMap `json:"target"`
}

func (r *ImagineRecipe) GetTag(variantName, configPath, contextPath string) (string, error) {
	suffix := ""
	if r.Git.BranchedOffSuffix != "" {
		branchedOff, err := r.Git.IsDev(r.Git.BaseBranch)
		if err != nil {
			return "", err
		}
		if branchedOff {
			suffix += "-dev"
		}
	}

	if r.Git.WorkInProgressSuffix != "" {
		configWIP, err := r.Git.IsWIP(configPath)
		if err != nil {
			return "", err
		}
		contextWIP, err := r.Git.IsWIP(contextPath)
		if err != nil {
			return "", err
		}
		if configWIP || contextWIP {
			suffix += "-wip"
		}
	}

	switch r.BuildSpec.TagMode {
	case "GitTreeHash":
		configTreeHash, err := r.Git.TreeHashForHead(configPath, true)
		if err != nil {
			return "", err
		}

		contextTreeHash, err := r.Git.TreeHashForHead(contextPath, true)
		if err != nil {
			return "", err
		}

		if variantName == "" {
			return fmt.Sprintf("%s.%s", configTreeHash, contextTreeHash) + suffix, nil
		}
		return fmt.Sprintf("%s.%s.%s", variantName, configTreeHash, contextTreeHash) + suffix, nil
	case "GitCommitHash":
		commitHash, err := r.Git.CommitHashForHead(true)
		if err != nil {
			return "", err
		}

		if variantName == "" {
			return fmt.Sprintf("%s", commitHash) + suffix, nil
		}
		return fmt.Sprintf("%s.%s", variantName, commitHash) + suffix, nil
	case "GitTagSemVer":
		// it doens't make sense to use a tag when tree is not clean, or
		// it is a development branch
		semVerTag, err := r.Git.SemVerTagForHead(false)
		if err != nil {
			return "", err
		}
		if semVerTag == nil {
			return "", fmt.Errorf("unexpected error: nil semver")
		}
		if suffix != "" {
			return "", fmt.Errorf("tree must be clean to use a git tag and it must be on given base branch")
		}
		return "v" + semVerTag.String(), nil

	default:
		return "", fmt.Errorf("unknown '.spec.tagMode' (%q)", r.BuildSpec.TagMode)
	}
}

func (r *ImagineRecipe) RegistryTags(variantName, variantContextPath string, registries ...string) ([]string, error) {
	registryTags := []string{}

	tag, err := r.GetTag(variantName, r.Config.Path, variantContextPath)
	if err != nil {
		return nil, fmt.Errorf("unable make image tag for image %q: %w", r.Name, err)
	}

	for _, registry := range registries {
		registryTag := fmt.Sprintf("%s/%s:%s", registry, r.Name, tag)
		registryTags = append(registryTags, registryTag)
	}

	return registryTags, nil
}

func (r *ImagineRecipe) newBakeTarget(buildInstructions *config.WithBuildInstructions) *bake.Target {
	target := &bake.Target{
		Context:   new(string),
		Platforms: r.Platforms,
		Args:      r.Args,
	}

	*target.Context = buildInstructions.ContextPath(r.WorkDir)

	if dockerfilePath := buildInstructions.DockerfilePath(r.WorkDir); dockerfilePath != "" {
		target.Dockerfile = new(string)
		*target.Dockerfile = dockerfilePath
	}
	if dockerfileBody := buildInstructions.Dockerfile.Body; dockerfileBody != "" {
		target.DockerfileInline = new(string)
		*target.DockerfileInline = dockerfileBody
	}

	return target
}

const (
	ImageTestStageName   = "test"
	DefaultBakeGroup     = "default"
	BakeTestTargetSuffix = "-test"
)

func (r *ImagineRecipe) buildVariantToBakeTargets(imageName, variantName string, buildInstructions *config.WithBuildInstructions, registries ...string) (bakeTargetMap, []string, error) {

	mainTargetName := imageName
	if variantName != "" {
		mainTargetName += "-" + variantName
	}
	testTargetName := mainTargetName + BakeTestTargetSuffix

	targets := bakeTargetMap{}

	mainTarget := r.newBakeTarget(buildInstructions)

	push := (r.Push && len(registries) != 0 && !*buildInstructions.Untagged)

	if !*buildInstructions.Untagged {
		registryTags, err := r.RegistryTags(variantName, *buildInstructions.Dir, registries...)
		if err != nil {
			return nil, nil, err
		}

		mainTarget.Tags = registryTags
	}

	if buildInstructions.Target != nil {
		mainTarget.Target = buildInstructions.Target
	}

	for _, secret := range buildInstructions.Secrets {
		mainTarget.Secrets = append(mainTarget.Secrets, secret.String())
	}

	configTreeHash, err := r.Git.TreeHashForHead(r.Config.Path, false)
	if err != nil {
		return nil, nil, err
	}

	contextTreeHash, err := r.Git.TreeHashForHead(*buildInstructions.Dir, false)
	if err != nil {
		return nil, nil, err
	}

	mainTarget.Labels = map[string]string{
		schemaVersionLabel:       schemaVersion,
		buildConfigDataLabel:     r.Config.Data,
		buildConfigTreeHashLabel: configTreeHash,
		ContextTreeHashLabel:     contextTreeHash,
	}

	// TODO: label for HEAD

	// this is a slice, but buildx doesn't support multiple outputs
	// at present (https://github.com/docker/buildx/issues/316)
	mainTarget.Outputs = []string{
		fmt.Sprintf("type=image,push=%v", push),
	}

	if r.Export {
		mainTarget.Outputs = []string{
			fmt.Sprintf("type=docker,dest=%s",
				filepath.Join(buildInstructions.ContextPath(r.WorkDir), fmt.Sprintf("image-%s.oci", r.Name))),
		}
	}

	targets[mainTargetName] = mainTarget

	if buildInstructions.Test != nil && *buildInstructions.Test {
		testTarget := r.newBakeTarget(buildInstructions)

		testTarget.Target = new(string)
		*testTarget.Target = ImageTestStageName

		targets[testTargetName] = testTarget

		return targets, []string{testTargetName, mainTargetName}, nil
	}

	return targets, []string{mainTargetName}, nil
}

func (r *ImagineRecipe) ToBakeManifest(registries ...string) (*BakeManifest, error) {
	if r.BuildSpec == nil {
		return nil, fmt.Errorf("unexpected error: BuildSpec not set in %T", *r)
	}

	if len(r.Variants) == 0 {
		targets, targetNames, err := r.buildVariantToBakeTargets(r.Name, "", r.WithBuildInstructions, registries...)
		if err != nil {
			return nil, err
		}

		return &BakeManifest{
			Group: bakeGroupMap{
				DefaultBakeGroup: &bake.Group{
					Targets: targetNames,
				},
			},
			Target: targets,
		}, nil
	}

	manifest := &BakeManifest{
		Group: bakeGroupMap{
			DefaultBakeGroup: &bake.Group{
				Targets: []string{},
			},
		},
		Target: bakeTargetMap{},
	}

	for _, variant := range r.Variants {
		targets, targetNames, err := r.buildVariantToBakeTargets(r.Name, variant.Name, variant.With, registries...)
		if err != nil {
			return nil, err
		}

		for _, targetName := range targetNames {
			manifest.Target[targetName] = targets[targetName]
		}
		manifest.Group[DefaultBakeGroup].Targets = append(manifest.Group[DefaultBakeGroup].Targets, targetNames...)
	}
	return manifest, nil
}

func (r *ImagineRecipe) WriteManifest(registries ...string) (string, func(), error) {
	imagineDirPath := filepath.Join(r.WorkDir, imagineDir)
	if err := os.MkdirAll(imagineDirPath, 0755); err != nil {
		return "", func() {}, err
	}
	tempDir, err := ioutil.TempDir(imagineDirPath, "build-*")
	if err != nil {
		return "", func() {}, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	manifest := filepath.Join(tempDir, fmt.Sprintf("buildx-%s.json", r.Name))

	m, err := r.ToBakeManifest(registries...)
	if err != nil {
		return "", func() {}, err
	}

	if err := m.WriteFile(manifest); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return manifest, cleanup, nil
}

func (m *BakeManifest) RegistryTags() []string {
	registryTags := []string{}
	for _, target := range m.Target {
		registryTags = append(registryTags, target.Tags...)
	}
	return registryTags
}

func (m *BakeManifest) ToJSON() (string, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *BakeManifest) WriteFile(filename string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}
