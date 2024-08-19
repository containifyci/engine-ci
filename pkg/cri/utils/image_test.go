package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDockerImage(t *testing.T) {
	tests := []struct {
		image     string
		server    string
		registry  string
		imageName string
		tag       string
	}{
		{"golang:alpine", "docker.io", "", "golang", "alpine"},
		{"maven:3-eclipse-temurin-17-alpine", "docker.io", "", "maven", "3-eclipse-temurin-17-alpine"},
		{"sonarqube:community", "docker.io", "", "sonarqube", "community"},
		{"nginx:latest", "docker.io", "", "nginx", "latest"},
		{"alpine:latest", "docker.io", "", "alpine", "latest"},
		{"datadog/agent:latest", "docker.io", "datadog", "agent", "latest"},
		{"golang", "docker.io", "", "golang", "latest"},
		{"golang:4-alpine", "docker.io", "", "golang", "4-alpine"},
		{"golangci/golangci-lint", "docker.io", "golangci", "golangci-lint", "latest"},
		{"sonarsource/sonar-scanner-cli:latest", "docker.io", "sonarsource", "sonar-scanner-cli", "latest"},
		{"postgres:latest", "docker.io", "", "postgres", "latest"},
		{"postgres:15", "docker.io", "", "postgres", "15"},
		{"eclipse-temurin:17-jdk", "docker.io", "", "eclipse-temurin", "17-jdk"},
		{"eclipse-temurin", "docker.io", "", "eclipse-temurin", "latest"},
		{"eclipse-temurin:17-jdk-focal", "docker.io", "", "eclipse-temurin", "17-jdk-focal"},
		{"testcontainers/ryuk:0.6.0", "docker.io", "testcontainers", "ryuk", "0.6.0"},
		{"registry.access.redhat.com/ubi8/openjdk-17:1.14", "registry.access.redhat.com", "registry.access.redhat.com/ubi8", "openjdk-17", "1.14"},
		{"quay.io/quarkus/ubi-quarkus-native-image:22.1.0-java17-amd64", "quay.io", "quay.io/quarkus", "ubi-quarkus-native-image", "22.1.0-java17-amd64"},
		{"europe-west3-docker.pkg.dev/shared/containifyci/protobuf:latest", "europe-west3-docker.pkg.dev", "europe-west3-docker.pkg.dev/shared/containifyci", "protobuf", "latest"},
		{"europe-west3-docker.pkg.dev/shared/containifyci/protobuf", "europe-west3-docker.pkg.dev", "europe-west3-docker.pkg.dev/shared/containifyci", "protobuf", "latest"},
		{"maven:3.3-jdk-8", "docker.io", "", "maven", "3.3-jdk-8"},
		{"ghcr.io/octopus/image:latest", "ghcr.io", "ghcr.io/octopus", "image", "latest"},
		{"aws_account_id.dkr.ecr.us-west-2.amazonaws.com/my-repository:tag", "aws_account_id.dkr.ecr.us-west-2.amazonaws.com", "aws_account_id.dkr.ecr.us-west-2.amazonaws.com", "my-repository", "tag"},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			info, err := ParseDockerImage(test.image)
			assert.NoError(t, err)
			assert.Equal(t, ImageInfo{test.server, test.registry, test.imageName, test.tag}, info)
		})
	}
}
