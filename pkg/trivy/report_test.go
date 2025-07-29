package trivy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Success(t *testing.T) {
	// Valid JSON data for trivy report
	validJSON := `{
		"CreatedAt": "2023-01-01T00:00:00Z",
		"ArtifactName": "test-image:latest",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": [
			{
				"Target": "test-image:latest (debian 12.0)",
				"Class": "os-pkgs",
				"Type": "debian",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-0001",
						"PkgID": "libc6@2.36-9",
						"PkgName": "libc6",
						"InstalledVersion": "2.36-9",
						"FixedVersion": "2.36-9+deb12u1",
						"Title": "Sample vulnerability",
						"Description": "A sample vulnerability for testing",
						"Severity": "HIGH",
						"PublishedDate": "2023-01-01T00:00:00Z",
						"LastModifiedDate": "2023-01-02T00:00:00Z",
						"References": ["https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-0001"],
						"PrimaryURL": "https://avd.aquasec.com/nvd/cve-2023-0001"
					}
				]
			}
		],
		"SchemaVersion": 2
	}`

	result, err := Parse(validJSON)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify the result contains expected content
	assert.Contains(t, result, "Container Security Risks")
	assert.Contains(t, result, "test-image:latest")
	assert.Contains(t, result, "CVE-2023-0001")
	assert.Contains(t, result, "HIGH")
}

func TestParse_InvalidJSON(t *testing.T) {
	// Test with invalid JSON
	invalidJSON := `{"invalid": json}`

	result, err := Parse(invalidJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON")
}

func TestParse_EmptyJSON(t *testing.T) {
	// Test with empty JSON
	emptyJSON := ""

	result, err := Parse(emptyJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON")
}

func TestParse_MalformedJSON(t *testing.T) {
	// Test with malformed JSON
	malformedJSON := `{"CreatedAt": "invalid-date", "Results"`

	result, err := Parse(malformedJSON)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON")
}

func TestParse_ValidJSONWithEmptyResults(t *testing.T) {
	// Test with valid JSON but empty results
	validEmptyJSON := `{
		"CreatedAt": "2023-01-01T00:00:00Z",
		"ArtifactName": "test-image:latest",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": [],
		"SchemaVersion": 2
	}`

	result, err := Parse(validEmptyJSON)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Container Security Risks")
	assert.Contains(t, result, "test-image:latest")
}

func TestParse_NullValues(t *testing.T) {
	// Test with JSON containing null values
	nullJSON := `{
		"CreatedAt": null,
		"ArtifactName": "test-image:latest",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": null,
		"SchemaVersion": 2
	}`

	result, err := Parse(nullJSON)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestParse_LargeJSON(t *testing.T) {
	// Test with a large JSON payload (using smaller number to avoid test timeout)
	var vulnerabilities []string
	for i := 0; i < 10; i++ {
		vuln := fmt.Sprintf(`{
			"VulnerabilityID": "CVE-2023-%04d",
			"PkgID": "pkg%d@1.0.0",
			"PkgName": "pkg%d",
			"InstalledVersion": "1.0.0",
			"FixedVersion": "1.0.1",
			"Title": "Vulnerability %d",
			"Description": "Description for vulnerability %d",
			"Severity": "MEDIUM",
			"PublishedDate": "2023-01-01T00:00:00Z",
			"LastModifiedDate": "2023-01-02T00:00:00Z",
			"References": ["https://example.com/%d"],
			"PrimaryURL": "https://example.com/%d"
		}`, i, i, i, i, i, i, i)
		vulnerabilities = append(vulnerabilities, vuln)
	}

	largeJSON := `{
		"CreatedAt": "2023-01-01T00:00:00Z",
		"ArtifactName": "test-image:latest",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": [
			{
				"Target": "test-image:latest (debian 12.0)",
				"Class": "os-pkgs",
				"Type": "debian",
				"Vulnerabilities": [` + strings.Join(vulnerabilities, ",") + `]
			}
		],
		"SchemaVersion": 2
	}`

	result, err := Parse(largeJSON)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Container Security Risks")
}

func TestFormatDate_ValidDate(t *testing.T) {
	// Test formatDate function with valid RFC3339 date
	validDate := "2023-01-01T12:34:56Z"
	result := formatDate(validDate)
	assert.Equal(t, "2023-01-01", result)
}

func TestFormatDate_InvalidDate(t *testing.T) {
	// Test formatDate function with invalid date - should return as-is
	invalidDate := "invalid-date"
	result := formatDate(invalidDate)
	assert.Equal(t, "invalid-date", result)
}

func TestFormatDate_EmptyDate(t *testing.T) {
	// Test formatDate function with empty date
	emptyDate := ""
	result := formatDate(emptyDate)
	assert.Equal(t, "", result)
}

func TestFormatDate_AlternativeDateFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RFC3339 with timezone",
			input:    "2023-01-01T12:34:56+02:00",
			expected: "2023-01-01",
		},
		{
			name:     "RFC3339 with nanoseconds",
			input:    "2023-01-01T12:34:56.123456789Z",
			expected: "2023-01-01",
		},
		{
			name:     "Wrong format",
			input:    "2023-01-01 12:34:56",
			expected: "2023-01-01 12:34:56", // Should return as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParse_TemplateExecutionErrors(t *testing.T) {
	// Test scenario where template parsing might fail
	// This would require modifying the embedded template, but we can test with data that might cause issues
	jsonWithSpecialChars := `{
		"CreatedAt": "2023-01-01T00:00:00Z",
		"ArtifactName": "test-image:latest{{.invalid}}",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": [
			{
				"Target": "test-image:latest (debian 12.0)",
				"Class": "os-pkgs", 
				"Type": "debian",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-0001",
						"PkgID": "libc6@2.36-9",
						"PkgName": "libc6",
						"InstalledVersion": "2.36-9",
						"FixedVersion": "2.36-9+deb12u1",
						"Title": "Sample {{.test}} vulnerability",
						"Description": "A sample vulnerability for testing",
						"Severity": "HIGH",
						"PublishedDate": "2023-01-01T00:00:00Z",
						"LastModifiedDate": "2023-01-02T00:00:00Z",
						"References": ["https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-0001"],
						"PrimaryURL": "https://avd.aquasec.com/nvd/cve-2023-0001"
					}
				]
			}
		],
		"SchemaVersion": 2
	}`

	// This should still work since the template system should handle this gracefully
	result, err := Parse(jsonWithSpecialChars)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Benchmark test to ensure performance is maintained
func BenchmarkParse(b *testing.B) {
	validJSON := `{
		"CreatedAt": "2023-01-01T00:00:00Z",
		"ArtifactName": "test-image:latest",
		"ArtifactType": "container_image",
		"Metadata": {
			"OS": {
				"Family": "debian",
				"Name": "12.0"
			},
			"ImageID": "sha256:abc123",
			"RepoTags": ["test-image:latest"]
		},
		"Results": [
			{
				"Target": "test-image:latest (debian 12.0)",
				"Class": "os-pkgs",
				"Type": "debian",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-0001",
						"PkgID": "libc6@2.36-9",
						"PkgName": "libc6",
						"InstalledVersion": "2.36-9",
						"FixedVersion": "2.36-9+deb12u1",
						"Title": "Sample vulnerability",
						"Description": "A sample vulnerability for testing",
						"Severity": "HIGH",
						"PublishedDate": "2023-01-01T00:00:00Z",
						"LastModifiedDate": "2023-01-02T00:00:00Z",
						"References": ["https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-0001"],
						"PrimaryURL": "https://avd.aquasec.com/nvd/cve-2023-0001"
					}
				]
			}
		],
		"SchemaVersion": 2
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(validJSON)
		if err != nil {
			b.Fatal(err)
		}
	}
}
