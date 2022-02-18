package recipe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/buildx/bake"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
)

const (
	schemaVersion      = "v1alpha1"
	indexSchemaVersion = "v1alpha1"
	labelPrefix        = "com.github.errordeveloper.imagine."

	schemaVersionLabel      = labelPrefix + "schemaVersion"
	indexSchemaVersionLabel = labelPrefix + "indexSchemaVersion"

	buildConfigDataLabel     = labelPrefix + "buildConfig.Data"
	buildConfigTreeHashLabel = labelPrefix + "buildConfig.TreeHash"
	contextTreeHashLabel     = labelPrefix + "context.TreeHash"
)

type ImagineRecipe struct {
	WorkDir string

	Platforms []string

	Config struct {
		Path, Data string
	}

	Push, Export bool
	ExportDir    string

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
			suffix += r.Git.BranchedOffSuffix
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
			suffix += r.Git.WorkInProgressSuffix
		}
	}

	switch r.BuildSpec.TagMode {
	case "GitTreeHash":
		return r.makeGitTreeHashTag(configPath, contextPath, variantName, suffix)
	case "GitCommitHash":
		return r.makeGitCommitHashTag(configPath, configPath, variantName, suffix)
	case "GitTagSemVer":
		return r.makeGitTagSemVerTag(configPath, configPath, variantName, suffix)
	default:
		return "", fmt.Errorf("unknown '.spec.tagMode' (%q)", r.BuildSpec.TagMode)
	}
}
func (r *ImagineRecipe) makeGitTreeHashTag(configPath, contextPath, variantName, suffix string) (string, error) {
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
}

func (r *ImagineRecipe) makeGitCommitHashTag(_, _, variantName, suffix string) (string, error) {
	commitHash, err := r.Git.CommitHashForHead(true)
	if err != nil {
		return "", err
	}

	if variantName == "" {
		return commitHash + suffix, nil
	}
	return fmt.Sprintf("%s.%s", variantName, commitHash) + suffix, nil
}

func (r *ImagineRecipe) makeGitTagSemVerTag(_, _, variantName, suffix string) (string, error) {
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
		return "", fmt.Errorf("cannot use tag because of %q suffix", suffix)
	}
	if variantName == "" {
		return fmt.Sprintf("v%s", semVerTag.String()), nil
	}
	return fmt.Sprintf("%s.v%s", variantName, semVerTag.String()), nil
}

func (r *ImagineRecipe) RegistryRefs(variantName, variantContextPath string, registries ...string) ([]string, error) {
	refs := []string{}

	tag, err := r.GetTag(variantName, r.Config.Path, variantContextPath)
	if err != nil {
		return nil, fmt.Errorf("unable make image tag for image %q: %w", r.Name, err)
	}

	for _, registry := range registries {
		ref := fmt.Sprintf("%s/%s:%s", registry, r.Name, tag)
		refs = append(refs, ref)
	}

	return refs, nil
}

func (r *ImagineRecipe) RegistryIndexRefs(registries ...string) ([]string, error) {
	refs := []string{}

	tag, err := r.makeGitCommitHashTag("", "", "index", "")
	if err != nil {
		return nil, fmt.Errorf("unable make index tag for image %q: %w", r.Name, err)
	}

	for _, registry := range registries {
		ref := fmt.Sprintf("%s/%s:%s", registry, r.Name, tag)
		refs = append(refs, ref)
	}

	return refs, nil
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
	ImageTestStageName    = "test"
	DefaultBakeGroup      = "default"
	BakeTestTargetSuffix  = "-test"
	IndexTargetNamePrefix = "index-"
)

func (r *ImagineRecipe) commonLabels() map[string]string {
	return map[string]string{
		schemaVersionLabel:   schemaVersion,
		buildConfigDataLabel: r.Config.Data,
	}
}

func (r *ImagineRecipe) indexAsBakeTarget(imageName string, registries ...string) (*bake.Target, string, error) {
	indexTargetName := IndexTargetNamePrefix + imageName
	indexTarget := &bake.Target{
		Context:          new(string),
		DockerfileInline: new(string),
		Labels:           r.commonLabels(),
	}

	indexTarget.Labels[indexSchemaVersionLabel] = indexSchemaVersion

	*indexTarget.DockerfileInline = fmt.Sprintf("FROM scratch\nCOPY index-%s.json /index.json\n", r.Name)

	refs, err := r.RegistryIndexRefs(registries...)
	if err != nil {
		return nil, "", err
	}

	indexTarget.Tags = refs

	shouldPush := (r.Push && len(registries) != 0)
	r.setOutputs(indexTargetName, indexTarget, shouldPush)

	return indexTarget, indexTargetName, nil
}

func (r *ImagineRecipe) buildVariantToBakeTargets(imageName, variantName string, buildInstructions *config.WithBuildInstructions, registries ...string) (bakeTargetMap, []string, error) {
	mainTargetName := imageName
	if variantName != "" {
		mainTargetName += "-" + variantName
	}
	testTargetName := mainTargetName + BakeTestTargetSuffix

	targets := bakeTargetMap{}

	mainTarget := r.newBakeTarget(buildInstructions)

	if !*buildInstructions.Untagged {
		refs, err := r.RegistryRefs(variantName, *buildInstructions.Dir, registries...)
		if err != nil {
			return nil, nil, err
		}

		mainTarget.Tags = refs
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

	mainTarget.Labels = r.commonLabels()

	mainTarget.Labels[buildConfigTreeHashLabel] = configTreeHash
	mainTarget.Labels[contextTreeHashLabel] = contextTreeHash

	for k, v := range buildInstructions.Labels {
		mainTarget.Labels[k] = v
	}

	shouldPush := (r.Push && len(registries) != 0 && !*buildInstructions.Untagged)
	r.setOutputs(mainTargetName, mainTarget, shouldPush)

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

func (r *ImagineRecipe) setOutputs(targetName string, target *bake.Target, shouldPush bool) {
	// this is a slice, but buildx doesn't support multiple outputs
	// at present (https://github.com/docker/buildx/issues/316)
	target.Outputs = []string{
		fmt.Sprintf("type=image,push=%v", shouldPush),
	}

	if r.Export {
		exportFilename := fmt.Sprintf("image-%s.oci", targetName)
		target.Outputs = []string{
			fmt.Sprintf("type=docker,dest=%s",
				filepath.Join(r.ExportDir, exportFilename)),
		}
	}

}

func (r *ImagineRecipe) ToBakeManifest(registries ...string) (*BakeManifest, error) {
	if r.BuildSpec == nil {
		return nil, fmt.Errorf("unexpected error: BuildSpec not set in %T", *r)
	}

	if r.ExportDir == "" {
		r.ExportDir = r.WorkDir
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

	indexBakeTarget, indexTargetName, err := r.indexAsBakeTarget(r.Name, registries...)
	if err != nil {
		return nil, err
	}

	manifest := &BakeManifest{
		Group: bakeGroupMap{
			DefaultBakeGroup: &bake.Group{
				Targets: []string{indexTargetName},
			},
		},
		Target: bakeTargetMap{indexTargetName: indexBakeTarget},
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

func (r *ImagineRecipe) WriteIndex(filename string) error {
	index := struct{}{}

	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (r *ImagineRecipe) WriteManifest(stateDirPath string, registries ...string) (string, func(), error) {
	if err := os.MkdirAll(stateDirPath, 0755); err != nil {
		return "", func() {}, err
	}
	tempDir, err := os.MkdirTemp(stateDirPath, "build-*")
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

	index := filepath.Join(tempDir, fmt.Sprintf("index-%s.json", r.Name))

	if err := r.WriteIndex(index); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return manifest, cleanup, nil
}

func (m *BakeManifest) RegistryRefs() []string {
	refs := []string{}
	for _, target := range m.Target {
		refs = append(refs, target.Tags...)
	}
	return refs
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
	return os.WriteFile(filename, data, 0644)
}
