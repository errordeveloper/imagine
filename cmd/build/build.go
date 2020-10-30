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
	Builder string

	Platforms, Registries []string
	UpstreamBranch        string

	Push, Export, Force, Debug bool

	Config string
}

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

	cmd.Flags().StringVar(&flags.Config, "config", "", "path to build config file")
	cmd.MarkFlagRequired("config")

	cmd.Flags().StringArrayVar(&flags.Platforms, "platform", []string{"linux/amd64"}, "platforms to target")
	cmd.Flags().StringArrayVar(&flags.Registries, "registry", []string{}, "registry prefixes to use for tags")
	cmd.Flags().StringVar(&flags.UpstreamBranch, "upstream-branch", "origin/master", "upstream branch of the repository")

	cmd.Flags().BoolVar(&flags.Push, "push", false, "whether to push image to registries or not (if any registries are given)")
	cmd.Flags().BoolVar(&flags.Export, "export", false, "whether to export the image to an OCI tarball 'image-<name>.oci'")

	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuild the image")
	cmd.Flags().BoolVar(&flags.Debug, "debug", false, "print debuging info and keep generated buildx manifest file")

	// TODO:
	// - flag to write summary (tags and variants) to a file
	//    - json
	//    - plain text
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

	bc, err := config.Load(f.Config)
	if err != nil {
		return err
	}

	if f.Debug {

		fmt.Printf("loaded config: %#v", *bc)

	}

	// TODO:
	// - new tagging convetion
	// - implement metadata labels
	// - store config as load from disk
	// - rebuilder must check all variants
	// - compose multi-target recipe directly from the config
	//    - there should be just one invocation of bake
	// - write exact image names at the end of the build

	if len(bc.Spec.Variants) == 0 {
		if err := f.doBuild(initialWD, bc.Spec.Name, "", bc.Spec.WithBuildInstructions, g); err != nil {
			return err
		}
	} else {
		for _, v := range bc.Spec.Variants {
			if err := f.doBuild(initialWD, bc.Spec.Name, v.Name, bc.Spec.WithBuildInstructions, g); err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *Flags) doBuild(wd, name, suffix string, instructions *config.WithBuildInstructions, g *git.GitRepo) error {

	ir := &recipe.ImagineRecipe{
		Name:            name,
		HasTests:        *instructions.Test,
		Push:            f.Push,
		Export:          f.Export,
		Platforms:       f.Platforms,
		Args:            instructions.Args,
		BaseDir:         wd,
		CustomTagSuffix: suffix,
	}

	ir.Scope = &recipe.ImageScopeSubDir{
		Git:     g,
		BaseDir: wd,

		RelativeImageDirPath: instructions.Dir,
		Dockerfile:           instructions.Dockerfile.Path,

		BaseBranch: f.UpstreamBranch,
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
	filename := filepath.Join(wd, fmt.Sprintf("buildx-%s.json", name))
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
