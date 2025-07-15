package controller

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	builderconf "github.com/drycc/builder/pkg/conf"
	"github.com/stretchr/testify/assert"

	drycc "github.com/drycc/controller-sdk-go"
)

func TestNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove service-key from %s (%s)", tmpDir, err)
		}
	}()

	builderconf.ServiceKeyLocation = filepath.Join(tmpDir, "service-key")
	data := []byte("testbuilderkey")
	if err := os.WriteFile(builderconf.ServiceKeyLocation, data, 0644); err != nil {
		t.Fatalf("error creating %s (%s)", builderconf.ServiceKeyLocation, err)
	}

	url := "http://127.0.0.1:80"
	cli, err := New(url)
	assert.Equal(t, err, nil)
	assert.Equal(t, cli.ControllerURL.String(), url, "data")
	assert.Equal(t, cli.ServiceKey, string(data), "data")
	assert.Equal(t, cli.UserAgent, "drycc-builder", "user-agent")

	url = "http://127.0.0.1:invalid-port-number"
	if _, err = New(url); err == nil {
		t.Errorf("expected error with invalid port number, got nil")
	}
}

func TestNewWithInvalidBuilderKeyPath(t *testing.T) {
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
