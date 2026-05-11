// Package controller provides utilities for interacting with the Drycc controller.
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	drycc "github.com/drycc/controller-sdk-go"
	"github.com/drycc/pkg/log"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// New creates a new SDK client configured as the builder.
func New(controllerURL string) (*drycc.Client, error) {
	client, err := drycc.New(true, controllerURL, "")
	if err != nil {
		return client, err
	}
	client.UserAgent = "drycc-builder"

	passportURL := os.Getenv("DRYCC_PASSPORT_URL")
	passportKey := os.Getenv("DRYCC_PASSPORT_KEY")
	passportSecret := os.Getenv("DRYCC_PASSPORT_SECRET")
	if passportURL == "" || passportKey == "" || passportSecret == "" {
		return client, fmt.Errorf("passport credentials not configured")
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", passportKey)
	data.Set("client_secret", passportSecret)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/token/", passportURL), strings.NewReader(data.Encode()))
	if err != nil {
		return client, fmt.Errorf("failed to create token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return client, fmt.Errorf("failed to request token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client, fmt.Errorf("failed to get token: HTTP %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return client, fmt.Errorf("failed to decode token response: %v", err)
	}

	client.Token = tr.TokenType + " " + tr.AccessToken

	return client, nil
}

// CheckAPICompat checks for API compatibility errors and warns about them.
func CheckAPICompat(c *drycc.Client, err error) error {
	if err == drycc.ErrAPIMismatch {
		log.Info("WARNING: SDK and Controller API versions do not match. SDK: %s Controller: %s",
			drycc.APIVersion, c.ControllerAPIVersion)

		// API mismatch isn't fatal, so after warning continue on.
		return nil
	}

	return err
}
