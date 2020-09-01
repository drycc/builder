package gitreceive

import (
	"encoding/json"
	"fmt"
	"github.com/drycc/controller-sdk-go/api"
	"github.com/drycc/pkg/log"
	"io/ioutil"
	"os"
	"strings"
)

// defaultStacks is default stacks json, order represents priority
var defaultStacks = `[
    {
        "name": "container",
        "image": "drycc/container:canary"
    },
    {
        "name": "heroku-20",
        "image": "drycc/slugrunner:canary.heroku-20"
    },
    {
        "name": "heroku-18",
        "image": "drycc/slugrunner:canary.heroku-18"
    }

]`

// Stacks for drycc
var Stacks []map[string]string

// initStack load stack by config
func initStack() error {
	data, err := ioutil.ReadFile("/etc/slugbuilder/images.json")
	if err == nil {
		var stacksSlugbuilder []map[string]string
		err = json.Unmarshal(data, &stacksSlugbuilder)
		if err == nil {
			data, err = ioutil.ReadFile("/etc/dockerbuilder/images.json")
			if err == nil {
				var stacksDockerbuilder []map[string]string
				err = json.Unmarshal(data, &stacksDockerbuilder)
				if err == nil {
					// Stacks order represents priority
					Stacks = stacksDockerbuilder
					Stacks = append(Stacks, stacksSlugbuilder...)
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
			if strings.Contains(stack["name"], "container") {
				return stack
			}
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/Procfile", dirName)); err == nil {
		for _, stack := range Stacks {
			if strings.Contains(stack["name"], "heroku") {
				return stack
			}
		}
	}
	return Stacks[0]
}
