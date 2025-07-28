package trivy

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed report.tmpl
var reportTemplate string

type OS struct {
	Family string `json:"Family"`
	Name   string `json:"Name"`
}

type Metadata struct {
	OS       OS       `json:"OS"`
	ImageID  string   `json:"ImageID"`
	RepoTags []string `json:"RepoTags"`
}

type Vulnerability struct {
	FixedVersion     string   `json:"FixedVersion"`
	VulnerabilityID  string   `json:"VulnerabilityID"`
	PkgID            string   `json:"PkgID"`
	PkgPath          string   `json:"PkgPath"`
	PkgName          string   `json:"PkgName"`
	InstalledVersion string   `json:"InstalledVersion"`
	PrimaryURL       string   `json:"PrimaryURL"`
	Status           string   `json:"Status"`
	Description      string   `json:"Description"`
	Title            string   `json:"Title"`
	Severity         string   `json:"Severity"`
	LastModifiedDate string   `json:"LastModifiedDate"`
	PublishedDate    string   `json:"PublishedDate"`
	References       []string `json:"References"`
}

type Result struct {
	Target          string          `json:"Target"`
	Class           string          `json:"Class"`
	Type            string          `json:"Type"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
}

type Comment struct {
	CreatedAt     time.Time `json:"CreatedAt"`
	ArtifactName  string    `json:"ArtifactName"`
	ArtifactType  string    `json:"ArtifactType"`
	Metadata      Metadata  `json:"Metadata"`
	Results       []Result  `json:"Results"`
	SchemaVersion int       `json:"SchemaVersion"`
}

func Parse(jsonData string) (string, error) {
	var comment Comment
	err := json.Unmarshal([]byte(jsonData), &comment)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Parse the template
	tmpl, err := template.New("security-report").Funcs(template.FuncMap{"formatDate": formatDate}).Parse(reportTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Create a buffer to store the rendered template output
	var output strings.Builder

	// Execute the template with the data and write to buffer
	err = tmpl.Execute(&output, comment)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Print the rendered template output
	return output.String(), nil
}

func formatDate(dateString string) string {
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		return dateString // return as-is if parsing fails
	}
	return date.Format("2006-01-02")
}
