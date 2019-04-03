package gitreceive

import (
	"fmt"
	"github.com/drycc/controller-sdk-go/api"
	"os"
)

type buildType string

func (b buildType) String() string {
	return string(b)
}

const (
	buildTypeSlugbuilder   buildType = "slugbuilder"
	buildTypeDockerbuilder buildType = "dockerbuilder"
)

func getBuildType(dirName string, config api.Config) buildType {

	hasDockerfile := false
	if _, err := os.Stat(fmt.Sprintf("%s/Dockerfile", dirName)); err == nil {
		hasDockerfile = true
	}

	hasProcfile := false
	if _, err := os.Stat(fmt.Sprintf("%s/Procfile", dirName)); err == nil {
		hasProcfile = true
	}
	if hasDockerfile && hasProcfile {
		if bTypeInterface, ok := config.Values["DRYCC_BUILDER"]; ok {
			if strType, ok := bTypeInterface.(string); ok {
				bType := buildType(strType)
				if bType == buildTypeSlugbuilder || bType == buildTypeDockerbuilder {
					return bType
				}
			}
		}
	} else if hasProcfile {
		return buildTypeSlugbuilder
	}
	return buildTypeDockerbuilder
}
