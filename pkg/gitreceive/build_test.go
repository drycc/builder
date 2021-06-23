package gitreceive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arschles/assert"

	//"github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/storage/driver/factory"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	builderconf "github.com/drycc/builder/pkg/conf"
	"github.com/drycc/builder/pkg/storage"
	"github.com/drycc/builder/pkg/sys"
	"github.com/drycc/controller-sdk-go/api"
	"github.com/drycc/pkg/log"
	"gopkg.in/yaml.v2"
)

const (
	objKey = "myobj"
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
	fs := sys.NewFakeFS()
	// NOTE(bacongobbler): there's a little easter egg here... ;)
	sha := "0462cef5812ce31fe12f25596ff68dc614c708af"

	tmpDir, err := ioutil.TempDir("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove tmpdir (%s)", err)
		}
	}()

	config.GitHome = tmpDir

	storageDriver, err := factory.Create("inmemory", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := build(config, storageDriver, nil, fs, env, "foo", sha); err == nil {
		t.Error("expected running build() without setting config.ImagebuilderImagePullPolicy to fail")
	}

	config.ImagebuilderImagePullPolicy = "Always"
	if err := build(config, storageDriver, nil, fs, env, "foo", sha); err == nil {
		t.Error("expected running build() without setting config.BuildpackerImagePullPolicy to fail")
	}

	config.BuildpackerImagePullPolicy = "Always"

	err = build(config, storageDriver, nil, fs, env, "foo", "abc123")
	expected := "git sha abc123 was invalid"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%v'", expected, err.Error())
	}

	if err := build(config, storageDriver, nil, fs, env, "foo", sha); err == nil {
		t.Error("expected running build() without valid controller client info to fail")
	}

	config.ControllerHost = "localhost"
	config.ControllerPort = "1234"

	if err := build(config, storageDriver, nil, fs, env, "foo", sha); err == nil {
		t.Error("expected running build() without a valid builder key to fail")
	}

	builderconf.BuilderKeyLocation = filepath.Join(tmpDir, "builder-key")
	data := []byte("testbuilderkey")
	if err := ioutil.WriteFile(builderconf.BuilderKeyLocation, data, 0644); err != nil {
		t.Fatalf("error creating %s (%s)", builderconf.BuilderKeyLocation, err)
	}

	if err := build(config, storageDriver, nil, fs, env, "foo", sha); err == nil {
		t.Error("expected running build() without a valid controller connection to fail")
	}
}

func TestRepoCmd(t *testing.T) {
	cmd := repoCmd("/tmp", "ls")
	if cmd.Dir != "/tmp" {
		t.Errorf("expected '%s', got '%s'", "/tmp", cmd.Dir)
	}
}

func TestGetProcfileFromRepoSuccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}
	data := []byte("web: example-go")
	if err := ioutil.WriteFile(tmpDir+"/Procfile", data, 0644); err != nil {
		t.Fatalf("error creating %s/Procfile (%s)", tmpDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove Procfile from %s (%s)", tmpDir, err)
		}
	}()
	getter := &storage.FakeObjectGetter{}
	config := api.Config{}
	config.Values = map[string]interface{}{
		"DRYCC_STACK": "buildpack",
	}
	procType, err := getProcfile(getter, tmpDir, getStack(tmpDir, config))
	actualData := api.ProcessType{}
	yaml.Unmarshal(data, &actualData)
	assert.NoErr(t, err)
	assert.Equal(t, procType, actualData, "data")
}

func TestGetProcfileFromRepoFailure(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}
	data := []byte("web= example-go")
	if err := ioutil.WriteFile(tmpDir+"/Procfile", data, 0644); err != nil {
		t.Fatalf("error creating %s/Procfile (%s)", tmpDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove Procfile from %s (%s)", tmpDir, err)
		}
	}()
	getter := &storage.FakeObjectGetter{}
	config := api.Config{}
	config.Values = map[string]interface{}{
		"DRYCC_STACK": "buildpack",
	}
	_, err = getProcfile(getter, tmpDir, getStack(tmpDir, config))

	assert.True(t, err != nil, "no error received when there should have been")
}

func TestGetProcfileFromServerSuccess(t *testing.T) {
	data := []byte("web: example-go")
	getter := &storage.FakeObjectGetter{
		Fn: func(context.Context, string) ([]byte, error) {
			return data, nil
		},
	}
	tmpDir := os.TempDir()
	config := api.Config{}
	config.Values = map[string]interface{}{
		"DRYCC_STACK": "buildpack",
	}

	_, err := getProcfile(getter, "", getStack(tmpDir, config))
	actualData := api.ProcessType{}
	yaml.Unmarshal(data, &actualData)
	assert.Err(t, err, fmt.Errorf("no Procfile can be matched in (%s)", ""))
}

func TestGetProcfileFromServerFailure(t *testing.T) {
	expectedErr := errors.New("test error")
	getter := &storage.FakeObjectGetter{
		Fn: func(context.Context, string) ([]byte, error) {
			return []byte("web: example-go"), expectedErr
		},
	}
	tmpDir := os.TempDir()
	config := api.Config{}
	config.Values = map[string]interface{}{
		"DRYCC_STACK": "buildpack",
	}
	_, err := getProcfile(getter, "", getStack(tmpDir, config))
	assert.Err(t, err, fmt.Errorf("no Procfile can be matched in (%s)", ""))
	assert.True(t, err != nil, "no error received when there should have been")
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
	assert.ExistsErr(t, err, "invalid format")
}
