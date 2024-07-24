package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDockerImage(t *testing.T) {
	tests := []struct {
		image     string
		registry  string
		imageName string
		tag       string
	}{
		{"golang:alpine", "", "golang", "alpine"},
		{"maven:3-eclipse-temurin-17-alpine", "", "maven", "3-eclipse-temurin-17-alpine"},
		{"sonarqube:community", "", "sonarqube", "community"},
		{"nginx:latest", "", "nginx", "latest"},
		{"alpine:latest", "", "alpine", "latest"},
		{"datadog/agent:latest", "datadog", "agent", "latest"},
		{"golang", "", "golang", "latest"},
		{"golang:4-alpine", "", "golang", "4-alpine"},
		{"golangci/golangci-lint", "golangci", "golangci-lint", "latest"},
		{"sonarsource/sonar-scanner-cli:latest", "sonarsource", "sonar-scanner-cli", "latest"},
		{"postgres:latest", "", "postgres", "latest"},
		{"postgres:15", "", "postgres", "15"},
		{"eclipse-temurin:17-jdk", "", "eclipse-temurin", "17-jdk"},
		{"eclipse-temurin", "", "eclipse-temurin", "latest"},
		{"eclipse-temurin:17-jdk-focal", "", "eclipse-temurin", "17-jdk-focal"},
		{"testcontainers/ryuk:0.6.0", "testcontainers", "ryuk", "0.6.0"},
		{"registry.access.redhat.com/ubi8/openjdk-17:1.14", "registry.access.redhat.com/ubi8", "openjdk-17", "1.14"},
		{"quay.io/quarkus/ubi-quarkus-native-image:22.1.0-java17-amd64", "quay.io/quarkus", "ubi-quarkus-native-image", "22.1.0-java17-amd64"},
		{"europe-west3-docker.pkg.dev/shared/containifyci/protobuf:latest", "europe-west3-docker.pkg.dev/shared/containifyci", "protobuf", "latest"},
		{"europe-west3-docker.pkg.dev/shared/containifyci/protobuf", "europe-west3-docker.pkg.dev/shared/containifyci", "protobuf", "latest"},
		{"maven:3.3-jdk-8", "", "maven", "3.3-jdk-8"},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			info, err := ParseDockerImage(test.image)
			assert.NoError(t, err)
			assert.Equal(t, ImageInfo{test.registry, test.imageName, test.tag}, info)
		})
	}
}
