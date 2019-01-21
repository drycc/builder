package sys

import (
	"os"
	"testing"
)

const expectedEnv string = "mmmcoffee"

func TestRealEnvGet(t *testing.T) {
	e := RealEnv()
	os.Setenv("DRYCC_BUILDER_REAL_ENV_TEST", expectedEnv)
	if actual := e.Get("DRYCC_BUILDER_REAL_ENV_TEST"); actual != expectedEnv {
		t.Errorf("expected '%s', got '%s'", expectedEnv, actual)
	}
}

func TestFakeEnvGet(t *testing.T) {
	e := NewFakeEnv()
	e.Envs["DRYCC_BUILDER_FAKE_ENV_TEST"] = expectedEnv
	if actual := e.Get("DRYCC_BUILDER_FAKE_ENV_TEST"); actual != expectedEnv {
		t.Errorf("expected '%s', got '%s'", expectedEnv, actual)
	}
}
