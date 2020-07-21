package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/recipe"
)

type Flags struct {
	Name          string
	Dir           string
	Registries    []string
	Root          bool
	Test          bool
	Push          bool
	Export        bool
	WithoutSuffix bool
}

const (
	baseBranch = "origin/master"
	dockerfile = "Dockerfile"
)

func ImageCmd() *cobra.Command {

	flags := &Flags{}

	cmd := &cobra.Command{
		Use: "image",
		//Args: cobra.NoArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitImageCmd(cmd); err != nil {
				return err
			}
			return flags.RunImageCmd()
		},
	}

	cmd.Flags().StringVar(&flags.Name, "name", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVar(&flags.Dir, "base", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVar(&flags.Registries, "registry", []string{}, "registry prefixes to use for tags")

	cmd.Flags().BoolVar(&flags.Root, "root", false, "where to use repo root as build context instead of base direcory")

	cmd.Flags().BoolVar(&flags.WithoutSuffix, "without-tag-suffix", false, "whether to exclude '-dev' and '-wip' suffix from image tags")

	return cmd
}

func (f *Flags) InitImageCmd(cmd *cobra.Command) error {
	return nil
}

func (f *Flags) RunImageCmd() error {
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
	}

	if f.Root {
		ir.Scope = &recipe.ImageScopeRootDir{
			Git:     g,
			BaseDir: initialWD,

			RelativeDockerfilePath: filepath.Join(f.Dir, dockerfile),

			WithoutSuffix: f.WithoutSuffix,
			BaseBranch:    baseBranch, // TODO: add a flag
		}
	} else {
		ir.Scope = &recipe.ImageScopeSubDir{
			Git:     g,
			BaseDir: initialWD,

			RelativeImageDirPath: f.Dir,
			Dockerfile:           dockerfile,

			WithoutSuffix: f.WithoutSuffix,
			BaseBranch:    baseBranch, // TODO: add a flag
		}
	}

	tags, err := ir.RegistryTags(f.Registries...)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		fmt.Println(tag)
	}
	return nil
}
