package image

/*
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
	*config.BasicFlags
}

func ImageCmd() *cobra.Command {

	flags := &Flags{
		BasicFlags: &config.BasicFlags{},
	}

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

	flags.BasicFlags.Register(cmd)

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
		Name:            f.Name,
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

	tags, err := ir.RegistryTags(f.Registries...)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		fmt.Println(tag)
	}
	return nil
}
*/
