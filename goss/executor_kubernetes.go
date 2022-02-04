package goss

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	k8sutils "github.com/radiofrance/dib/kubernetes"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodConfig hold the configuration for the kubernetes pod to create.
type PodConfig struct {
	// Kubernetes generic configuration.
	NameGenerator    func() string // A function that generates the pod name.
	Namespace        string        // The namespace where the pod should be created.
	Image            string        // The goss image.
	ImagePullSecrets []string      // A list of `imagePullSecret` secret names used to pull pod images.

	// Advanced customisations (raw YAML overrides)
	ContainerOverride string // YAML string to override the test container object.
	PodOverride       string // YAML string to override the pod object.

}

// KubernetesExecutor will run goss tests in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet  kubernetes.Interface
	restConfig rest.Config
	PodConfig  PodConfig // The default pod configuration used to run goss tests.
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(restConfig rest.Config, clientSet kubernetes.Interface, config PodConfig,
) *KubernetesExecutor {
	return &KubernetesExecutor{
		clientSet:  clientSet,
		restConfig: restConfig,
		PodConfig:  config,
	}
}

// Execute the goss test using a Kubernetes Pod.
func (e KubernetesExecutor) Execute(ctx context.Context, output io.Writer, opts types.RunTestOptions, args ...string,
) error {
	logrus.Info("Testing image with goss kubernetes executor")

	var podName string
	if e.PodConfig.NameGenerator == nil {
		podName = k8sutils.UniquePodName("goss-" + opts.ImageName)()
	} else {
		podName = e.PodConfig.NameGenerator()
	}
	containerName := "test"

	labels := map[string]string{
		"app.kubernetes.io/name":      "goss",
		"app.kubernetes.io/component": "goss-pod",
		"app.kubernetes.io/instance":  podName,
	}

	objectMeta := metav1.ObjectMeta{
		Name:      podName,
		Namespace: e.PodConfig.Namespace,
		Labels:    labels,
	}

	var imagePullSecrets []corev1.LocalObjectReference
	for _, secretName := range e.PodConfig.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: secretName,
		})
	}

	initContainer := corev1.Container{
		Name:            "setup-goss",
		Image:           e.PodConfig.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"cp", "/goss/goss", "/shared"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/shared",
				ReadOnly:  false,
			},
		},
	}
	container := corev1.Container{
		Name:            containerName,
		Image:           opts.ImageReference,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"sleep", "1h"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/goss",
				ReadOnly:  false,
			},
		},
	}
	err := k8sutils.MergeObjectWithYaml(&container, e.PodConfig.ContainerOverride)
	if err != nil {
		return err
	}

	pod := corev1.Pod{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			ImagePullSecrets: imagePullSecrets,
			InitContainers: []corev1.Container{
				initContainer,
			},
			Containers: []corev1.Container{
				container,
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "shared",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMediumMemory,
						},
					},
				},
			},
		},
	}

	err = k8sutils.MergeObjectWithYaml(&pod, e.PodConfig.PodOverride)
	if err != nil {
		return err
	}

	watcher, err := e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", pod.Name),
		Watch:         true,
	})
	if err != nil {
		return fmt.Errorf("failed to watch pod: %w", err)
	}
	defer watcher.Stop()

	readyChan, watchErrChan := k8sutils.WaitPodReady(ctx, watcher)

	errChan := make(chan error)
	go func() {
		defer close(errChan)
		<-readyChan
		go k8sutils.PrintPodLogs(ctx, output, e.clientSet, e.PodConfig.Namespace, podName, containerName)

		pod, err := e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			errChan <- err
			return
		}

		execOpts := k8sutils.NewExecOptions(e.clientSet, e.restConfig).WithContainer(pod, containerName)

		srcGossFile := path.Join(opts.DockerContextPath, gossFilename)
		remoteGossFile := path.Join("/goss", gossFilename)
		logrus.Debugf("Copying %s to %s/%s:%s", srcGossFile, e.PodConfig.Namespace, pod.Name, remoteGossFile)
		err = k8sutils.CopyToContainer(*execOpts, srcGossFile, remoteGossFile)
		if err != nil {
			errChan <- err
			return
		}

		gossCmd := []string{"/goss/goss", "--gossfile", remoteGossFile, "validate"}
		gossCmd = append(gossCmd, args...)
		logrus.Debugf("Executing command: %v", gossCmd)
		err = k8sutils.Exec(*execOpts.WithWriters(output, os.Stderr), gossCmd)
		if err != nil {
			errChan <- errGossCommandFailed
			return
		}
		errChan <- nil
	}()

	logrus.Debugf("Creating pod: %s/%s", e.PodConfig.Namespace, pod.Name)
	_, err = e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create goss pod: %w", err)
	}
	defer func() {
		logrus.Debugf("Deleting pod %s/%s", e.PodConfig.Namespace, pod.Name)
		_ = e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}()

	select {
	case watchErr := <-watchErrChan:
		if watchErr != nil {
			return fmt.Errorf("error watching goss pod: %w", watchErr)
		}
	case err = <-errChan:
		if err != nil {
			return fmt.Errorf("error running goss tests: %w", err)
		}
	}
	return nil
}
