package config

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"sigs.k8s.io/yaml"
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

	Variants []VariantSummary `json:"images"`
}

type VariantSummary struct {
	Name         *string  `json:"name"`
	Digest       *string  `json:"digest"`
	RegistryRefs []string `json:"registryRefs"`
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
		APIVersion: apiVersion,
		Kind:       buildSummaryKind,
		Name:       name,
	}
}

func (s *BuildSummary) WriteText(w io.Writer) error {
	var err error
	if _, err = fmt.Fprintln(w, "built refs:"); err != nil {
		return err
	}
	for _, variant := range s.Variants {
		if variant.Name == nil {
			_, err = fmt.Fprintf(w, "%s:\n", s.Name)
		} else {
			_, err = fmt.Fprintf(w, "%s (%s):\n", s.Name, *variant.Name)
		}
		if err != nil {
			return err
		}
		for _, ref := range variant.RegistryRefs {
			if _, err = fmt.Fprintf(w, "- %s@%s\n", ref, *variant.Digest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *BuildSummary) WriteLines(w io.Writer) error {
	var err error
	for _, variant := range s.Variants {
		for _, ref := range variant.RegistryRefs {
			if variant.Name == nil {
				_, err = fmt.Fprintf(w, "%s,,%s@%s\n", s.Name, ref, *variant.Digest)
			} else {
				_, err = fmt.Fprintf(w, "%s,%s,%s@%s\n", s.Name, *variant.Name, ref, *variant.Digest)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *BuildSummary) WriteJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(s)
}

func (s *BuildSummary) WriteYAML(w io.Writer) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
