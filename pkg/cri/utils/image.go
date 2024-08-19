package utils

import (
	"fmt"
	"regexp"
	"strings"
)

const DEFAULT_DOCKER_ADDRESS = "docker.io"

type ImageInfo struct {
	Server   string
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
			if info.Registry != "" {
				srv := strings.Split(info.Registry, "/")[0]
				if ContainsDomain(srv) {
					info.Server = srv
				} else {
					info.Server = DEFAULT_DOCKER_ADDRESS
				}
			} else {
				info.Server = DEFAULT_DOCKER_ADDRESS
			}
		case "image":
			info.Image = name
		case "tag":
			info.Tag = name
		}
	}

	if info.Registry == "" {
		info.Server = DEFAULT_DOCKER_ADDRESS
	}

	// Default to "latest" tag if no tag is specified
	if info.Tag == "" {
		info.Tag = "latest"
	}

	return info, nil
}

// Function to check if a string contains a domain address
func ContainsDomain(s string) bool {
	// Regular expression to match domain names
	domainRegex := regexp.MustCompile(`(?i)\b(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+(?:[a-z]{2,6}\b|xn--[a-z0-9]{1,59}\b)`)

	// FindString returns the matched string if found
	return domainRegex.FindString(s) != ""
}

func ImageURI(registry, image, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registry, image, tag)
}
