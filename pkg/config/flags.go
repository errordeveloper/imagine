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
	Config         string
	Registries     []string // TODO(post-mvp): make this a repo config field
	UpstreamBranch string   // TODO(post-mvp): make this a repo config field
	WithoutSuffix  bool
}

type CommonFlags struct {
	*BasicFlags

	Push, Export bool
	Platforms    []string
}

func (f *BasicFlags) Register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Config, "config", "", "path to build config file")
	cmd.MarkFlagRequired("config")

	cmd.Flags().StringArrayVar(&f.Registries, "registry", []string{}, "registry prefixes to use for tags")

	cmd.Flags().StringVar(&f.UpstreamBranch, "upstream-branch", defaultUpstreamBranch, "upstream branch of the repository")

	cmd.Flags().BoolVar(&f.WithoutSuffix, "without-tag-suffix", false, "whether to exclude '-dev' and '-wip' suffix from image tags")
}

func (f *CommonFlags) Register(cmd *cobra.Command) {
	f.BasicFlags = &BasicFlags{}
	f.BasicFlags.Register(cmd)

	cmd.Flags().BoolVar(&f.Push, "push", false, "whether to push images to registries or not (if any registries are given)")

	cmd.Flags().BoolVar(&f.Export, "export", false, "whether to export images to an OCI tarball 'image-<name>.oci'")

	cmd.Flags().StringArrayVar(&f.Platforms, "platform", []string{defaultPlatform}, "platforms to target")
}
