package config

import (
	"encoding/json"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

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
		if variant.Name != nil && s.Name != *variant.Name {
			_, err = fmt.Fprintf(w, "%s (%s):\n", s.Name, *variant.Name)
		} else {
			_, err = fmt.Fprintf(w, "%s:\n", s.Name)
		}
		if err != nil {
			return err
		}
		if len(variant.RegistryRefs) == 0 {
			if _, err = fmt.Fprintf(w, "- %s\n", *variant.Digest); err != nil {
				return err
			}
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
		if len(variant.RegistryRefs) == 0 {
			if variant.Name != nil && s.Name != *variant.Name {
				_, err = fmt.Fprintf(w, "%s,%s,%s\n", s.Name, *variant.Name, *variant.Digest)
			} else {
				_, err = fmt.Fprintf(w, "%s,,%s\n", s.Name, *variant.Digest)
			}
			if err != nil {
				return err
			}
		}
		for _, ref := range variant.RegistryRefs {
			if variant.Name != nil && s.Name != *variant.Name {
				_, err = fmt.Fprintf(w, "%s,%s,%s@%s\n", s.Name, *variant.Name, ref, *variant.Digest)
			} else {
				_, err = fmt.Fprintf(w, "%s,,%s@%s\n", s.Name, ref, *variant.Digest)
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
