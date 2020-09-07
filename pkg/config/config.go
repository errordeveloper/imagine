package config

import (
	"github.com/spf13/cobra"
)

const (
	defaultUpstreamBranch = "origin/master"
	defaultDockerfile     = "Dockerfile"
	defaultPlatform       = "linux/amd64"
)

type BasicFlags struct {
	Name           string
	Dir            string
	Registries     []string
	Root           bool
	WithoutSuffix  bool
	UpstreamBranch string
	Dockerfile     string
}

type CommonFlags struct {
	*BasicFlags

	Test      bool
	Push      bool
	Export    bool
	Platforms []string
}

func (f *BasicFlags) Register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Name, "name", "", "name of the image")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVar(&f.Dir, "base", "", "base directory of image")
	cmd.MarkFlagRequired("base")

	cmd.Flags().StringArrayVar(&f.Registries, "registry", []string{}, "registry prefixes to use for tags")

	cmd.Flags().BoolVar(&f.Root, "root", false, "whether to use repo root as build context instead of base direcory")

	cmd.Flags().BoolVar(&f.WithoutSuffix, "without-tag-suffix", false, "whether to exclude '-dev' and '-wip' suffix from image tags")

	cmd.Flags().StringVar(&f.UpstreamBranch, "upstream-branch", defaultUpstreamBranch, "upstream branch of the repository")

	cmd.Flags().StringVar(&f.Dockerfile, "dockerfile", defaultDockerfile, "base directory of the image")
}

func (f *CommonFlags) Register(cmd *cobra.Command) {
	f.BasicFlags = &BasicFlags{}
	f.BasicFlags.Register(cmd)

	cmd.Flags().BoolVar(&f.Test, "test", false, "whether to test image first (depends on 'test' build stage being defined)")

	cmd.Flags().BoolVar(&f.Push, "push", false, "whether to push image to registries or not (if any registries are given)")

	cmd.Flags().BoolVar(&f.Export, "export", false, "whether to export the image to an OCI tarball 'image-<name>.oci'")

	cmd.Flags().StringArrayVar(&f.Platforms, "platform", []string{defaultPlatform}, "platforms to target")
}
