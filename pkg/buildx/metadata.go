package buildx

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/errordeveloper/imagine/pkg/config"
)

type BakeMetadata map[string]BakeImageMetadata

type BakeImageMetadata struct {
	ConfigDigest string `json:"containerimage.config.digest"`
	Digest       string `json:"containerimage.digest"`
	RegistryRefs string `json:"image.name"`
}

func LoadBakeMetadata(filename string) (*BakeMetadata, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	m := &BakeMetadata{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *BakeMetadata) ToBuildSummary(name string) *config.BuildSummary {
	s := config.NewBuildSummary(name)
	for k, v := range *m {
		i := config.ImageSummary{
			Digest: new(string),
		}
		i.RegistryRefs = strings.Split(v.RegistryRefs, ",")
		i.Digest = &v.Digest
		if variantName := strings.TrimLeft(k, name+"-"); variantName != "" {
			i.VarianName = new(string)
			*i.VarianName = variantName
		}
		s.Images = append(s.Images, i)
	}
	return s
}
