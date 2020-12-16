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
