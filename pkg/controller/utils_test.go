package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	drycc "github.com/drycc/controller-sdk-go"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "testing_token", "token_type": "Bearer"}`))
	}))
	defer ts.Close()

	os.Setenv("DRYCC_PASSPORT_URL", ts.URL)
	os.Setenv("DRYCC_PASSPORT_KEY", "testing_key")
	os.Setenv("DRYCC_PASSPORT_SECRET", "testing_secret")
	defer os.Unsetenv("DRYCC_PASSPORT_URL")
	defer os.Unsetenv("DRYCC_PASSPORT_KEY")
	defer os.Unsetenv("DRYCC_PASSPORT_SECRET")

	url := "http://127.0.0.1:80"
	cli, err := New(url)
	assert.Equal(t, err, nil)
	assert.Equal(t, cli.ControllerURL.String(), url, "data")
	assert.Equal(t, cli.Token, "Bearer testing_token", "data")
	assert.Equal(t, cli.UserAgent, "drycc-builder", "user-agent")

	url = "http://127.0.0.1:invalid-port-number"
	if _, err = New(url); err == nil {
		t.Errorf("expected error with invalid port number, got nil")
	}
}

func TestNewWithInvalidCredentials(t *testing.T) {
	os.Unsetenv("DRYCC_PASSPORT_URL")
	url := "http://127.0.0.1:80"
	_, err := New(url)
	assert.True(t, err != nil, "no error received when there should have been")
}

func TestCheckAPICompat(t *testing.T) {
	client := &drycc.Client{ControllerAPIVersion: drycc.APIVersion}
	err := drycc.ErrAPIMismatch

	if apiErr := CheckAPICompat(client, err); apiErr != nil {
		t.Errorf("api errors are non-fatal and should return nil, got '%v'", apiErr)
	}

	err = errors.New("random error")
	if apiErr := CheckAPICompat(client, err); apiErr == nil {
		t.Error("expected error to be returned, got nil")
	}
}
