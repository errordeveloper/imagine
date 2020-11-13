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

		if filepath.IsAbs(o.Spec.Dockerfile.Path) {
			return fmt.Errorf("absolute path in '.spec.dockerfile.path' is prohibited (%q)", o.Spec.Dockerfile.Path)
		}

		if strings.HasPrefix(o.Spec.Dockerfile.Path, "..") {
			return fmt.Errorf("'.spec.dockerfile.path' points outside of '.spec.dir' (%q) - you can try '.spec.dockerfile.body' instead", o.Spec.Dockerfile.Path)
		}

		if o.Spec.Dir == "" {
			return fieldMustBeSetErr(".spec.dir")
		}

		if o.Spec.Test == nil {
			o.Spec.Test = new(bool)
			*o.Spec.Test = false
		}

		if o.Spec.Target != nil && *o.Spec.Target == "" {
			return fieldMustBeSetErr(".spec.target")
		}

		if o.Spec.Untagged == nil {
			o.Spec.Untagged = new(bool)
			*o.Spec.Untagged = false
		}
	}

	for i, secret := range o.Spec.Secrets {
		if secret.Type == "" {
			o.Spec.Secrets[i].Type = "file"
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

	for i, variant := range o.Spec.Variants {
		p := fmt.Sprintf(".spec.variants[%d]", i)
		if variant.Name == "" {
			return fieldMustBeSetErr(p + ".name")
		}
		if variant.With == nil {
			variant.With = o.Spec.WithBuildInstructions
		} else {
			if variant.With.Dockerfile == nil ||
				(variant.With.Dockerfile.Path == "" && variant.With.Dockerfile.Body == "") {
				variant.With.Dockerfile = o.Spec.Dockerfile
			}

			if filepath.IsAbs(variant.With.Dockerfile.Path) {
				return fmt.Errorf("absolute path in '%s.dockerfile.path' is prohibited (%q)", p, variant.With.Dockerfile.Path)
			}

			if strings.HasPrefix(variant.With.Dockerfile.Path, "..") {
				return fmt.Errorf("'%s.dockerfile.path' points outside of '%s.dir' (%q) - you can try '%s.dockerfile.body' instead", p, p, variant.With.Dockerfile.Path, p)
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

			if variant.With.Target != nil && *variant.With.Target == "" {
				return fieldMustBeSetErr(p + ".target")
			}

			if variant.With.Target == nil {
				variant.With.Target = o.Spec.Target
			}

			if variant.With.Untagged == nil {
				variant.With.Untagged = o.Spec.Untagged
			}

			if len(variant.With.Secrets) == 0 {
				variant.With.Secrets = o.Spec.Secrets
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
