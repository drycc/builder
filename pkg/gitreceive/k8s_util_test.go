package gitreceive

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/drycc/builder/pkg/k8s"
	"github.com/drycc/controller-sdk-go/api"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestImagebuilderPodName(t *testing.T) {
	name := imagebuilderJobName("demo", "12345678")
	if !strings.HasPrefix(name, "imagebuild-demo-12345678-") {
		t.Errorf("expected pod name imagebuild-demo-12345678-*, got %s", name)
	}

	name = imagebuilderJobName("this-name-has-more-than-24-characters-in-length", "12345678")
	if !strings.HasPrefix(name, "imagebuild-this-name-has-more-than-24-charac-12345678-") {
		t.Errorf("expected pod name imagebuild-this-name-has-more-than-24-charac-12345678-*, got %s", name)
	}
	if len(name) > 63 {
		t.Errorf("expected imagebuilder pod name length to be <= 63 characters in length, got %d", len(name))
	}
}

type imageBuildCase struct {
	debug                       bool
	name                        string
	namespace                   string
	env                         []api.ConfigValue
	tarKey                      string
	gitShortHash                string
	imgName                     string
	imagebuilderName            string
	imagebuilderImage           string
	imagebuilderImagePullPolicy corev1.PullPolicy
	builderPodNodeSelector      map[string]string
}

func TestBuildJob(t *testing.T) {
	emptyValues := []api.ConfigValue{}

	values := []api.ConfigValue{
		{
			Group: "global",
			ConfigVar: api.ConfigVar{
				Name:  "KEY",
				Value: "VALUE",
			},
		},
	}
	var job *batchv1.Job

	emptyNodeSelector := make(map[string]string)

	nodeSelector1 := make(map[string]string)
	nodeSelector1["disk"] = "ssd"

	nodeSelector2 := make(map[string]string)
	nodeSelector2["disk"] = "magnetic"
	nodeSelector2["network"] = "fast"

	imageBuilds := []imageBuildCase{
		{true, "test", "default", emptyValues, "tar", "deadbeef", "imagebuilder", "", "", corev1.PullAlways, nodeSelector1},
		{true, "test", "default", values, "tar", "deadbeef", "", "imagebuilder", "", corev1.PullAlways, nodeSelector2},
		{true, "test", "default", emptyValues, "tar", "deadbeef", "img", "imagebuilder", "", corev1.PullAlways, emptyNodeSelector},
		{true, "test", "default", values, "tar", "deadbeef", "img", "imagebuilder", "", corev1.PullAlways, emptyNodeSelector},
		{true, "test", "default", values, "tar", "deadbeef", "img", "imagebuilder", "customimage", corev1.PullAlways, emptyNodeSelector},
		{true, "test", "default", values, "tar", "deadbeef", "img", "imagebuilder", "customimage", corev1.PullIfNotPresent, emptyNodeSelector},
		{true, "test", "default", values, "tar", "deadbeef", "img", "imagebuilder", "customimage", corev1.PullNever, nil},
	}
	buildImageEnv := map[string]string{"DRYCC_REGISTRY_LOCATION": "on-cluster"}
	for _, build := range imageBuilds {
		job = createBuilderJob(
			build.debug,
			build.name,
			build.namespace,
			build.env,
			build.tarKey,
			build.gitShortHash,
			build.imgName,
			build.imagebuilderName,
			build.imagebuilderImage,
			buildImageEnv,
			build.imagebuilderImagePullPolicy,
			k8s.SecurityContextFromPrivileged(false),
			build.builderPodNodeSelector,
		)

		if job.ObjectMeta.Name != build.name {
			t.Errorf("expected %v but returned %v ", build.name, job.ObjectMeta.Name)
		}
		if job.ObjectMeta.Namespace != build.namespace {
			t.Errorf("expected %v but returned %v ", build.namespace, job.ObjectMeta.Namespace)
		}

		checkForEnv(t, job, "SOURCE_VERSION", build.gitShortHash)
		checkForEnv(t, job, "TAR_PATH", build.tarKey)
		checkForEnv(t, job, "IMAGE_NAME", build.imgName)
		checkForEnv(t, job, "DRYCC_REGISTRY_LOCATION", "on-cluster")
		if build.imagebuilderImage != "" {
			if job.Spec.Template.Spec.Containers[0].Image != build.imagebuilderImage {
				t.Errorf("expected %v but returned %v", build.imagebuilderImage, job.Spec.Template.Spec.Containers[0].Image)
			}
		}
		if build.imagebuilderImagePullPolicy != "" {
			if job.Spec.Template.Spec.Containers[0].ImagePullPolicy != "" {
				if job.Spec.Template.Spec.Containers[0].ImagePullPolicy != build.imagebuilderImagePullPolicy {
					t.Errorf("expected %v but returned %v", build.imagebuilderImagePullPolicy, job.Spec.Template.Spec.Containers[0].ImagePullPolicy)
				}
			}
		}

		if len(job.Spec.Template.Spec.NodeSelector) > 0 || len(build.builderPodNodeSelector) > 0 {
			assert.Equal(t, job.Spec.Template.Spec.NodeSelector, build.builderPodNodeSelector, "node selector")
		}
	}
}

func checkForEnv(t *testing.T, job *batchv1.Job, key, expVal string) {
	val, err := envValueFromKey(job, key)
	if err != nil {
		t.Errorf("%v", err)
	}
	if expVal != val {
		t.Errorf("expected %v but returned %v ", expVal, val)
	}
}

func envValueFromKey(job *batchv1.Job, key string) (string, error) {
	for _, env := range job.Spec.Template.Spec.Containers[0].Env {
		if env.Name == key {
			return env.Value, nil
		}
	}

	return "", fmt.Errorf("no key with name %v found in pod env", key)
}

func TestCreateAppEnvConfigSecretErr(t *testing.T) {
	expectedErr := errors.New("get secret error")
	secretsClient := &k8s.FakeSecret{
		FnCreate: func(*corev1.Secret) (*corev1.Secret, error) {
			return &corev1.Secret{}, expectedErr
		},
	}
	err := createAppEnvConfigSecret(secretsClient, "test", nil)
	assert.Error(t, err, expectedErr)
}

func TestCreateAppEnvConfigSecretSuccess(t *testing.T) {
	secretsClient := &k8s.FakeSecret{
		FnCreate: func(*corev1.Secret) (*corev1.Secret, error) {
			return &corev1.Secret{}, nil
		},
	}
	err := createAppEnvConfigSecret(secretsClient, "test", nil)
	assert.Equal(t, err, nil)
}

func TestCreateAppEnvConfigSecretAlreadyExists(t *testing.T) {
	alreadyExistErr := apierrors.NewAlreadyExists(corev1.Resource("tests"), "1")
	secretsClient := &k8s.FakeSecret{
		FnCreate: func(*corev1.Secret) (*corev1.Secret, error) {
			return &corev1.Secret{}, alreadyExistErr
		},
		FnUpdate: func(*corev1.Secret) (*corev1.Secret, error) {
			return &corev1.Secret{}, nil
		},
	}
	err := createAppEnvConfigSecret(secretsClient, "test", nil)
	assert.Equal(t, err, nil)
}
