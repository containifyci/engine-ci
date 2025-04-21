package sonarcloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/network"

	"gopkg.in/yaml.v3"
)

const (
	SONARQUBE_PASSWORD = "password"
	SONARQUBE_PATH     = "/.containifyci/metadata.txt"
)

var sonarqube = Image{
	Image:       "sonarqube:community",
	Name:        "sonarqube",
	HealthCheck: "%s/api/system/status",
}

type Image struct {
	Image       string
	Name        string
	HealthCheck HealthCheck
}

type HealthCheck string

func (h HealthCheck) URL(addr *network.Address) string {
	return fmt.Sprintf(string(sonarqube.HealthCheck), addr.InternalHost)
}

type SonarqubeContainer struct {
	token *string
	*container.Container
}

func (c *SonarqubeContainer) Token() *string {
	return c.token
}

func (c *SonarqubeContainer) Address() *network.Address {
	return &network.Address{Host: "https://sonarcloud.io:443", InternalHost: "http://localhost:9000"}
}

func NewSonarQube(build container.Build) *SonarqubeContainer {
	_token := os.Getenv("SONAR_TOKEN")
	return &SonarqubeContainer{
		Container: container.New(build),
		token:     &_token,
	}
}

func (c *SonarqubeContainer) Pull() error {
	return c.PullDefault(sonarqube.Image)
}

func (c *SonarqubeContainer) Images() []string {
	return []string{sonarqube.Image}
}

func (c *SonarqubeContainer) Start() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull sonarqube image", "error", err, "image", sonarqube.Image)
		return err
	}

	opts := types.ContainerConfig{}
	opts.Image = sonarqube.Image
	opts.Name = sonarqube.Name
	opts.Readiness = &types.ReadinessProbe{
		Endpoint: sonarqube.HealthCheck.URL(c.Address()),
		Timeout:  5 * time.Minute,
		Validate: func(body []byte) bool {
			return strings.Contains(string(body), "\"status\":\"UP\"")
		},
	}
	opts.Env = []string{
		"SONAR_ES_BOOTSTRAP_CHECKS_DISABLE=true",
		"JAVA_OPTS=-Xms1024m",
	}

	opts.ExposedPorts = []types.Binding{
		{
			Host: types.PortBinding{
				Port: "9000",
			},
			Container: types.PortBinding{
				Port: "9000",
			},
		},
	}

	opts.Memory = int64(3073741824)
	opts.CPU = uint64(2048)

	err = c.Create(opts)
	if err != nil {
		return err
	}

	err = c.Container.Start()
	if err != nil {
		return err
	}

	err = c.Ready()
	if err != nil {
		return err
	}

	localToken, err := c.Setup()
	if err != nil {
		return err
	}

	c.token = localToken
	return err
}

// PostRequest performs an HTTP POST request with the given URL, headers, and data.
func PostRequest(url, authHeader, data string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}

	return res, nil
}

func (c *SonarqubeContainer) Setup() (*string, error) {
	cnt, err := c.CopyFileFromContainer(SONARQUBE_PATH)
	if err != nil && err != io.EOF {
		slog.Error("Failed to copy metadata file from container", "error", err)
		os.Exit(1)
	}

	if err == io.EOF {
		slog.Info("Metadata file not found so perform setup")
	}

	if err == nil {
		slog.Info("Metadata file found so skip setup and read token from file", "metadata", cnt)
		var tokenResp map[string]string
		err := yaml.Unmarshal([]byte(cnt), &tokenResp)
		if err != nil {
			slog.Error("Failed to marshal metadata: %s", "error", err)
			os.Exit(1)
		}
		slog.Info("Metadata file found so skip setup and read token from file", "token", tokenResp)

		token, exists := tokenResp["token"]
		if !exists {
			slog.Error("Failed to get token from response")
			os.Exit(1)
		}
		return &token, nil
	}

	// Command 2
	url2 := "http://localhost:9000/api/users/change_password"
	data2 := fmt.Sprintf("login=admin&password=%s&previousPassword=admin", SONARQUBE_PASSWORD)
	authHeader := "Basic YWRtaW46YWRtaW4=" //admin:admin in base64

	res2, err := PostRequest(url2, authHeader, data2)
	if err != nil {
		slog.Error("Request 2 failed", "error", err)
		os.Exit(1)
	}
	defer res2.Body.Close()

	body2, err := io.ReadAll(res2.Body)
	if err != nil {
		slog.Error("Error reading response 2", "error", err)
		os.Exit(1)
	}

	fmt.Println("Response 2 Status:", res2.Status)
	fmt.Println("Response 2 Body:", string(body2))

	// Command 1
	url1 := "http://localhost:9000/api/user_tokens/generate"
	data1 := "name=local"
	authCreds := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "admin", SONARQUBE_PASSWORD)))
	authHeader = fmt.Sprintf("Basic %s", authCreds)

	res1, err := PostRequest(url1, authHeader, data1)
	if err != nil {
		slog.Error("Request 1 failed: %v", "error", err)
		os.Exit(1)
	}
	defer res1.Body.Close()

	body1, err := io.ReadAll(res1.Body)
	if err != nil {
		slog.Error("Error reading response 1", "error", err)
		os.Exit(1)
	}

	fmt.Println("Response 1 Status:", res1.Status)
	fmt.Println("Response 1 Body:", string(body1))

	var tokenResp map[string]string

	err = json.Unmarshal(body1, &tokenResp)
	if err != nil {
		slog.Error("Failed to unmarshal json response: %s", "error", err)
		os.Exit(1)
	}

	meta, err := yaml.Marshal(tokenResp)
	if err != nil {
		slog.Error("Failed to marshal metadata: %s", "error", err)
		os.Exit(1)
	}

	err = c.CopyContentTo(string(meta), SONARQUBE_PATH)
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	token, exists := tokenResp["token"]
	if !exists {
		slog.Error("Failed to get token from response")
		os.Exit(1)
	}
	return &token, nil
}
