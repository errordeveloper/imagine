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
	*config.CommonFlags

	Builder string

	Force bool
}

const (
	stateDir = ".imagine" // TODO(post-mvp): make this repo config field

	defaultPlatform       = "linux/amd64"
	defaultUpstreamBranch = "origin/master"
)

func BuildCmd() *cobra.Command {

	flags := &Flags{
		CommonFlags: &config.CommonFlags{},
	}

	cmd := &cobra.Command{
		Use:   "build",
		Short: "build and test image(s) using a config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitBuildCmd(cmd); err != nil {
				return err
			}
			return flags.RunBuildCmd()
		},
	}

	flags.Register(cmd)

	cmd.Flags().StringVar(&flags.Builder, "builder", "", "use a global buildx builder instead of creating one")

	// TODO(post-mvp): --no-cache for rebuilding without cache
	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuilding")

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

	repo, err := git.New(initialWD)
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
	// - [ ] (post-mvp) improve TagMode
	//    - [ ] expose various options for treating multiple tags
	//    - [ ] enable tags on release branches (either as an option
	//            or by documenting that upstream branch needs to
	//  		  change in repo config on a release branch)
	//    - [ ] enable semver tags along with tree hash tags by default
	//    - [ ] enable non-semver tags
	// - [ ] (post-mvp) should unnamed/main variants be allowed along with named variants?

	ir := &recipe.ImagineRecipe{
		Push:      f.Push,
		Export:    f.Export,
		Platforms: f.Platforms,
		WorkDir:   initialWD,
		BuildSpec: &bc.Spec,
	}

	ir.Git.Git = repo
	ir.Git.BaseBranch = f.UpstreamBranch
	if !f.WithoutSuffix {
		ir.Git.BranchedOffSuffix = "-dev"    // TODO(post-mvp): make this a flag and a repo config field
		ir.Git.WorkInProgressSuffix = "-wip" // TODO(post-mvp): make this a repo and a repo config field
	}

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
