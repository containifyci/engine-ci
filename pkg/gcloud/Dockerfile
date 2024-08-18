FROM golang:1.22.5-alpine as builder

WORKDIR /src
RUN cat > main.go <<EOF
package main

//go:generate go mod init gcloud
//go:generate go mod tidy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"google.golang.org/api/option"
)

func main() {
	idToken, accessToken := gcpAuth()

	err := os.Mkdir("/src/.gcloud", 0744)
	if err != nil {
		slog.Error("error create dir", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("/src/.gcloud/idtoken", []byte(idToken), 0744)
	if err != nil {
		slog.Error("error write id token", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("/src/.gcloud/accesstoken", []byte(accessToken), 0744)
	if err != nil {
		slog.Error("error write access token", "error", err)
		os.Exit(1)
	}
}

func gcpAuth() (idToken string, accessToken string) {
	aud := os.Getenv("WORKLOAD_IDENTITY_PROVIDER")
	url := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	token := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	credBytes := fmt.Sprintf(\`{
  	"universe_domain": "googleapis.com",
		"type": "external_account",
		"audience": "%s",
		"subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"token_url": "https://sts.googleapis.com/v1/token",
		"credential_source": {
			"url": "%s&audience=%s",
			"headers": {
				"Authorization": "Bearer %s"
			},
			"format": {
				"type": "json",
				"subject_token_field_name": "value"
			}
		}
	}\`, aud, url, aud, token)

	ctx := context.Background()

	http.DefaultClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	slog.Info("cred", "credFile", credBytes)
	raw := []byte(credBytes)

	c, err := credentials.NewIamCredentialsClient(ctx, option.WithCredentialsJSON(raw))
	if err != nil {
		slog.Error("error iam credentials client", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	req := &credentialspb.GenerateIdTokenRequest{
		Name:         "projects/-/serviceAccounts/pactflow-employee-reward@flink-core-shared.iam.gserviceaccount.com",
		Audience:     aud,
		IncludeEmail: true,
	}
	resp, err := c.GenerateIdToken(ctx, req)
	if err != nil {
		slog.Error("error generate id token", "error", err)
		os.Exit(1)
	}

	idToken = resp.Token

	acReq := credentialspb.GenerateAccessTokenRequest{
		Name:  "projects/-/serviceAccounts/pactflow-employee-reward@flink-core-shared.iam.gserviceaccount.com",
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform.read-only"},
	}

	resp3, err := c.GenerateAccessToken(ctx, &acReq)
	if err != nil {
		slog.Error("error generate access token", "error", err)
		os.Exit(1)
	}

	accessToken = resp3.AccessToken
	return
}
EOF

RUN go generate main.go && \
  go build -o oidc main.go

FROM alpine

WORKDIR /

COPY --from=builder /src/oidc /bin/oidc

CMD /bin/oidc
