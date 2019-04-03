package gitreceive

import (
	"github.com/drycc/controller-sdk-go/api"
	"os"
	"testing"
)

func TestGetBuildType(t *testing.T) {
	tmpDir := os.TempDir()
	config := api.Config{}
	bType := getBuildType(tmpDir, config)
	if bType != buildTypeDockerbuilder {
		t.Fatalf("expected procfile build, got %s", bType)
	}
	if _, err := os.Create(tmpDir + "/Dockerfile"); err != nil {
		t.Fatalf("error creating %s/Dockerfile (%s)", tmpDir, err)
	}

	bType = getBuildType(tmpDir, config)
	if bType != buildTypeDockerbuilder {
		t.Fatalf("expected dockerfile build, got %s", bType)
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
	config.Values = map[string]interface{}{
		"DRYCC_BUILDER": "slugbuilder",
	}
	bType = getBuildType(tmpDir, config)
	if bType != buildTypeSlugbuilder {
		t.Fatalf("expected procfile build, got %s", bType)
	}

	config.Values = map[string]interface{}{
		"DRYCC_BUILDER": "dockerbuilder",
	}
	bType = getBuildType(tmpDir, config)
	if bType != buildTypeDockerbuilder {
		t.Fatalf("expected Dockerfile build, got %s", bType)
	}
}
