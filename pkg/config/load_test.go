package config_test

import (
	"testing"

	"sigs.k8s.io/yaml"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/imagine/pkg/config"
)

func TestBasicSample(t *testing.T) {
	g := NewGomegaWithT(t)


	sample := `{
		"kind": "ImagineBuildConfig",
		"apiVersion": "v1alpha1",
		"spec": {
			"name": "imagine-alpine-example",
			"dir": "./examples/alpine"
		}
	}`

	obj := &BuildConfig{}

	g.Expect(yaml.Unmarshal([]byte(sample), obj)).To(Succeed())

	g.Expect(obj.ApplyDefaultsAndValidate()).To(Succeed())
	g.Expect(obj.Spec.Name).To(Equal("imagine-alpine-example"))
	g.Expect(*obj.Spec.Dir).To(Equal("./examples/alpine"))

	g.Expect(obj.Spec.Secrets).To(HaveLen(0))
	g.Expect(obj.Spec.Variants).To(HaveLen(0))

	g.Expect(obj.Spec.Dockerfile.Path).To(Equal("Dockerfile"))
	g.Expect(obj.Spec.Dockerfile.Body).To(BeEmpty())
	
	g.Expect(*obj.Spec.Test).To(BeFalse())
	g.Expect(*obj.Spec.Untagged).To(BeFalse())
	g.Expect(obj.Spec.TagMode).To(Equal("GitTreeHash"))
}


func TestErrorCases(t *testing.T) {
	g := NewGomegaWithT(t)

	samples := []struct{
		errMessage string
		configData string
	}{
		{
			errMessage: `'.apiVersion' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig"
			}`,
		},
		{
			errMessage: `'.kind' cannot be an empty string`,
			configData: `{
				"apiVersion": "v1alpha1"
			}`,
		},
		{
			errMessage: `'.apiVersion: "v1alpha"' is not valid, should be '.apiVersion: "v1alpha1"'`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha"
			}`,
		},
		{
			errMessage: `'.apiVersion: "v1alpha"' is not valid, should be '.apiVersion: "v1alpha1"'`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha"
			}`,
		},
		{
			errMessage: `'.spec.name' must be set`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1"
			}`,
		},
		{
			errMessage: "at least '.spec.dir' or '.spec.variants' must be set",
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example"
				}
			}`,
		},
		{
			errMessage: `absolute path in '.spec.dockerfile.path: "/src/Dockerfile"' is prohibited`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"dockerfile": { "path": "/src/Dockerfile" }
				}
			}`,
		},
		{
			errMessage: "at least '.spec.dir' or '.spec.variants' must be set",
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dockerfile": { "path": "Dockerfile" }
				}
			}`,
		},
		{
			errMessage: "at least '.spec.dir' or '.spec.variants' must be set",
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dockerfile": { "path": "Dockerfile" },
					"varians": []
				}
			}`,
		},
		{
			errMessage: "'.spec.target' cannot be an empty string",
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dockerfile": { "path": "Dockerfile" },
					"dir": "/src",
					"target": ""
				}
			}`,
		},
		{
			errMessage: `'.spec.dockerfile.path: "../Dockerfile"' points outside of '.spec.dir: "/src"' - you can try '.spec.dockerfile.body' instead`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"dockerfile": { "path": "../Dockerfile" }
				}
			}`,
		},
		{
			errMessage: `usupported '.spec.secrets[0].type: "foo"' - must be "file"`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"secrets": [{ "type": "foo" }]
				}
			}`,
		},
		{
			errMessage: `'.spec.secrets[0].id' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"secrets": [{ "type": "file" }]
				}
			}`,
		},
		{
			errMessage: `'.spec.secrets[0].source' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"secrets": [{ "id": "foo" }]
				}
			}`,
		},
		{
			errMessage: `'.spec.variants[1].name' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "example",
					"dir": "/src",
					"variants": [
						{
							"name": "imagine-alpine-example",
						},
						{
						
						}
					]
				}
			}`,
		},
		{
			errMessage: `absolute path in '.spec.variants[0].with.dockerfile.path: "/foo/Dockerfile"' is prohibited`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "dockerfile": { "path": "/foo/Dockerfile" } }
					}]
				}
			}`,
		},
		{
			errMessage: `'.spec.variants[0].with.dockerfile.path: "../Dockerfile"' points outside of '.spec.variants[0].with.dir: "/src"' - you can try '.spec.variants[0].with.dockerfile.body' instead`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "dockerfile": { "path": "../Dockerfile" } }
					}]
				}
			}`,
		},
		{
			errMessage: `'.spec.variants[0].with.target' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "target": "" }
					}]
				}
			}`,
		},
		{
			errMessage: `usupported '.spec.variants[0].with.secrets[0].type: "foo"' - must be "file"`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "secrets": [{ "type": "foo" }] }
					}]
				}
			}`,
		},
		{
			errMessage: `'.spec.variants[0].with.secrets[0].id' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "secrets": [{ "type": "file" }] }
					}]
				}
			}`,
		},
		{
			errMessage: `'.spec.variants[0].with.secrets[0].source' cannot be an empty string`,
			configData: `{
				"kind": "ImagineBuildConfig",
				"apiVersion": "v1alpha1",
				"spec": {
					"name": "imagine-alpine-example",
					"dir": "/src",
					"variants": [{
						"name": "imagine-alpine-example",
						"with": { "secrets": [{ "id": "foo" }] }
					}]
				}
			}`,
		},
	}

	for _, sample := range samples {
		obj := &BuildConfig{}

		g.Expect(yaml.Unmarshal([]byte(sample.configData), obj)).To(Succeed())

		err := obj.ApplyDefaultsAndValidate()
		g.Expect(err).NotTo(Succeed())
		g.Expect(err.Error()).To(Equal(sample.errMessage))

	}
}
