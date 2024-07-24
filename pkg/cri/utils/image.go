package utils

import (
	"fmt"
	"regexp"
)

type ImageInfo struct {
	Registry string
	Image    string
	Tag      string
}

func ParseDockerImage(image string) (ImageInfo, error) {
	var info ImageInfo

	// Regular expression to parse the Docker image string
	re := regexp.MustCompile(`^(?:(?P<registry>[^/]+(?:/[^/]+)*)/)?(?P<image>[^:]+)(?::(?P<tag>.+))?$`)
	match := re.FindStringSubmatch(image)
	if match == nil {
		return info, fmt.Errorf("invalid Docker image string: %s", image)
	}

	// Map to hold the matched groups
	groupNames := re.SubexpNames()
	for i, name := range match {
		switch groupNames[i] {
		case "registry":
			info.Registry = name
		case "image":
			info.Image = name
		case "tag":
			info.Tag = name
		}
	}

	// Default to "latest" tag if no tag is specified
	if info.Tag == "" {
		info.Tag = "latest"
	}

	return info, nil
}

func ImageURI(registry, image, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registry, image, tag)
}
