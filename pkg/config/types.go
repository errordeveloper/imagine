package config

const (
	apiVersion = "v1alpha1"
	kind       = "ImagineBuildConfig"
)

type BuildConfig struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Spec BuildSpec `json:"spec"`
	// TODO:
	// - taging modes & behaviour
	//   - custom dev & wip suffixes
}

type BuildSpec struct {
	Name                   string `json:"name"`
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

type DockerfileBuildInstructions struct {
	Path string `json:"path"`
	Body string `json:"body"` // TODO: this should be written out to a temp file
}
