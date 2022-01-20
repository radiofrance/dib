package kaniko_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/radiofrance/dib/kaniko"
)

const dockerSecretName = "some_kubernetes_secret_name" //nolint:gosec

func Test_KubernetesExecutor_ExecuteRequiresDockerSecret(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.JobConfig{})

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "the DockerConfigSecret option is required")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidContainerYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.JobConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.JobConfig = kaniko.JobConfig{
		ContainerOverride: "{\n",
	}

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "invalid yaml override for type *v1.Container: unexpected EOF")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidPodTemplateYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.JobConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.JobConfig = kaniko.JobConfig{
		PodTemplateOverride: "{\n",
	}

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "invalid yaml override for type *v1.PodTemplateSpec: unexpected EOF")
}

func Test_KubernetesExecutor_Execute(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	jobConfig := kaniko.JobConfig{
		Name: "name-overridden-by-name-generator",
		NameGenerator: func() string {
			return "kaniko-job"
		},
		Namespace: "kaniko-ns",
		Labels: map[string]string{
			"some_label": "some_value",
		},
		Image: "my-kaniko-image:tag",
		ImagePullSecrets: []string{
			"my-pull-secret",
		},
		Env: map[string]string{
			"MY_ENV_VAR": "env_value",
		},
		EnvSecrets: []string{
			"my-env-secret",
		},
		ContainerOverride: `
resources:
  limits:
    cpu: 2
  requests:
    memory: 1Gi
`,
		PodTemplateOverride: `
spec:
  restartPolicy: OnFailure
`,
	}

	expectedLabels := map[string]string{
		"app.kubernetes.io/name":      "kaniko",
		"app.kubernetes.io/component": "build-job",
		"app.kubernetes.io/instance":  "kaniko-job",
		"some_label":                  "some_value",
	}

	executor := kaniko.NewKubernetesExecutor(clientSet, jobConfig)
	executor.DockerConfigSecret = dockerSecretName

	go func() {
		// Wait for the Job to be created before running assertions
		<-time.After(1 * time.Second)

		// Check the created Job
		job, err := clientSet.BatchV1().Jobs("kaniko-ns").Get(context.Background(), "kaniko-job", metav1.GetOptions{})
		require.NoError(t, err)

		// Job assertions
		assert.Equal(t, expectedLabels, job.Labels)
		assert.Len(t, job.Spec.Template.Spec.Containers, 1)

		// Pod template assertions
		assert.Equal(t, expectedLabels, job.Spec.Template.Labels)
		assert.Contains(t, job.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: "my-pull-secret",
		})
		assert.Equal(t, corev1.RestartPolicyOnFailure, job.Spec.Template.Spec.RestartPolicy)

		assert.Len(t, job.Spec.Template.Spec.Volumes, 1)
		volume := job.Spec.Template.Spec.Volumes[0]
		assert.Equal(t, dockerSecretName, volume.Name)
		assert.Equal(t, dockerSecretName, volume.VolumeSource.Secret.SecretName)
		assert.Equal(t, int32(420), *volume.VolumeSource.Secret.DefaultMode)

		// Container assertions
		container := job.Spec.Template.Spec.Containers[0]
		assert.ElementsMatch(t, container.Args, []string{
			"kaniko-arg1",
			"kaniko-arg2",
		})
		assert.ElementsMatch(t, container.Env, []corev1.EnvVar{
			{
				Name:  "DOCKER_CONFIG",
				Value: "/kaniko/.docker",
			},
			{
				Name:  "MY_ENV_VAR",
				Value: "env_value",
			},
		})
		assert.Len(t, container.EnvFrom, 1)
		assert.Equal(t, "my-env-secret", container.EnvFrom[0].SecretRef.Name)

		assert.True(t, container.Resources.Limits[corev1.ResourceCPU].Equal(resource.MustParse("2")))
		assert.True(t, container.Resources.Requests[corev1.ResourceMemory].Equal(resource.MustParse("1Gi")))

		assert.Equal(t, "my-kaniko-image:tag", container.Image)

		assert.ElementsMatch(t, container.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      dockerSecretName,
				MountPath: "/kaniko/.docker",
				ReadOnly:  true,
			},
		})

		simulateJobExecution(t, clientSet, job, true)
	}()

	// Run the executor
	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})
	assert.NoError(t, err)

	// Check the Job was deleted by the executor before returning.
	_, err = clientSet.BatchV1().Jobs("kaniko-ns").Get(context.Background(), "kaniko-job", metav1.GetOptions{})
	assert.Error(t, err)
}

// simulateJobExecution simulates the default behaviour of a Kubernetes job controller
// by creating a pod, and also simulates the pod lifecycle until it reaches completion.
func simulateJobExecution(t *testing.T, clientSet kubernetes.Interface, job *batchv1.Job, isSuccess bool) {
	t.Helper()

	// Create a pod and set the job status to active
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kaniko-job-pod-1",
			Namespace: job.Namespace,
			Labels: map[string]string{
				"job-name": job.Name,
			},
		},
	}
	pod, err := clientSet.CoreV1().Pods(job.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err)

	job.Status.Active = 1
	job, err = clientSet.BatchV1().Jobs(job.Namespace).Update(context.Background(), job, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Wait a moment to simulate the pod taking time to complete its task.
	<-time.After(3 * time.Second)

	// Set pod status to completed
	pod.Status.Phase = corev1.PodSucceeded
	_, err = clientSet.CoreV1().Pods(job.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Set the job status to Succeeded or Failed
	job.Status.Active = 0
	if isSuccess {
		job.Status.Succeeded = 1
	} else {
		job.Status.Failed = 1
	}
	_, err = clientSet.BatchV1().Jobs(job.Namespace).Update(context.Background(), job, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func Test_UniqueJobName(t *testing.T) {
	t.Parallel()

	dataset := []struct {
		identifier     string
		expectedPrefix string
	}{
		{
			identifier:     "dib",
			expectedPrefix: "kaniko-dib-",
		},
		{
			identifier:     "semicolon:slashes/dib",
			expectedPrefix: "kaniko-semicolon-slashes-dib-",
		},
		{
			identifier:     "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
			expectedPrefix: "kaniko-veryveryveryveryveryveryveryveryveryveryveryver-",
		},
	}

	// Only alphanumeric characters, or dashes, maximum 63 chars
	validationRegexp := regexp.MustCompile(`^[a-z0-9\-]{1,63}`)

	for _, ds := range dataset {
		jobName := kaniko.UniqueJobName(ds.identifier)

		assert.Truef(t, strings.HasPrefix(jobName, ds.expectedPrefix),
			"Job name %s does not have prefix %s", jobName, ds.expectedPrefix)

		assert.Regexp(t, validationRegexp, jobName)
	}
}
