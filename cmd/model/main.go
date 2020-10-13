package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
)

type Config struct {
	DefaultHost string `yaml:"defaultHost"`
	IgnoredNamespaces []string `yaml:"ignoredNamespaces"`
	ImageMappings []ImageMapping `yaml:"imageMappings"`
}

// constructor function
func(c *Config) fill_defaults(){

	// setting default values
	// if no values present
	if c.DefaultHost == "" {
		c.DefaultHost = "docker.io"
	}

	if len(c.IgnoredNamespaces) == 0 {
		c.IgnoredNamespaces = []string{
			metav1.NamespaceSystem,
			metav1.NamespacePublic,
		}
	}

	c.ImageMappings = sortImageMappings(c.ImageMappings)
}

func sortImageMappings(imageMappings []ImageMapping) []ImageMapping {
	result := make([]ImageMapping, len(imageMappings))
	copy(result, imageMappings)

	sort.SliceStable(result, func (i, j int) bool { return result[i].Source > result[j].Source })

	return result
}

type ImageMapping struct {
	Source string `yaml:"source"`
	Mirror  string `yaml:"mirror"`
}
