//go:build submodule
package main

//go:generate mv go.mod.embed go.mod
//go:generate mv go.sum.embed go.sum
//go:generate go mod tidy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"google.golang.org/api/option"
)

// folderExists checks if a folder exists at the given path.
func folderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false // Folder does not exist
	}

	return info.IsDir() // Returns true if it exists and is a directory, false otherwise
}

// SetKeyValue sets a key-value pair in the store
func setValue(key, value string) error {
	baseURL := fmt.Sprintf("http://%s", os.Getenv("CONTAINIFYCI_HOST"))
	url := fmt.Sprintf("%s/mem/%s", baseURL, key)

	fmt.Printf("Store in mem %s", url)

	resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(value)))
	if err != nil {
		slog.Error("error post request", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set key-value pair: %s", resp.Status)
	}

	return nil
}

func main() {
	idToken, accessToken := gcpAuth()

	if os.Getenv("CONTAINIFYCI_HOST") != "" {
		setValue("idtoken", idToken)
		setValue("accesstoken", accessToken)

	} else {
		if !folderExists("/src/.gcloud") {
			err := os.Mkdir("/src/.gcloud", 0744)
			if err != nil {
				slog.Error("error create dir", "error", err)
				os.Exit(1)
			}
		}
		err := os.WriteFile("/src/.gcloud/idtoken", []byte(idToken), 0744)
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
}

func gcpAuth() (idToken string, accessToken string) {
	aud := os.Getenv("WORKLOAD_IDENTITY_PROVIDER")
	url := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	token := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	account := os.Getenv("ACCOUNT_EMAIL_OR_UNIQUEID")
	adc := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	ctx := context.Background()

	if adc != "" {
		creds, err := google.FindDefaultCredentials(ctx, iam.CloudPlatformScope)
		if err != nil {
			log.Fatalf("Failed to find default credentials: %v", err)
		}

		// Extract the email of the service account or user
		tokenSource := creds.TokenSource
		token, err := tokenSource.Token()
		if err != nil {
			log.Fatalf("Failed to get token: %v", err)
		}

		if idtoken, ok := token.Extra("id_token").(string); ok {
			idToken = idtoken

			// ID Token is present, this usually means a service account is used
			fmt.Println("Running as a service account")
			// Decode the ID token to get the email
			segments := strings.Split(idToken, ".")
			if len(segments) < 2 {
				log.Fatalf("Invalid ID token format")
			}

			payload, err := base64.RawURLEncoding.DecodeString(segments[1])
			if err != nil {
				log.Fatalf("Failed to decode ID token payload: %v", err)
			}

			var claims struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(payload, &claims); err != nil {
				log.Fatalf("Failed to unmarshal ID token payload: %v", err)
			}

			fmt.Printf("Authenticated as: %s (service account)\n", claims.Email)
		}

		accessToken = token.AccessToken
	} else {
		credBytes := fmt.Sprintf(`{
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
	}`, aud, url, aud, token)

		http.DefaultClient = &http.Client{
			Timeout: 10 * time.Second,
		}

		c, err := credentials.NewIamCredentialsClient(ctx, option.WithCredentialsJSON([]byte(credBytes)))
		if err != nil {
			slog.Error("error iam credentials client", "error", err)
			os.Exit(1)
		}
		defer c.Close()

		accountName := fmt.Sprintf("projects/-/serviceAccounts/%s", account)
		req := &credentialspb.GenerateIdTokenRequest{
			Name:         accountName,
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
			Name:  accountName,
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
	return
}
