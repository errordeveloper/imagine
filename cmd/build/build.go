package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/buildx"
	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/rebuilder"
	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"
)

type Flags struct {
	Builder string

	Platforms, Registries []string // TODO(post-mvp): make these repo config fields
	UpstreamBranch        string   // TODO(post-mvp): make this repo config fields

	Push, Export, Force, Debug bool

	Config string
}

const (
	stateDir = ".imagine" // TODO(post-mvp): make this repo config field

	defaultPlatform       = "linux/amd64"
	defaultUpstreamBranch = "origin/master"
)

func BuildCmd() *cobra.Command {

	flags := &Flags{}

	cmd := &cobra.Command{
		Use: "build",
		//Args: cobra.NoArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitBuildCmd(cmd); err != nil {
				return err
			}
			return flags.RunBuildCmd()
		},
	}

	cmd.Flags().StringVar(&flags.Builder, "builder", "", "use a global buildx builder instead of creating one")

	cmd.Flags().StringVar(&flags.Config, "config", "", "path to build config file")
	cmd.MarkFlagRequired("config")

	cmd.Flags().StringArrayVar(&flags.Platforms, "platform", []string{defaultPlatform}, "platforms to target")
	cmd.Flags().StringArrayVar(&flags.Registries, "registry", []string{}, "registry prefixes to use for tags")
	cmd.Flags().StringVar(&flags.UpstreamBranch, "upstream-branch", defaultUpstreamBranch, "upstream branch of the repository")

	cmd.Flags().BoolVar(&flags.Push, "push", false, "whether to push image to registries or not (if any registries are given)")
	cmd.Flags().BoolVar(&flags.Export, "export", false, "whether to export the image to an OCI tarball 'image-<name>.oci'")

	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuild the image")
	cmd.Flags().BoolVar(&flags.Debug, "debug", false, "print debuging info and keep generated buildx manifest file")

	return cmd
}

func (f *Flags) InitBuildCmd(cmd *cobra.Command) error {
	return nil
}

func (f *Flags) RunBuildCmd() error {
	initialWD, err := os.Getwd()
	if err != nil {
		return err
	}

	stateDirPath := filepath.Join(initialWD, stateDir)

	g, err := git.New(initialWD)
	if err != nil {
		return err
	}

	bcPath := f.Config
	if filepath.IsAbs(bcPath) {
		bcPath, err = filepath.Rel(initialWD, bcPath)
		if err != nil {
			return err
		}
	}

	bc, bcData, err := config.Load(bcPath)
	if err != nil {
		return err
	}

	if f.Debug {
		fmt.Printf("loaded config: %#v\n", *bc)
		fmt.Printf(".Spec.WithBuildInstructions: %#v\n", bc.Spec.WithBuildInstructions)
		for i, variant := range bc.Spec.Variants {
			fmt.Printf(".Spec.Vairiants[%d]: {Name:%q, With:%#v}\n", i, variant.Name, variant.With)
		}
	}

	// TODO:
	// - [x] new tagging convetion
	// - [ ] implement metadata labels
	// - [x] store config as loaded from disk
	// - [x] rebuilder must check all variants
	// - [x] compose multi-target recipe directly from the config
	//    - [x] there should be just one invocation of bake
	// - [ ] TODO(post-mvp) --repo-config for repo-wide config
	// - [ ] write exact image names at the end of the build
	//    - [ ] as plain text summary to stdout
	//    - [ ] as JSON/YAML
	//    - [ ] (post-mvp) as plain text file with custom formatting
	//    - [ ] (post-mvp) lookup digests and include them in
	// - [ ] (post-mvp) define index image schema and implement it
	// - [ ] (post-mvp) implement some usefull cheks
	//    - [ ] presence of Dockerfile.dockerignore in the same direcory

	ir := &recipe.ImagineRecipe{
		Push:      f.Push,
		Export:    f.Export,
		Platforms: f.Platforms,
		WorkDir:   initialWD,
		BuildSpec: &bc.Spec,
	}

	ir.Git.Git = g

	ir.Git.BaseBranch = f.UpstreamBranch
	ir.Git.BranchedOffSuffix = "dev"    // TODO(post-mvp): make this a flag and repo config field
	ir.Git.WorkInProgressSuffix = "wip" // TODO(post-mvp): make this a repo and repo config field

	ir.Config.Data = bcData
	ir.Config.Path = bcPath

	m, err := ir.ToBakeManifest(f.Registries...)
	if err != nil {
		return err
	}

	rb := rebuilder.Rebuilder{
		RegistryAPI: &registry.Registry{},
	}

	fmt.Printf("current tags: %s", strings.Join(m.RegistryTags(), ", "))

	rebuild, reason, err := rb.ShouldRebuild(m)
	if err != nil {
		return err
	}
	if f.Export {
		rebuild = true
		reason = "forcing image rebuild due to export option being set"
	}
	if f.Force {
		rebuild = true
		reason = "forcing image rebuild due to force option being set"
	}
	if !rebuild {
		fmt.Println("no need to rebuild")
		return nil
	}
	fmt.Println(reason)

	manifest, cleanup, err := ir.WriteManifest(stateDirPath, f.Registries...)
	if err != nil {
		return err
	}

	if f.Debug {
		fmt.Printf("writen manifest %v\n", manifest)
	}

	bx := buildx.New(stateDirPath)
	bx.Debug = f.Debug
	bx.Platforms = f.Platforms

	if err := bx.InitBuilder(f.Builder); err != nil {
		return err
	}

	if err := bx.Bake(manifest); err != nil {
		return err
	}
	if !f.Debug {
		cleanup()
	} else {
		fmt.Printf("keeping %q for debugging\n", manifest)
	}
	return nil
}
