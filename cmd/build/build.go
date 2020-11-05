package build

import (
	"fmt"
	"os"
	"path/filepath"

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
	Force   bool
	Debug   bool

	Args map[string]string
}

func BuildCmd() *cobra.Command {

	flags := &Flags{
		CommonFlags: &config.CommonFlags{},
	}

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

	flags.CommonFlags.Register(cmd)

	cmd.Flags().StringVar(&flags.Builder, "builder", "", "name of buildx builder")
	cmd.MarkFlagRequired("builder")

	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuild the image")
	cmd.Flags().BoolVar(&flags.Debug, "debug", false, "print debuging info and keep generated buildx manifest file")

	cmd.Flags().StringToStringVar(&flags.Args, "args", nil, "build args")

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

	g, err := git.New(initialWD)
	if err != nil {
		return err
	}

	ir := &recipe.ImagineRecipe{
		Name:            f.Name,
		HasTests:        f.Test,
		Push:            f.Push,
		Export:          f.Export,
		Platforms:       f.Platforms,
		Args:            f.Args,
		BaseDir:         initialWD,
		CustomTagSuffix: f.CustomTagSuffix,
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

	rb := rebuilder.Rebuilder{
		RegistryAPI: &registry.Registry{},
	}

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
	filename := filepath.Join(initialWD, fmt.Sprintf("buildx-%s.json", f.Name))
	if f.Debug {
		fmt.Printf("writing manifest to %q\n", filename)
	}
	if err := m.WriteFile(filename); err != nil {
		return err
	}

	bx := buildx.Buildx{
		Builder: f.Builder,
	}
	if err := bx.Bake(filename); err != nil {
		return err
	}
	if !f.Debug {
		if err := os.RemoveAll(filename); err != nil {
			return err
		}
	} else {
		fmt.Printf("keeping %q for debugging\n", filename)
	}
	return nil
}
