package main

import (
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
	"github.com/urfave/cli"
)

const (
	serverConfAppName     = "drycc-builder-server"
	gitReceiveConfAppName = "drycc-builder-git-receive"
	gitHomeDir            = "/home/git"
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

	app.Commands = []cli.Command{
		{
			Name:    "server",
			Aliases: []string{"srv"},
			Usage:   "Run the git server",
			Action: func(c *cli.Context) {
				cnf := new(sshd.Config)
				if err := envconfig.Process(serverConfAppName, cnf); err != nil {
					pkglog.Err("getting config for %s [%s]", serverConfAppName, err)
					os.Exit(1)
				}
				fs := sys.RealFS()
				env := sys.RealEnv()
				pushLock := sshd.NewInMemoryRepositoryLock(cnf.GitLockTimeout())
				circ := sshd.NewCircuit()

				storageParams, err := conf.GetStorageParams(env)
				if err != nil {
					log.Printf("Error getting storage parameters (%s)", err)
					os.Exit(1)
				}
				var storageDriver storagedriver.StorageDriver
				storageDriver, err = factory.Create("s3", storageParams)

				if err != nil {
					log.Printf("Error creating storage driver (%s)", err)
					os.Exit(1)
				}

				kubeClient, err := k8s.NewInCluster()
				if err != nil {
					log.Printf("Error getting kubernetes client [%s]", err)
					os.Exit(1)
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
					log.Printf("Error running health server (%s)", err)
					os.Exit(1)
				case i := <-sshCh:
					log.Printf("Unexpected SSH server stop with code %d", i)
					os.Exit(i)
				case err := <-cleanerErrCh:
					log.Printf("Error running the deleted app cleaner (%s)", err)
					os.Exit(1)
				}
			},
		},
		{
			Name:    "git-receive",
			Aliases: []string{"gr"},
			Usage:   "Run the git-receive hook",
			Action: func(c *cli.Context) {
				cnf := new(gitreceive.Config)
				if err := envconfig.Process(gitReceiveConfAppName, cnf); err != nil {
					log.Printf("Error getting config for %s [%s]", gitReceiveConfAppName, err)
					os.Exit(1)
				}
				cnf.CheckDurations()
				fs := sys.RealFS()
				env := sys.RealEnv()
				storageParams, err := conf.GetStorageParams(env)
				if err != nil {
					log.Printf("Error getting storage parameters (%s)", err)
					os.Exit(1)
				}
				var storageDriver storagedriver.StorageDriver
				storageDriver, err = factory.Create("s3", storageParams)

				if err != nil {
					log.Printf("Error creating storage driver (%s)", err)
					os.Exit(1)
				}

				if err := gitreceive.Run(cnf, fs, env, storageDriver); err != nil {
					log.Printf("Error running git receive hook [%s]", err)
					os.Exit(1)
				}
			},
		},
	}

	app.Run(os.Args)
}
