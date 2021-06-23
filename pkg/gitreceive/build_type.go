package gitreceive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/drycc/controller-sdk-go/api"
	"github.com/drycc/pkg/log"
)

// defaultStacks is default stacks json, order represents priority
var defaultStacks = `[
	{
        "name": "buildpack",
        "image": "drycc/buildpacker:canary"
    },
    {
        "name": "container",
        "image": "drycc/imagebuilder:canary"
    }

]`

// Stacks for drycc
var Stacks []map[string]string

// initStack load stack by config
func initStack() error {
	data, err := ioutil.ReadFile("/etc/buildpacker/images.json")
	if err == nil {
		var stacksBuildpacker []map[string]string
		err = json.Unmarshal(data, &stacksBuildpacker)
		if err == nil {
			data, err = ioutil.ReadFile("/etc/imagebuilder/images.json")
			if err == nil {
				var stacksImagebuilder []map[string]string
				err = json.Unmarshal(data, &stacksImagebuilder)
				if err == nil {
					// Stacks order represents priority
					Stacks = stacksImagebuilder
					Stacks = append(Stacks, stacksBuildpacker...)
				}
				return nil
			}
		}
	}
	return json.Unmarshal([]byte(defaultStacks), &Stacks)
}

func getStack(dirName string, config api.Config) map[string]string {
	if len(Stacks) == 0 {
		initStack()
	}
	log.Debug("Stacks: %s", Stacks)
	log.Debug("Config values %s", config.Values)
	if stackInterface, ok := config.Values["DRYCC_STACK"]; ok {
		if strStack, ok := stackInterface.(string); ok {
			for _, stack := range Stacks {
				if stack["name"] == strStack {
					return stack
				}
			}
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/Dockerfile", dirName)); err == nil {
		for _, stack := range Stacks {
			if stack["name"] == "container" {
				return stack
			}
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/Procfile", dirName)); err == nil {
		for _, stack := range Stacks {
			if stack["name"] == "buildpack" {
				return stack
			}
		}
	} else if _, err := os.Stat(fmt.Sprintf("%s/project.toml", dirName)); err == nil {
		for _, stack := range Stacks {
			if stack["name"] == "buildpack" {
				return stack
			}
		}
	}
	return Stacks[0]
}
