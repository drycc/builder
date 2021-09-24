package gitreceive

import (
	"bytes"
	ctx "context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/distribution/distribution/v3/context"
	storagedriver "github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/drycc/builder/pkg/controller"
	"github.com/drycc/builder/pkg/git"
	"github.com/drycc/builder/pkg/k8s"
	"github.com/drycc/builder/pkg/sys"
	dryccAPI "github.com/drycc/controller-sdk-go/api"
	"github.com/drycc/controller-sdk-go/hooks"
	"github.com/drycc/pkg/log"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	// TarKeyPattern is the template for storing tar key files.
	TarKeyPattern = "%s/tar"
	// GitKeyPattern is the template for storing git key files.
	GitKeyPattern = "home/%s:git-%s"
)

// repoCmd returns exec.Command(first, others...) with its current working directory repoDir
func repoCmd(repoDir, first string, others ...string) *exec.Cmd {
	cmd := exec.Command(first, others...)
	cmd.Dir = repoDir
	return cmd
}

// run prints the command it will execute to the debug log, then runs it and returns the result
// of run
func run(cmd *exec.Cmd) error {
	cmdStr := strings.Join(cmd.Args, " ")
	if cmd.Dir != "" {
		log.Debug("running [%s] in directory %s", cmdStr, cmd.Dir)
	} else {
		log.Debug("running [%s]", cmdStr)
	}
	return cmd.Run()
}

func build(
	conf *Config,
	storageDriver storagedriver.StorageDriver,
	//kubeClient *client.Client,
	kubeClient *kubernetes.Clientset,
	fs sys.FS,
	env sys.Env,
	builderKey,
	rawGitSha string) error {

	// Rewrite regular expression, compatible with slug type
	storagedriver.PathRegexp = regexp.MustCompile(`^([A-Za-z0-9._:-]*(/[A-Za-z0-9._:-]+)*)+$`)

	repo := conf.Repository
	gitSha, err := git.NewSha(rawGitSha)
	if err != nil {
		return err
	}

	appName := conf.App()

	repoDir := filepath.Join(conf.GitHome, repo)
	buildDir := filepath.Join(repoDir, "build")

	if err := os.MkdirAll(buildDir, os.ModeDir); err != nil {
		return fmt.Errorf("making the build directory %s (%s)", buildDir, err)
	}

	tmpDir, err := ioutil.TempDir(buildDir, "tmp")
	if err != nil {
		return fmt.Errorf("unable to create tmpdir %s (%s)", buildDir, err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Info("unable to remove tmpdir %s (%s)", tmpDir, err)
		}
	}()

	client, err := controller.New(conf.ControllerHost, conf.ControllerPort)
	if err != nil {
		return err
	}

	// Get the application config from the controller, so we can check for a custom buildpack URL
	appConf, err := hooks.GetAppConfig(client, conf.Username, appName)
	if controller.CheckAPICompat(client, err) != nil {
		return err
	}

	// build a tarball from the new objects
	appTgz := fmt.Sprintf("%s.tar.gz", appName)
	gitArchiveCmd := repoCmd(repoDir, "git", "archive", "--format=tar.gz", fmt.Sprintf("--output=%s", appTgz), gitSha.Short())
	gitArchiveCmd.Stdout = os.Stdout
	gitArchiveCmd.Stderr = os.Stderr
	if err := run(gitArchiveCmd); err != nil {
		return fmt.Errorf("running %s (%s)", strings.Join(gitArchiveCmd.Args, " "), err)
	}
	absAppTgz := fmt.Sprintf("%s/%s", repoDir, appTgz)

	// untar the archive into the temp dir
	tarCmd := repoCmd(repoDir, "tar", "-xzf", appTgz, "-C", fmt.Sprintf("%s/", tmpDir))
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	if err := run(tarCmd); err != nil {
		return fmt.Errorf("running %s (%s)", strings.Join(tarCmd.Args, " "), err)
	}

	stack := getStack(tmpDir, appConf)

	appTgzdata, err := ioutil.ReadFile(absAppTgz)
	if err != nil {
		return fmt.Errorf("error while reading file %s: (%s)", appTgz, err)
	}

	tarKey := fmt.Sprintf(TarKeyPattern, fmt.Sprintf(GitKeyPattern, appName, gitSha.Short()))
	log.Debug("Uploading tar to %s", tarKey)

	if err := storageDriver.PutContent(context.Background(), tarKey, appTgzdata); err != nil {
		return fmt.Errorf("uploading %s to %s (%v)", absAppTgz, tarKey, err)
	}

	builderPodNodeSelector, err := buildBuilderPodNodeSelector(conf.BuilderPodNodeSelector)
	if err != nil {
		return fmt.Errorf("error build builder pod node selector %s", err)
	}
	builderName := "drycc-imagebuilder"
	imagePullPolicy, err := k8s.PullPolicyFromString(conf.ImagebuilderImagePullPolicy)
	if err != nil {
		return err
	}
	securityContext := k8s.SecurityContextFromPrivileged(true)

	imageName := fmt.Sprintf("%s:git-%s", appName, gitSha.Short())
	buildJobName := imagebuilderJobName(appName, gitSha.Short())
	registryLocation := conf.RegistryLocation
	builderImageEnv := make(map[string]string)
	if registryLocation != "on-cluster" {
		builderImageEnv, err = getRegistryDetails(kubeClient.CoreV1(), &imageName, registryLocation, conf.PodNamespace)
		if err != nil {
			return fmt.Errorf("error getting private registry details %s", err)
		}
	}
	builderImageEnv["DRYCC_STACK"] = stack["name"]
	builderImageEnv["DRYCC_REGISTRY_LOCATION"] = registryLocation

	job := createBuilderJob(
		conf.Debug,
		buildJobName,
		conf.PodNamespace,
		appConf.Values,
		tarKey,
		gitSha.Short(),
		imageName,
		conf.StorageType,
		builderName,
		stack["image"],
		conf.RegistryHost,
		conf.RegistryPort,
		builderImageEnv,
		imagePullPolicy,
		securityContext,
		builderPodNodeSelector,
	)

	log.Info("Starting build... but first, coffee!")
	log.Debug("Use image %s: %s", stack["name"], stack["image"])
	log.Debug("Starting job %s", buildJobName)
	json, err := prettyPrintJSON(job)
	if err == nil {
		log.Debug("Job spec: %v", json)
	} else {
		log.Debug("Error creating json representation of Job spec: %v", err)
	}
	jobsInterface := kubeClient.BatchV1().Jobs(conf.PodNamespace)

	newJob, err := jobsInterface.Create(ctx.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("creating builder pod (%s)", err)
	}

	pw := k8s.NewPodWatcher(*kubeClient, conf.PodNamespace)
	stopCh := make(chan struct{})
	defer close(stopCh)
	go pw.Controller.Run(stopCh)

	if err := waitForPod(pw, newJob.Namespace, newJob.Name, conf.SessionIdleInterval(), conf.BuilderPodTickDuration(), conf.BuilderPodWaitDuration()); err != nil {
		return fmt.Errorf("watching events for builder pod startup (%s)", err)
	}

	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("heritage=%s", newJob.Name),
	}
	podList, err := kubeClient.CoreV1().Pods(newJob.Namespace).List(context.Background(), options)
	if err != nil {
		return fmt.Errorf("list pods %s fail: (%s)", newJob.Name, err)
	}

	req := kubeClient.CoreV1().RESTClient().Get().Namespace(newJob.Namespace).Name(podList.Items[0].Name).Resource("pods").SubResource("log").VersionedParams(
		&corev1.PodLogOptions{
			Follow: true,
		}, scheme.ParameterCodec)

	rc, err := req.Stream(ctx.TODO())
	if err != nil {
		return fmt.Errorf("attempting to stream logs (%s)", err)
	}
	defer rc.Close()

	size, err := io.Copy(os.Stdout, rc)
	if err != nil {
		return fmt.Errorf("fetching builder logs (%s)", err)
	}
	log.Debug("size of streamed logs %v", size)

	log.Debug(
		"Waiting for the %s/%s pod to end. Checking every %s for %s",
		newJob.Namespace,
		newJob.Name,
		conf.BuilderPodTickDuration(),
		conf.BuilderPodWaitDuration(),
	)
	// check the state and exit code of the build pod.
	// if the code is not 0 return error
	if err := waitForPodEnd(pw, newJob.Namespace, newJob.Name, conf.BuilderPodTickDuration(), conf.BuilderPodWaitDuration()); err != nil {
		return fmt.Errorf("error getting builder pod status (%s)", err)
	}
	log.Debug("Done")
	log.Debug("Checking for builder pod exit code")
	buildPod, err := kubeClient.CoreV1().Pods(newJob.Namespace).Get(ctx.TODO(), podList.Items[0].Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting builder pod status (%s)", err)
	}

	for _, containerStatus := range buildPod.Status.ContainerStatuses {
		state := containerStatus.State.Terminated
		if state.ExitCode != 0 {
			return fmt.Errorf("build pod exited with code %d, stopping build", state.ExitCode)
		}
	}
	log.Debug("Done")

	procfile, err := getProcfile(tmpDir, stack)
	if err != nil {
		return err
	}
	dockerfile, err := getDockerfile(tmpDir, stack)
	if err != nil {
		return err
	}
	log.Info("Build complete.")

	quit := progress("...", conf.SessionIdleInterval())
	log.Info("Launching App...")
	release, err := hooks.CreateBuild(client, conf.Username, conf.App(), imageName, stack["name"], gitSha.Short(), procfile, dockerfile)
	quit <- true
	<-quit
	if controller.CheckAPICompat(client, err) != nil {
		return fmt.Errorf("The controller returned an error when publishing the release: %s", err)
	}

	log.Info("Done, %s:v%d deployed to Workflow\n", appName, release)
	log.Info("Use 'drycc open' to view this application in your browser\n")
	log.Info("To learn more, use 'drycc help' or visit https://drycc.cc/\n")

	run(repoCmd(repoDir, "git", "gc"))

	return nil
}

func buildBuilderPodNodeSelector(config string) (map[string]string, error) {
	selector := make(map[string]string)
	if config != "" {
		for _, line := range strings.Split(config, ",") {
			param := strings.Split(line, ":")
			if len(param) != 2 {
				return nil, fmt.Errorf("Invalid BuilderPodNodeSelector value format: %s", config)
			}
			selector[strings.TrimSpace(param[0])] = strings.TrimSpace(param[1])
		}
	}
	return selector, nil
}

func prettyPrintJSON(data interface{}) (string, error) {
	output := &bytes.Buffer{}
	if err := json.NewEncoder(output).Encode(data); err != nil {
		return "", err
	}
	formatted := &bytes.Buffer{}
	if err := json.Indent(formatted, output.Bytes(), "", "  "); err != nil {
		return "", err
	}
	return formatted.String(), nil
}

func getProcfile(dirName string, stack map[string]string) (dryccAPI.ProcessType, error) {
	procfile := dryccAPI.ProcessType{}
	if _, err := os.Stat(fmt.Sprintf("%s/Procfile", dirName)); err == nil {
		rawProcFile, err := ioutil.ReadFile(fmt.Sprintf("%s/Procfile", dirName))
		if err != nil {
			return nil, fmt.Errorf("error in reading %s/Procfile (%s)", dirName, err)
		}
		if err := yaml.Unmarshal(rawProcFile, &procfile); err != nil {
			return nil, fmt.Errorf("procfile %s/ProcFile is malformed (%s)", dirName, err)
		}
		return procfile, nil
	}
	return nil, fmt.Errorf("no Procfile can be matched in (%s)", dirName)
}

func getDockerfile(dirName string, stack map[string]string) (string, error) {
	if stack["name"] == "container" {
		if _, err := os.Stat(fmt.Sprintf("%s/Dockerfile", dirName)); err == nil {
			rawDockerfile, err := ioutil.ReadFile(fmt.Sprintf("%s/Dockerfile", dirName))
			if err != nil {
				return "", fmt.Errorf("error in reading %s/Dockerfile (%s)", dirName, err)
			}
			return string(rawDockerfile), nil
		}
	}
	return "", nil
}
