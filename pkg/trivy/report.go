package trivy

import (
	_ "embed"
	"encoding/json"
	"log"
	"log/slog"
	"os"
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
	VulnerabilityID  string   `json:"VulnerabilityID"`
	PrimaryURL       string   `json:"PrimaryURL"`
	PkgID            string   `json:"PkgID"`
	PkgPath          string   `json:"PkgPath"`
	PkgName          string   `json:"PkgName"`
	InstalledVersion string   `json:"InstalledVersion"`
	FixedVersion     string   `json:"FixedVersion"`
	Status           string   `json:"Status"`
	Severity         string   `json:"Severity"`
	Title            string   `json:"Title"`
	Description      string   `json:"Description"`
	References       []string `json:"References"`
	PublishedDate    string   `json:"PublishedDate"`
	LastModifiedDate string   `json:"LastModifiedDate"`
}

type Result struct {
	Target          string          `json:"Target"`
	Class           string          `json:"Class"`
	Type            string          `json:"Type"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
}

type Comment struct {
	SchemaVersion int       `json:"SchemaVersion"`
	CreatedAt     time.Time `json:"CreatedAt"`
	ArtifactName  string    `json:"ArtifactName"`
	ArtifactType  string    `json:"ArtifactType"`
	Metadata      Metadata  `json:"Metadata"`
	Results       []Result  `json:"Results"`
}

func Parse(jsonData string) string {
	var comment Comment
	err := json.Unmarshal([]byte(jsonData), &comment)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Parse the template
	tmpl, err := template.New("security-report").Funcs(template.FuncMap{"formatDate": formatDate}).Parse(reportTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Create a buffer to store the rendered template output
	var output strings.Builder

	// Execute the template with the data and write to buffer
	err = tmpl.Execute(&output, comment)
	if err != nil {
		slog.Error("Failed to execute template.", "error", err)
		os.Exit(1)
	}

	// Print the rendered template output
	return output.String()
}

func formatDate(dateString string) string {
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		return dateString // return as-is if parsing fails
	}
	return date.Format("2006-01-02")
}
