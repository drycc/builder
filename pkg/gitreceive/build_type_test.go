package gitreceive

import (
	"os"
	"testing"

	"github.com/drycc/controller-sdk-go/api"
)

func TestGetStack(t *testing.T) {
	tmpDir := os.TempDir()
	config := api.Config{}
	stack := getStack(tmpDir, config)
	if stack["name"] != "buildpack" {
		t.Fatalf("expected procfile build, got %s", stack)
	}
	if _, err := os.Create(tmpDir + "/Dockerfile"); err != nil {
		t.Fatalf("error creating %s/Dockerfile (%s)", tmpDir, err)
	}

	stack = getStack(tmpDir, config)
	if stack["name"] != "container" {
		t.Fatalf("expected dockerfile build, got %s", stack)
	}

	if _, err := os.Create(tmpDir + "/Procfile"); err != nil {
		t.Fatalf("error creating %s/Procfile (%s)", tmpDir, err)
	}

	defer func() {
		if err := os.Remove(tmpDir + "/Dockerfile"); err != nil {
			t.Fatalf("failed to remove Dockerfile from %s (%s)", tmpDir, err)
		}
		if err := os.Remove(tmpDir + "/Procfile"); err != nil {
			t.Fatalf("failed to remove Procfile from %s (%s)", tmpDir, err)
		}
	}()
	config.Values = []api.ConfigValue{
		{
			Group: "global",
			KV: api.KV{
				Name:  "DRYCC_STACK",
				Value: "buildpack",
			},
		},
	}
	stack = getStack(tmpDir, config)
	if stack["name"] != "buildpack" {
		t.Fatalf("expected procfile build, got %s", stack)
	}

	config.Values = []api.ConfigValue{
		{
			Group: "global",
			KV: api.KV{
				Name:  "DRYCC_STACK",
				Value: "container",
			},
		},
	}
	stack = getStack(tmpDir, config)
	if stack["name"] != "container" {
		t.Fatalf("expected Dockerfile build, got %s", stack)
	}
}
