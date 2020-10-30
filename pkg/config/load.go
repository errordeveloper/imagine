package config

import (
	"fmt"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

func Load(path string) (*BuildConfig, error) {
	obj := &BuildConfig{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, obj); err != nil {
		return nil, err
	}

	if err := obj.ApplyDefaultsAndValidate(); err != nil {
		return nil, err
	}

	return obj, nil
}

func fieldMustBeSetErr(filepath string) error {
	return fmt.Errorf("'%s' must be set", filepath)
}

func fieldValueInvalidErr(filepath, value string) error {
	return fmt.Errorf("'%s: %s' is not valid", filepath, value)
}

func (o *BuildConfig) ApplyDefaultsAndValidate() error {
	if o.APIVersion != apiVersion {
		return fieldValueInvalidErr(".apiVersion", o.APIVersion)
	}

	if o.Kind != kind {
		return fieldValueInvalidErr(".kind", kind)
	}

	if o.Spec.Name == "" {
		return fieldMustBeSetErr(".spec.name")
	}

	if (o.Spec.WithBuildInstructions == nil || o.Spec.Dir == "") && len(o.Spec.Variants) == 0 {
		return fmt.Errorf("at least either '.spec.dir' or '.spec.variants' must be set")
	}

	if o.Spec.WithBuildInstructions != nil {
		if o.Spec.Dockerfile == nil {
			o.Spec.Dockerfile = &DockerfileBuildInstructions{}
		}
		if o.Spec.Dockerfile.Path == "" && o.Spec.Dockerfile.Body == "" {
			o.Spec.Dockerfile.Path = defaultDockerfile
		}

		if o.Spec.Dir == "" {
			return fieldMustBeSetErr(".spec.dir")
		}

		if o.Spec.Test == nil {
			o.Spec.Test = new(bool)
			*o.Spec.Test = false
		}
	}

	for i, variant := range o.Spec.Variants {
		if variant.Name == "" {
			return fieldMustBeSetErr(fmt.Sprintf(".spec.variants[%d]", i))
		}
		if variant.With == nil {
			variant.With = o.Spec.WithBuildInstructions
		} else {
			if variant.With.Dockerfile.Path == "" && variant.With.Dockerfile.Body == "" {
				variant.With.Dockerfile = o.Spec.Dockerfile
			}
			if variant.With.Dir == "" {
				variant.With.Dir = o.Spec.Dir
			}

			for k, v := range o.Spec.Args {
				if _, ok := variant.With.Args[k]; !ok {
					variant.With.Args[k] = v
				}
			}

			if variant.With.Test == nil {
				variant.With.Test = o.Spec.Test
			}
		}
	}

	return nil
}
