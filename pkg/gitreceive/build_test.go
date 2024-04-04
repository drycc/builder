package gitreceive

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	builderconf "github.com/drycc/builder/pkg/conf"
	"github.com/drycc/builder/pkg/sys"
	"github.com/drycc/controller-sdk-go/api"
	"github.com/drycc/pkg/log"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type testJSONStruct struct {
	Foo string `json:"foo,omitempty"`
}

type podSelectorBuildCase struct {
	Config string
	Output map[string]string
}

func TestBuild(t *testing.T) {
	config := &Config{}
	env := sys.NewFakeEnv()
	// NOTE(bacongobbler): there's a little easter egg here... ;)
	sha := "0462cef5812ce31fe12f25596ff68dc614c708af"

	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove tmpdir (%s)", err)
		}
	}()

	config.GitHome = tmpDir

	storageDriver, err := factory.Create(context.Background(), "inmemory", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := build(config, storageDriver, nil, env, sha); err == nil {
		t.Error("expected running build() without setting config.ImagebuilderImagePullPolicy to fail")
	}

	config.ImagebuilderImagePullPolicy = "Always"
	if err := build(config, storageDriver, nil, env, sha); err == nil {
		t.Error("expected running build() without setting config.ImagebuilderImagePullPolicy to fail")
	}

	err = build(config, storageDriver, nil, env, "abc123")
	expected := "git sha abc123 was invalid"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%v'", expected, err.Error())
	}

	if err := build(config, storageDriver, nil, env, sha); err == nil {
		t.Error("expected running build() without valid controller client info to fail")
	}

	config.ControllerHost = "localhost"
	config.ControllerPort = "1234"

	if err := build(config, storageDriver, nil, env, sha); err == nil {
		t.Error("expected running build() without a valid builder key to fail")
	}

	builderconf.BuilderKeyLocation = filepath.Join(tmpDir, "builder-key")
	data := []byte("testbuilderkey")
	if err := os.WriteFile(builderconf.BuilderKeyLocation, data, 0644); err != nil {
		t.Fatalf("error creating %s (%s)", builderconf.BuilderKeyLocation, err)
	}

	if err := build(config, storageDriver, nil, env, sha); err == nil {
		t.Error("expected running build() without a valid controller connection to fail")
	}
}

func TestRepoCmd(t *testing.T) {
	cmd := repoCmd("/tmp", "ls")
	if cmd.Dir != "/tmp" {
		t.Errorf("expected '%s', got '%s'", "/tmp", cmd.Dir)
	}
}

func TestGetDryccfileFromRepoSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	data := `
build:
  docker:
    web: Dockerfile
    worker: worker/Dockerfile
  config:
    RAILS_ENV: development
    FOO: bar
run:
  command:
  - ./deployment-tasks.sh
  image: worker
deploy:
  web:
    command:
    - bash
    - -c
    args: bundle exec puma -C config/puma.rb
  worker:
    command:
    - bash
    - -c
    args:
    - python myworker.py
  asset-syncer:
    command:
    - bash
    - -c
    args:
    - python asset-syncer.py
    image: worker
`
	if err := os.WriteFile(tmpDir+"/drycc.yaml", []byte(data), 0644); err != nil {
		t.Fatalf("error creating %s/drycc.yaml (%s)", tmpDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove drycc.yaml from %s (%s)", tmpDir, err)
		}
	}()
	dryccfile, err := getDryccfile(tmpDir)
	actualData := map[string]interface{}{}
	yaml.Unmarshal([]byte(data), &actualData)
	assert.Equal(t, err, nil)
	assert.Equal(t, dryccfile, actualData, "data")
}

func TestGetProcfileFromRepoSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}
	data := []byte("web: example-go")
	if err := os.WriteFile(tmpDir+"/Procfile", data, 0644); err != nil {
		t.Fatalf("error creating %s/Procfile (%s)", tmpDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove Procfile from %s (%s)", tmpDir, err)
		}
	}()
	procType, err := getProcfile(tmpDir)
	actualData := api.ProcessType{}
	yaml.Unmarshal(data, &actualData)
	assert.Equal(t, err, nil)
	assert.Equal(t, procType, actualData, "data")
}

func TestGetProcfileFromRepoFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}
	data := []byte("web= example-go")
	if err := os.WriteFile(tmpDir+"/Procfile", data, 0644); err != nil {
		t.Fatalf("error creating %s/Procfile (%s)", tmpDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove Procfile from %s (%s)", tmpDir, err)
		}
	}()
	_, err = getProcfile(tmpDir)

	assert.True(t, err != nil, "no error received when there should have been")
}

func TestGetProcfileFromServerSuccess(t *testing.T) {
	data := []byte("")
	expect, _ := getProcfile("")
	actualData := api.ProcessType{}
	yaml.Unmarshal(data, &actualData)
	assert.Equal(t, expect, actualData)
}

func TestPrettyPrintJSON(t *testing.T) {
	f := testJSONStruct{Foo: "bar"}
	output, err := prettyPrintJSON(f)
	if err != nil {
		t.Errorf("expected error to be nil, got '%v'", err)
	}
	expected := `{
  "foo": "bar"
}
`
	if output != expected {
		t.Errorf("expected\n%s, got\n%s", expected, output)
	}
	output, err = prettyPrintJSON(testJSONStruct{})
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
	}
	expected = "{}\n"
	if output != expected {
		t.Errorf("expected\n%s, got\n%s", expected, output)
	}
}

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.DefaultLogger.SetDebug(true)
	log.DefaultLogger.SetStdout(&buf)
	f()
	return buf.String()
}

func TestRunCmd(t *testing.T) {
	cmd := exec.Command("ls")
	if err := run(cmd); err != nil {
		t.Errorf("expected error to be nil, got %v", err)
	}

	// test log output
	output := captureOutput(func() {
		run(cmd)
	})
	expected := "running [ls]\n"
	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
	cmd.Dir = "/"
	expected = "running [ls] in directory /\n"
	output = captureOutput(func() {
		run(cmd)
	})
	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
}

func TestBuildBuilderPodNodeSelector(t *testing.T) {
	emptyNodeSelector := make(map[string]string)

	cazes := []podSelectorBuildCase{
		{"", emptyNodeSelector},
		{"pool:worker", map[string]string{"pool": "worker"}},
		{"pool:worker,network:fast", map[string]string{"pool": "worker", "network": "fast"}},
		{"pool:worker ,network:fast, disk:ssd", map[string]string{"pool": "worker", "network": "fast", "disk": "ssd"}},
	}

	for _, caze := range cazes {
		output, err := buildBuilderPodNodeSelector(caze.Config)
		assert.Nil(t, err, "error")
		assert.Equal(t, output, caze.Output, "pod selector")
	}

	_, err := buildBuilderPodNodeSelector("invalidformat")
	assert.NotEqual(t, err, nil, "invalid format")
}
