package config

import (
	"fmt"
	"io"
	"path/filepath"
)

const (
	apiVersion       = "v1alpha1"
	buildConfigKind  = "ImagineBuildConfig"
	buildSummaryKind = "ImagineBuildSummary"
)

type BuildConfig struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Spec BuildSpec `json:"spec"`
}

type BuildSpec struct {
	Name    string `json:"name"`
	TagMode string `json:"tagMode"`

	*WithBuildInstructions `json:",inline"`

	Variants []BuildVariant `json:"variants"`
}

type BuildVariant struct {
	Name string                 `json:"name"`
	With *WithBuildInstructions `json:"with"`
}

type WithBuildInstructions struct {
	Dir  *string           `json:"dir"`
	Args map[string]string `json:"args"`
	Test *bool             `json:"test"`

	Secrets []Secret `json:"secrets"`

	Dockerfile *DockerfileBuildInstructions `json:"dockerfile"`

	Target   *string `json:"target"`
	Untagged *bool   `json:"untagged"`

	Labels map[string]string `json:"labels"`
}

type Secret struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type DockerfileBuildInstructions struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

type BuildSummary struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Name string `json:"name"`

	Images []ImageSummary
}

type ImageSummary struct {
	VarianName   *string
	Digest       *string
	RegistryRefs []string
}

func (s Secret) String() string {
	return fmt.Sprintf("id=%s,type=%s,source=%s", s.ID, s.Type, s.Source)
}

func (i *WithBuildInstructions) ContextPath(workDir string) string {
	return filepath.Join(workDir, *i.Dir)
}

func (i *WithBuildInstructions) DockerfilePath(workDir string) string {
	if i.Dockerfile.Path == "" {
		return ""
	}
	if filepath.IsAbs(i.Dockerfile.Path) {
		// temp file is used for inline dockerfile
		return i.Dockerfile.Path
	}
	return filepath.Join(i.ContextPath(workDir), i.Dockerfile.Path)
}

func NewBuildSummary(name string) *BuildSummary {
	return &BuildSummary{
		APIVersion: buildSummaryKind,
		Kind:       apiVersion,
		Name:       name,
	}
}

func (s *BuildSummary) WriteText(w io.Writer) error {
	fmt.Fprintln(w, "built refs:")
	for _, image := range s.Images {
		for _, ref := range image.RegistryRefs {
			if _, err := fmt.Fprintf(w, "- %s@%s\n", ref, *image.Digest); err != nil {
				return err
			}
		}
	}
	return nil
}
