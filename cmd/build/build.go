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
	Push       bool
	Export     bool

	Builder string
	Force   bool
	Cleanup bool
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

	cmd.Flags().StringVar(&flags.Builder, "builder", "", "name of buildx builder")
	cmd.MarkFlagRequired("builder")

	cmd.Flags().StringVar(&flags.Name, "name", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVar(&flags.Dir, "base", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVar(&flags.Registries, "registry", []string{}, "registry prefixes to use for tags")

	cmd.Flags().BoolVar(&flags.Root, "root", false, "where to use repo root as build context instead of base direcory")
	cmd.Flags().BoolVar(&flags.Test, "test", false, "whether to test image first (depends on 'test' build stage being defined)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuild the image")
	cmd.Flags().BoolVar(&flags.Cleanup, "cleanup", false, "cleanup generated manifest file")

	cmd.Flags().BoolVar(&flags.Push, "push", false, "whether to push image to registries or not (if any registries are given)")
	cmd.Flags().BoolVar(&flags.Export, "export", false, "whether to export the image to an OCI tarball 'image-<name>.oci'")

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
		Push:     f.Push,
		Export:   f.Export,
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
