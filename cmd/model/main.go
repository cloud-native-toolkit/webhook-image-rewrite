package model

import "sort"

type Config struct {
	DefaultHost string `yaml:"defaultHost"`
	ImageMappings []ImageMapping `yaml:"imageMappings"`
}

// constructor function
func(c *Config) fill_defaults(){

	// setting default values
	// if no values present
	if c.DefaultHost == "" {
		c.DefaultHost = "docker.io"
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
