package exec

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/radiofrance/dib/internal/logger"
	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// KubernetesExecutor will run Buildkit in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet kubernetes.Interface
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(clientSet kubernetes.Interface) *KubernetesExecutor {
	return &KubernetesExecutor{
		clientSet: clientSet,
	}
}

// ApplyWithWriters executes a Buildkit build using a Kubernetes Pod.
// Currently, this function is designed to handle only Pod objects.
// It may evolve in the future to support other types of Kubernetes objects.
//
//nolint:lll
func (e KubernetesExecutor) ApplyWithWriters(ctx context.Context, stdout, stderr io.Writer, k8sObject runtime.Object, containerNames string) error {
	pod, ok := k8sObject.(*corev1.Pod)
	if !ok {
		return errors.New("only pod object is supported")
	}

	watcher, err := e.clientSet.CoreV1().Pods(pod.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", pod.Name),
		Watch:         true,
	})
	if err != nil {
		return fmt.Errorf("failed to watch pod: %w", err)
	}
	defer watcher.Stop()

	readyChan, errChan := k8sutils.MonitorPod(ctx, watcher)
	go func() {
		<-readyChan
		// Kubernetes logs API returns a single combined stdout+stderr stream.
		// Avoid duplicating logs when stdout and stderr point to the same writer.
		var out io.Writer
		switch {
		case stdout == nil && stderr == nil:
			out = io.Discard
		case stderr == nil || stderr == stdout:
			out = stdout
		case stdout == nil:
			out = stderr
		default:
			out = io.MultiWriter(stdout, stderr)
		}
		k8sutils.PrintPodLogs(ctx, out, e.clientSet, pod.Namespace, pod.Name, containerNames)
	}()

	_, err = e.clientSet.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Buildkit pod: %w", err)
	}

	defer func() {
		err := e.clientSet.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Warnf("Failed to delete Buildkit pod %s, ignoring: %v", pod.Name, err)
		}
	}()

	err = <-errChan
	if err != nil {
		return fmt.Errorf("error watching Buildkit pod: %w", err)
	}

	return nil
}
