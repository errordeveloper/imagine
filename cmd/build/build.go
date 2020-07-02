package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/buildx"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/rebuilder"
	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"
)

type Flags struct {
	Name       string
	Dir        string
	Registries []string
	Root       bool
	Test       bool
	Builder    string
	Force      bool
	Cleanup    bool
}

const (
	baseBranch = "origin/master"
	dockerfile = "Dockerfile"
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

	cmd.Flags().StringVarP(&flags.Builder, "builder", "", "", "name of buildx builder")
	cmd.MarkFlagRequired("builder")

	cmd.Flags().StringVarP(&flags.Name, "name", "", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVarP(&flags.Dir, "base", "", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVarP(&flags.Registries, "registry", "", []string{}, "registry prefixes to use for tags")
	cmd.MarkFlagRequired("registry")

	cmd.Flags().BoolVarP(&flags.Root, "root", "", false, "where to use repo root as build context instead of base direcory")
	cmd.Flags().BoolVarP(&flags.Test, "test", "", false, "whether to test image first (depends on 'test' build stage being defined)")
	cmd.Flags().BoolVarP(&flags.Force, "force", "", false, "force rebuild the image")
	cmd.Flags().BoolVarP(&flags.Cleanup, "cleanup", "", false, "cleanup generated manifest file")

	return cmd
}

func (f *Flags) InitBuildCmd(cmd *cobra.Command) error {
	return nil
}

func (f *Flags) RunBuildCmd() error {
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
			Git:     g,
			RootDir: g.TopLevel,

			RelativeDockerfilePath: filepath.Join(f.Dir, dockerfile),

			WithoutSuffix: true,       // TODO: add a flag
			BaseBranch:    baseBranch, // TODO: add a flag
		}
	} else {
		ir.Scope = &recipe.ImageScopeSubDir{
			Git:     g,
			RootDir: g.TopLevel,

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

	rb := rebuilder.Rebuilder{
		RegistryAPI: &registry.Registry{},
	}

	rebuild, reason, err := rb.ShouldRebuild(m)
	if err != nil {
		return err
	}
	if f.Force {
		rebuild = true
		reason = "forcing image rebuild"
	}
	if !rebuild {
		fmt.Println("no need to rebuild")
		return nil
	}
	fmt.Println(reason)

	filename := fmt.Sprintf("buildx-%s.json", f.Name)
	fmt.Printf("writing manifest to %q\n", filename)
	if err := m.WriteFile(filename); err != nil {
		return err
	}

	bx := buildx.Buildx{
		Builder: f.Builder,
	}
	if err := bx.Bake(filename); err != nil {
		return err
	}
	if f.Cleanup {
		fmt.Printf("removing %q\n", filename)
		if err := os.RemoveAll(filename); err != nil {
			return err
		}
	}
	return nil
}
