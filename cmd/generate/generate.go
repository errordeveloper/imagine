package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
)

type Flags struct {
	Name       string
	Dir        string
	Registries []string
	Root       bool
	Test       bool
	Push       bool
	Export     bool
}

const (
	baseBranch = "origin/master"
	dockerfile = "Dockerfile"
)

func GenerateCmd() *cobra.Command {

	flags := &Flags{}

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

	cmd.Flags().StringVar(&flags.Name, "name", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVar(&flags.Dir, "base", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVar(&flags.Registries, "registry", []string{}, "registry prefixes to use for tags")

	cmd.Flags().BoolVar(&flags.Root, "root", false, "where to use repo root as build context instead of base direcory")
	cmd.Flags().BoolVar(&flags.Test, "test", false, "whether to test image first (depends on 'test' build stage being defined)")

	cmd.Flags().BoolVar(&flags.Push, "push", false, "whether to push image to registries or not (if any registries are given)")
	cmd.Flags().BoolVar(&flags.Export, "export", false, "whether to export the image to an OCI tarball 'image-<name>.oci'")

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

			RelativeDockerfilePath: filepath.Join(f.Dir, dockerfile),

			WithoutSuffix: true,       // TODO: add a flag
			BaseBranch:    baseBranch, // TODO: add a flag
		}
	} else {
		ir.Scope = &recipe.ImageScopeSubDir{
			Git:     g,
			BaseDir: initialWD,

			RelativeImageDirPath: f.Dir,
			Dockerfile:           dockerfile,

			WithoutSuffix: true,       // TODO: add a flag
			BaseBranch:    baseBranch, // TODO: add a flag
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
