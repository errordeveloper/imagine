package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
)

type Flags struct {
	*config.CommonFlags
}

func GenerateCmd() *cobra.Command {

	flags := &Flags{
		CommonFlags: &config.CommonFlags{},
	}

	cmd := &cobra.Command{
		Use: "generate",
		//Args: cobra.NoArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitGenerateCmd(cmd); err != nil {
				return err
			}
			return flags.RunGenerateCmd()
		},
	}

	flags.CommonFlags.Register(cmd)

	return cmd
}

func (f *Flags) InitGenerateCmd(cmd *cobra.Command) error {
	return nil
}

func (f *Flags) RunGenerateCmd() error {
	initialWD, err := os.Getwd()
	if err != nil {
		return err
	}

	g, err := git.New(initialWD)
	if err != nil {
		return err
	}

	ir := &recipe.ImagineRecipe{
		Name:     f.Name,
		HasTests: f.Test,
		Push:     f.Push,
		Export:   f.Export,
		BaseDir:  initialWD,
	}

	if f.Root {
		ir.Scope = &recipe.ImageScopeRootDir{
			Git:     g,
			BaseDir: initialWD,

			RelativeDockerfilePath: filepath.Join(f.Dir, f.Dockerfile),

			WithoutSuffix: f.WithoutSuffix,
			BaseBranch:    f.UpstreamBranch,
		}
	} else {
		ir.Scope = &recipe.ImageScopeSubDir{
			Git:     g,
			BaseDir: initialWD,

			RelativeImageDirPath: f.Dir,
			Dockerfile:           f.Dockerfile,

			WithoutSuffix: f.WithoutSuffix,
			BaseBranch:    f.UpstreamBranch,
		}
	}

	// TODO implement usefull cheks:
	// - presence of Dockerfile.dockerignore in the same direcory

	m, err := ir.ToBakeManifest(f.Registries...)
	if err != nil {
		return err
	}
	js, err := m.ToJSON()
	if err != nil {
		return err
	}

	fmt.Println(js)

	return nil
}
