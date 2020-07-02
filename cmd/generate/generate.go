package generate

import (
	"fmt"
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
}

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

	cmd.Flags().StringVarP(&flags.Name, "name", "", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVarP(&flags.Dir, "base", "", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVarP(&flags.Registries, "registry", "", []string{}, "registry prefixes to use for tags")
	cmd.MarkFlagRequired("registry")

	cmd.Flags().BoolVarP(&flags.Root, "root", "", false, "where to use repo root as build context instead of base direcory")
	cmd.Flags().BoolVarP(&flags.Test, "test", "", false, "whether to test image first (depends on 'test' build stage being defined)")

	return cmd
}

func (f *Flags) InitGenerateCmd(cmd *cobra.Command) error {
	return nil
}

func (f *Flags) RunGenerateCmd() error {
	g, err := git.NewFromCWD()
	if err != nil {
		return err
	}

	ir := &recipe.ImagineRecipe{
		Name:     f.Name,
		HasTests: f.Test,
	}

	if f.Root {
		ir.Scope = &recipe.ImageScopeRootDir{
			Git:                    g,
			RootDir:                g.TopLevel,
			WithoutSuffix:          true, // TODO: add a flag
			RelativeDockerfilePath: filepath.Join(f.Dir, "Dockerfile"),
		}
	} else {
		ir.Scope = &recipe.ImageScopeSubDir{
			Git:                  g,
			RootDir:              g.TopLevel,
			RelativeImageDirPath: f.Dir,
			WithoutSuffix:        true, // TODO: add a flag
			Dockerfile:           "Dockerfile",
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
