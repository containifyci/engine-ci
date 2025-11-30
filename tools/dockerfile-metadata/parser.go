package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// From represents a FROM instruction in a Dockerfile
type From struct {
	BaseImage   string // e.g., "golang:1.23-alpine"
	BaseVersion string // e.g., "1.23"
	StageName   string // e.g., "builder" (empty if no AS clause)
	Original    string // Original instruction line
	Line        int    // Line number in the Dockerfile
}

type Parser struct {
	result     *parser.Result
	dockerfile []byte
}

func New(dockerfile []byte) *Parser {
	return &Parser{dockerfile: dockerfile}
}

func (p *Parser) parse() (*parser.Result, error) {
	if p.result != nil {
		return p.result, nil
	}

	res, err := parser.Parse(bytes.NewReader(p.dockerfile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	p.result = res
	return res, nil
}

// ParseFrom parses a Dockerfile and returns all FROM instructions
func (p *Parser) ParseFrom() ([]From, error) {

	res, err := p.parse()
	if err != nil {
		return nil, err
	}

	var froms []From

	for _, node := range res.AST.Children {
		if strings.EqualFold(node.Value, "from") {
			baseImage, stageName, err := extractFromDetails(node.Original)
			if err != nil {
				return nil, err
			}

			froms = append(froms, From{
				BaseImage:   baseImage,
				BaseVersion: extractBaseVersion(baseImage),
				StageName:   stageName,
				Line:        node.StartLine,
				Original:    node.Original,
			})
		}
	}

	if len(froms) == 0 {
		return nil, fmt.Errorf("no FROM instructions found")
	}

	return froms, nil
}

func extractBaseVersion(baseImage string) string {
	// Extract the version part from the base image string
	// Example: "golang:1.23-alpine" -> "1.23"
	parts := strings.Split(baseImage, ":")
	if len(parts) < 2 {
		return "latest"
	}
	versionPart := parts[1]
	return versionPart
}

func extractFromDetails(original string) (baseImage, stageName string, err error) {
	fields := strings.Fields(original)
	if len(fields) < 2 {
		return "", "", fmt.Errorf("invalid FROM: %q", original)
	}
	baseImage = fields[1]

	if strings.HasPrefix(baseImage, "--platform=") && len(fields) >= 3 {
		baseImage = fields[2]
	}

	if len(fields) == 4 && strings.EqualFold(fields[2], "as") {
		stageName = fields[3]
		return baseImage, stageName, nil
	}

	return baseImage, stageName, nil
}
