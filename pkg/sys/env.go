package sys

import (
	"os"
	"strings"
)

// Env is an interface to a set of environment variables.
type Env interface {
	// Get gets the environment variable of the given name.
	Get(name string) string
	Environ(prefixs []string) map[string]string
}

type realEnv struct{}

func (r realEnv) Get(name string) string {
	return os.Getenv(name)
}

func (r realEnv) Environ(prefixs []string) map[string]string {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		for index := range prefixs {
			if _, hasKey := envs[pair[0]]; !hasKey && strings.HasPrefix(pair[0], prefixs[index]) {
				envs[pair[0]] = pair[1]
			}
		}
	}
	return envs
}

// RealEnv returns an Env implementation that uses os.Getenv every time Get is called.
func RealEnv() Env {
	return realEnv{}
}

// FakeEnv is an Env implementation that stores the environment in a map.
type FakeEnv struct {
	// Envs is the map from which Get calls will look to retrieve environment variables.
	Envs map[string]string
}

// NewFakeEnv returns a new FakeEnv with no values in Envs.
func NewFakeEnv() *FakeEnv {
	return &FakeEnv{Envs: make(map[string]string)}
}

// Get is the Env interface implementation for Env.
func (f *FakeEnv) Get(name string) string {
	return f.Envs[name]
}

func (f *FakeEnv) Environ(prefixs []string) map[string]string {
	envs := make(map[string]string)
	for key := range f.Envs {
		for index := range prefixs {
			if _, hasKey := envs[key]; !hasKey && strings.HasPrefix(key, prefixs[index]) {
				envs[key] = f.Envs[key]
			}
		}
	}
	return envs
}
