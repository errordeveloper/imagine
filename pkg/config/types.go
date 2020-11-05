package config

import (
	"path/filepath"
)

const (
	apiVersion = "v1alpha1"
	kind       = "ImagineBuildConfig"
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
	Dir  string            `json:"dir"`
	Args map[string]string `json:"args"`
	Test *bool             `json:"test"`

	Dockerfile *DockerfileBuildInstructions `json:"dockerfile"`
}

func (i *WithBuildInstructions) ContextPath(workDir string) string {
	return filepath.Join(workDir, i.Dir)
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

type DockerfileBuildInstructions struct {
	Path string `json:"path"`
	Body string `json:"body"`
}
