package config

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

func Load(path string) (*BuildConfig, string, error) {
	obj := &BuildConfig{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	if err := yaml.Unmarshal(data, obj); err != nil {
		return nil, "", err
	}

	if err := obj.ApplyDefaultsAndValidate(); err != nil {
		return nil, "", err
	}

	return obj, base64.StdEncoding.EncodeToString(data), nil
}

func fieldMustBeSetErr(filepath string) error {
	return fmt.Errorf("'%s' must be set", filepath)
}

func fieldValueInvalidErr(filepath, value string) error {
	return fmt.Errorf("'%s: %s' is not valid", filepath, value)
}

// TODO: write tests for this
func (o *BuildConfig) ApplyDefaultsAndValidate() error {
	if o.APIVersion != apiVersion {
		return fieldValueInvalidErr(".apiVersion", o.APIVersion)
	}

	if o.Kind != kind {
		return fieldValueInvalidErr(".kind", kind)
	}

	return o.Spec.ApplyDefaultsAndValidate()
}

func (o *BuildSpec) ApplyDefaultsAndValidate() error {
	if o.Name == "" {
		return fieldMustBeSetErr(".spec.name")
	}

	if (o.WithBuildInstructions == nil || o.Dir == nil) && len(o.Variants) == 0 {
		return fmt.Errorf("at least either '.spec.dir' or '.spec.variants' must be set")
	}

	if o.WithBuildInstructions != nil {
		if o.Dockerfile == nil {
			o.Dockerfile = &DockerfileBuildInstructions{}
		}
		if o.Dockerfile.Path == "" && o.Dockerfile.Body == "" {
			o.Dockerfile.Path = defaultDockerfile
		}

		if filepath.IsAbs(o.Dockerfile.Path) {
			return fmt.Errorf("absolute path in '.spec.dockerfile.path' is prohibited (%q)", o.Dockerfile.Path)
		}

		if strings.HasPrefix(o.Dockerfile.Path, "..") {
			return fmt.Errorf("'.spec.dockerfile.path' points outside of '.spec.dir' (%q) - you can try '.spec.dockerfile.body' instead", o.Dockerfile.Path)
		}

		if o.Dir == nil {
			return fieldMustBeSetErr(".spec.dir")
		}

		if o.Test == nil {
			o.Test = new(bool)
			*o.Test = false
		}

		if o.Target != nil && *o.Target == "" {
			return fieldMustBeSetErr(".spec.target")
		}

		if o.Untagged == nil {
			o.Untagged = new(bool)
			*o.Untagged = false
		}
	}

	if o.TagMode == "" {
		o.TagMode = "GitTreeHash"
	}

	for i, secret := range o.Secrets {
		if secret.Type == "" {
			o.Secrets[i].Type = "file"
		}

		if secret.Type != "file" {
			return fmt.Errorf("usupported '.spec.secrets[%d].type' (%q) - must be \"file\"", i, secret.Type)
		}

		if secret.ID != "" {
			return fieldMustBeSetErr(fmt.Sprintf(".spec.secrets[%d].id", i))
		}

		if secret.Source != "" {
			return fieldMustBeSetErr(fmt.Sprintf(".spec.secrets[%d].source", i))
		}
	}

	for i, variant := range o.Variants {
		p := fmt.Sprintf(".spec.variants[%d]", i)
		if variant.Name == "" {
			return fieldMustBeSetErr(p + ".name")
		}
		if variant.With == nil {
			variant.With = o.WithBuildInstructions
		} else {
			if variant.With.Dockerfile == nil ||
				(variant.With.Dockerfile.Path == "" && variant.With.Dockerfile.Body == "") {
				variant.With.Dockerfile = o.Dockerfile
			}

			if filepath.IsAbs(variant.With.Dockerfile.Path) {
				return fmt.Errorf("absolute path in '%s.dockerfile.path' is prohibited (%q)", p, variant.With.Dockerfile.Path)
			}

			if strings.HasPrefix(variant.With.Dockerfile.Path, "..") {
				return fmt.Errorf("'%s.dockerfile.path' points outside of '%s.dir' (%q) - you can try '%s.dockerfile.body' instead", p, p, variant.With.Dockerfile.Path, p)
			}

			if variant.With.Dir == nil {
				variant.With.Dir = o.Dir
			}

			for k, v := range o.Args {
				if _, ok := variant.With.Args[k]; !ok {
					variant.With.Args[k] = v
				}
			}

			if variant.With.Test == nil {
				variant.With.Test = o.Test
			}

			if variant.With.Target != nil && *variant.With.Target == "" {
				return fieldMustBeSetErr(p + ".target")
			}

			if variant.With.Target == nil {
				variant.With.Target = o.Target
			}

			if variant.With.Untagged == nil {
				variant.With.Untagged = o.Untagged
			}

			if len(variant.With.Secrets) == 0 {
				variant.With.Secrets = o.Secrets
			} else {
				for i, secret := range variant.With.Secrets {
					if secret.Type == "" {
						variant.With.Secrets[i].Type = "file"
					}

					if secret.Type != "file" {
						return fmt.Errorf("usupported '%s.secrets[%d].type' (%q) - must be \"file\"", p, i, secret.Type)
					}

					if secret.ID != "" {
						return fieldMustBeSetErr(fmt.Sprintf("%s.secrets[%d].id", p, i))
					}

					if secret.Source != "" {
						return fieldMustBeSetErr(fmt.Sprintf("%s.secrets[%d].source", p, i))
					}
				}
			}
		}
	}

	return nil
}
