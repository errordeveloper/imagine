package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
)

type Flags struct {
	*config.CommonFlags

	Output string
}

const (
	stateDir = ".imagine" // TODO(post-mvp): make this repo config field

	defaultPlatform       = "linux/amd64"
	defaultUpstreamBranch = "origin/master"
)

func GenerateCmd() *cobra.Command {

	flags := &Flags{
		CommonFlags: &config.CommonFlags{},
	}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "generate bake manifest based on a config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitGenerateCmd(cmd); err != nil {
				return err
			}
			return flags.RunGenerateCmd()
		},
	}

	flags.Register(cmd)

	cmd.Flags().StringVar(&flags.Output, "output", "", "write generated manifest to a specific file")

	return cmd
}

func (f *Flags) InitGenerateCmd(cmd *cobra.Command) error {
	return f.Validate()
}

func (f *Flags) RunGenerateCmd() error {
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

	fmt.Printf("current registry refs: %s", strings.Join(m.RegistryRefs(), ", "))

	if f.Output != "" {
		if err := m.WriteFile(f.Output); err != nil {
			return err
		}
	} else {
		manifest, _, err := ir.WriteManifest(stateDirPath, f.Registries...)
		if err != nil {
			return err
		}
		f.Output = manifest
	}

	fmt.Printf("writen manifest %v\n", f.Output)

	return nil
}
