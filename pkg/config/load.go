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
		return nil, "", fmt.Errorf("unable to open config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, obj); err != nil {
		return nil, "", fmt.Errorf("unable to parse config file %q: %w", path, err)
	}

	if err := obj.ApplyDefaultsAndValidate(); err != nil {
		return nil, "", fmt.Errorf("config file %q is invalid: %w", path, err)
	}

	return obj, base64.StdEncoding.EncodeToString(data), nil
}

func fieldMustBeSetErr(fieldpath string) error {
	return fmt.Errorf("'%s' must be set", fieldpath)
}

func fieldMustBeNonEmptyErr(fieldpath string) error {
	return fmt.Errorf("'%s' cannot be an empty string", fieldpath)
}

func fieldValueInvalidErr(fieldpath, invalidValue, validValue string) error {
	return fmt.Errorf("'%s: %q' is not valid, should be '%s: %q'", fieldpath, invalidValue, fieldpath, validValue)
}

// TODO: write tests for this
func (o *BuildConfig) ApplyDefaultsAndValidate() error {
	if o.APIVersion == "" {
		return fieldMustBeNonEmptyErr(".apiVersion")
	}
	if o.APIVersion != apiVersion {
		return fieldValueInvalidErr(".apiVersion", o.APIVersion, apiVersion)
	}

	if o.Kind == "" {
		return fieldMustBeNonEmptyErr(".kind")
	}
	if o.Kind != kind {
		return fieldValueInvalidErr(".kind", o.Kind, kind)
	}

	return o.Spec.ApplyDefaultsAndValidate()
}

func (o *BuildSpec) ApplyDefaultsAndValidate() error {
	if o.Name == "" {
		return fieldMustBeSetErr(".spec.name")
	}

	if (o.WithBuildInstructions == nil || o.WithBuildInstructions.Dir == nil) && len(o.Variants) == 0 {
		return fmt.Errorf("at least '.spec.dir' or '.spec.variants' must be set")
	}

	if o.WithBuildInstructions != nil {
		if o.Dockerfile == nil {
			o.Dockerfile = &DockerfileBuildInstructions{}
		}
		if o.Dockerfile.Path == "" && o.Dockerfile.Body == "" {
			o.Dockerfile.Path = defaultDockerfile
		}

		if filepath.IsAbs(o.Dockerfile.Path) {
			return fmt.Errorf("absolute path in '.spec.dockerfile.path: %q' is prohibited", o.Dockerfile.Path)
		}

		if strings.HasPrefix(o.Dockerfile.Path, "..") {
			return fmt.Errorf("'.spec.dockerfile.path: %q' points outside of '.spec.dir: %q' - you can try '.spec.dockerfile.body' instead", o.Dockerfile.Path, *o.Dir)
		}

		if o.Test == nil {
			o.Test = new(bool)
			*o.Test = false
		}

		if o.Target != nil && *o.Target == "" {
			return fieldMustBeNonEmptyErr(".spec.target")
		}

		if o.Untagged == nil {
			o.Untagged = new(bool)
			*o.Untagged = false
		}

		if len(o.Labels) != 0 {
			for k := range o.Labels {
				if strings.HasPrefix(k, "com.github.errordeveloper.imagine.") {
					return fmt.Errorf("label key %q is reseved for internal use", k)
				}
			}
		}
	}

	if o.TagMode == "" {
		o.TagMode = "GitTreeHash"
	}

	for i, secret := range o.Secrets {
		if secret.Type == "" {
			o.Secrets[i].Type = "file"
		}

		if o.Secrets[i].Type != "file" {
			return fmt.Errorf("usupported '.spec.secrets[%d].type: %q' - must be \"file\"", i, secret.Type)
		}

		if secret.ID == "" {
			return fieldMustBeNonEmptyErr(fmt.Sprintf(".spec.secrets[%d].id", i))
		}

		if secret.Source == "" {
			return fieldMustBeNonEmptyErr(fmt.Sprintf(".spec.secrets[%d].source", i))
		}
	}

	for i := range o.Variants {
		p := fmt.Sprintf(".spec.variants[%d]", i)
		variant := &o.Variants[i]
		if variant.Name == "" {
			return fieldMustBeNonEmptyErr(p + ".name")
		}
		if variant.With == nil {
			variant.With = o.WithBuildInstructions
		} else {
			p += ".with"

			if variant.With.Dir == nil {
				variant.With.Dir = o.Dir
			}

			if variant.With.Dockerfile == nil ||
				(variant.With.Dockerfile.Path == "" && variant.With.Dockerfile.Body == "") {
				variant.With.Dockerfile = o.Dockerfile
			}

			if filepath.IsAbs(variant.With.Dockerfile.Path) {
				return fmt.Errorf("absolute path in '%s.dockerfile.path: %q' is prohibited", p, variant.With.Dockerfile.Path)
			}

			if strings.HasPrefix(variant.With.Dockerfile.Path, "..") {
				return fmt.Errorf("'%s.dockerfile.path: %q' points outside of '%s.dir: %q' - you can try '%s.dockerfile.body' instead", p, variant.With.Dockerfile.Path, p, *variant.With.Dir, p)
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
				return fieldMustBeNonEmptyErr(p + ".target")
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

					if variant.With.Secrets[i].Type != "file" {
						return fmt.Errorf("usupported '%s.secrets[%d].type: %q' - must be \"file\"", p, i, secret.Type)
					}

					if secret.ID == "" {
						return fieldMustBeNonEmptyErr(fmt.Sprintf("%s.secrets[%d].id", p, i))
					}

					if secret.Source == "" {
						return fieldMustBeNonEmptyErr(fmt.Sprintf("%s.secrets[%d].source", p, i))
					}
				}
			}
		}
	}

	return nil
}
