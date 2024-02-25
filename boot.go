package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	storagedriver "github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/s3-aws"
	"github.com/drycc/builder/pkg"
	"github.com/drycc/builder/pkg/cleaner"
	"github.com/drycc/builder/pkg/conf"
	"github.com/drycc/builder/pkg/gitreceive"
	"github.com/drycc/builder/pkg/healthsrv"
	"github.com/drycc/builder/pkg/k8s"
	"github.com/drycc/builder/pkg/sshd"
	"github.com/drycc/builder/pkg/sys"
	pkglog "github.com/drycc/pkg/log"
	"github.com/kelseyhightower/envconfig"
	"github.com/urfave/cli/v2"
)

const (
	serverConfAppName     = "drycc-builder-server"
	gitReceiveConfAppName = "drycc-builder-git-receive"
	gitHomeDir            = "/workspace"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	if os.Getenv("DRYCC_DEBUG") == "true" {
		pkglog.DefaultLogger.SetDebug(true)
		log.Printf("Running in debug mode")
	}

	app := cli.NewApp()

	app.Commands = []*cli.Command{
		{
			Name:    "server",
			Aliases: []string{"srv"},
			Usage:   "Run the git server",
			Action: func(*cli.Context) error {
				cnf := new(sshd.Config)
				if err := envconfig.Process(serverConfAppName, cnf); err != nil {
					return fmt.Errorf("getting config for %s [%s]", serverConfAppName, err)
				}
				fs := sys.RealFS()
				env := sys.RealEnv()
				pushLock := sshd.NewInMemoryRepositoryLock(cnf.GitLockTimeout())
				circ := sshd.NewCircuit()

				storageParams, err := conf.GetStorageParams(env)
				if err != nil {
					return fmt.Errorf("error getting storage parameters (%s)", err)
				}
				var storageDriver storagedriver.StorageDriver
				storageDriver, err = factory.Create(context.Background(), "s3", storageParams)

				if err != nil {
					return fmt.Errorf("error creating storage driver (%s)", err)
				}

				kubeClient, err := k8s.NewInCluster()
				if err != nil {
					return fmt.Errorf("error getting kubernetes client [%s]", err)
				}
				log.Printf("Starting health check server on port %d", cnf.HealthSrvPort)
				healthSrvCh := make(chan error)
				go func() {
					if err := healthsrv.Start(cnf, kubeClient.CoreV1().Namespaces(), storageDriver, circ); err != nil {
						healthSrvCh <- err
					}
				}()
				log.Printf("Starting deleted app cleaner")
				cleanerErrCh := make(chan error)
				go func() {
					if err := cleaner.Run(gitHomeDir, kubeClient.CoreV1().Namespaces(), fs, cnf.CleanerPollSleepDuration(), storageDriver); err != nil {
						cleanerErrCh <- err
					}
				}()

				log.Printf("Starting SSH server on %s:%d", cnf.SSHHostIP, cnf.SSHHostPort)
				sshCh := make(chan int)
				go func() {
					sshCh <- pkg.RunBuilder(cnf, gitHomeDir, circ, pushLock)
				}()

				select {
				case err := <-healthSrvCh:
					return fmt.Errorf("error running health server (%s)", err)
				case i := <-sshCh:
					return fmt.Errorf("unexpected SSH server stop with code %d", i)
				case err := <-cleanerErrCh:
					return fmt.Errorf("error running the deleted app cleaner (%s)", err)
				}
			},
		},
		{
			Name:    "git-receive",
			Aliases: []string{"gr"},
			Usage:   "Run the git-receive hook",
			Action: func(*cli.Context) error {
				cnf := new(gitreceive.Config)
				if err := envconfig.Process(gitReceiveConfAppName, cnf); err != nil {
					return fmt.Errorf("error getting config for %s [%s]", gitReceiveConfAppName, err)
				}
				cnf.CheckDurations()
				env := sys.RealEnv()
				storageParams, err := conf.GetStorageParams(env)
				if err != nil {
					return fmt.Errorf("error getting storage parameters (%s)", err)
				}
				var storageDriver storagedriver.StorageDriver
				storageDriver, err = factory.Create(context.Background(), "s3", storageParams)

				if err != nil {
					return fmt.Errorf("error creating storage driver (%s)", err)
				}

				if err := gitreceive.Run(cnf, env, storageDriver); err != nil {
					return fmt.Errorf("error running git receive hook [%s]", err)
				}
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
