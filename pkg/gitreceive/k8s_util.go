package gitreceive

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/drycc/builder/pkg/k8s"
	"github.com/drycc/controller-sdk-go/api"
	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	tarPath                = "TAR_PATH"
	debugKey               = "DRYCC_DEBUG"
	sourceVersion          = "SOURCE_VERSION"
	imagebuilderConfig     = "imagebuilder-config"
	imagebuilderConfigPath = "/etc/imagebuilder"
)

func imagebuilderJobName(appName, shortSha string) string {
	uid := uuid.New().String()[:8]
	// NOTE(bacongobbler): pod names cannot exceed 63 characters in length, so we truncate
	// the application name to stay under that limit when adding all the extra metadata to the name
	if len(appName) > 33 {
		appName = appName[:33]
	}
	return fmt.Sprintf("imagebuild-%s-%s-%s", appName, shortSha, uid)
}

func createBuilderJob(
	debug bool,
	name,
	namespace string,
	config []api.ConfigValue,
	tarKey,
	gitShortHash string,
	imageName,
	builderName,
	builderImage string,
	builderImageEnv map[string]string,
	pullPolicy corev1.PullPolicy,
	securityContext corev1.SecurityContext,
	nodeSelector map[string]string,
) *batchv1.Job {

	job := buildJob(debug, name, namespace, builderName, pullPolicy, securityContext, nodeSelector, config)
	job.Spec.Template.Spec.Containers[0].Name = builderName
	job.Spec.Template.Spec.Containers[0].Image = builderImage

	addEnvToJob(job, tarPath, tarKey)
	addEnvToJob(job, sourceVersion, gitShortHash)
	addEnvToJob(job, "IMAGE_NAME", imageName)

	for key, value := range builderImageEnv {
		addEnvToJob(job, key, value)
	}

	return &job
}

func newInt32(i int32) *int32 {
	return &i
}

func buildJob(
	debug bool,
	name,
	namespace,
	builderName string,
	pullPolicy corev1.PullPolicy,
	securityContext corev1.SecurityContext,
	nodeSelector map[string]string,
	values []api.ConfigValue) batchv1.Job {
	TTLSecondsAfterFinished := newInt32(21600)
	if os.Getenv("TTL_SECONDS_AFTER_FINISHED") != "" {
		ttl, err := strconv.ParseInt(os.Getenv("TTL_SECONDS_AFTER_FINISHED"), 10, 32)
		if err == nil {
			TTLSecondsAfterFinished = newInt32(int32(ttl))
		}
	}

	job := batchv1.Job{
		Spec: batchv1.JobSpec{
			BackoffLimit:            newInt32(0),
			TTLSecondsAfterFinished: TTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      builderName,
						"job-name": name,
						"heritage": "drycc",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ImagePullPolicy: pullPolicy,
							SecurityContext: &securityContext,
						},
					},
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":      builderName,
				"job-name": name,
				"heritage": "drycc",
			},
		},
	}
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	job.Spec.Template.Spec.Containers[0].ImagePullPolicy = pullPolicy
	job.Spec.Template.Spec.Containers[0].SecurityContext = &securityContext

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: imagebuilderConfig,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: imagebuilderConfig,
				},
			},
		},
	})

	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      imagebuilderConfig,
		MountPath: imagebuilderConfigPath,
		ReadOnly:  true,
	})

	if len(job.Spec.Template.Spec.Containers) > 0 {
		cvs := []api.ConfigVar{}
		// only take the value of the global group
		for _, v := range values {
			if v.Group == "global" {
				cvs = append(cvs, v.ConfigVar)
			}
		}
		for _, v := range cvs {
			job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  v.Name,
				Value: fmt.Sprintf("%v", v.Value),
			})
		}
	}

	if len(nodeSelector) > 0 {
		job.Spec.Template.Spec.NodeSelector = nodeSelector
	}

	if debug {
		addEnvToJob(job, debugKey, "1")
	}

	return job
}

func addEnvToJob(job batchv1.Job, key, value string) {
	if len(job.Spec.Template.Spec.Containers) > 0 {
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
}

// waitForPod waits for a pod in state running, succeeded or failed
func waitForPod(pw *k8s.PodWatcher, jobName string, ticker, interval, timeout time.Duration) error {
	condition := func(pod *corev1.Pod) (bool, error) {
		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return true, fmt.Errorf("giving up; pod went into failed status: \n[%s]:%s", pod.Status.Reason, pod.Status.Message)
		}
		return false, nil
	}

	quit := progress("...", ticker)
	err := waitForPodCondition(pw, jobName, condition, interval, timeout)
	quit <- true
	<-quit
	return err
}

// waitForPodEnd waits for a pod in state succeeded or failed
func waitForPodEnd(pw *k8s.PodWatcher, jobName string, interval, timeout time.Duration) error {
	condition := func(pod *corev1.Pod) (bool, error) {
		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return true, nil
		}
		return false, nil
	}

	return waitForPodCondition(pw, jobName, condition, interval, timeout)
}

// waitForPodCondition waits for a pod in state defined by a condition (func)
func waitForPodCondition(pw *k8s.PodWatcher, jobName string, condition func(pod *corev1.Pod) (bool, error),
	interval, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(context.Context) (done bool, err error) {
		selector := labels.Set{
			"job-name": jobName,
			"heritage": "drycc",
		}.AsSelector()
		pods, err := pw.Store.List(selector)
		if err != nil || len(pods) == 0 {
			return false, nil
		}
		done, err = condition(pods[0])
		if err != nil {
			return false, err
		}
		if done {
			return true, nil
		}
		return false, nil
	})
}

func progress(msg string, interval time.Duration) chan bool {
	tick := time.NewTicker(interval)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-quit:
				close(quit)
				return
			case <-tick.C:
				fmt.Println(msg)
			}
		}
	}()
	return quit
}

func createAppEnvConfigSecret(secretsClient typedcorev1.SecretInterface, secretName string, env map[string]interface{}) error {
	newSecret := new(corev1.Secret)
	newSecret.Name = secretName
	newSecret.Type = corev1.SecretTypeOpaque
	newSecret.Data = make(map[string][]byte)
	for k, v := range env {
		newSecret.Data[k] = []byte(fmt.Sprintf("%v", v))
	}
	if _, err := secretsClient.Create(context.TODO(), newSecret, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			_, err = secretsClient.Update(context.TODO(), newSecret, metav1.UpdateOptions{})
			return err
		}
		return err
	}
	return nil
}
