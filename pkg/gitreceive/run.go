package gitreceive

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	storagedriver "github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/drycc/builder/pkg/k8s"
	"github.com/drycc/builder/pkg/sys"
	"github.com/drycc/pkg/log"
)

func readLine(line string) (string, string, string, error) {
	spl := strings.Split(line, " ")
	if len(spl) != 3 {
		return "", "", "", fmt.Errorf("malformed line [%s]", line)
	}
	return spl[0], spl[1], spl[2], nil
}

// Run runs the git-receive hook. This func is effectively the main for the git-receive hook,
// although it is called from the main in boot.go.
func Run(conf *Config, env sys.Env, storageDriver storagedriver.StorageDriver) error {
	log.Debug("Running git hook")
	// kubeClient, err := client.NewInCluster()
	kubeClient, err := k8s.NewInCluster()
	if err != nil {
		return fmt.Errorf("couldn't reach the api server (%s)", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		oldRev, newRev, refName, err := readLine(line)
		if err != nil {
			return fmt.Errorf("reading STDIN (%s)", err)
		}

		log.Debug("read [%s,%s,%s]", oldRev, newRev, refName)

		// if we're processing a receive-pack on an existing repo, run a build
		if strings.HasPrefix(conf.SSHOriginalCommand, "git-receive-pack") {
			if err := build(conf, storageDriver, kubeClient, env, newRev); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
