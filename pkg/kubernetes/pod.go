package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/radiofrance/dib/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// ErrPodDisrupted is wrapped by the errors of MonitorPod when the pod was stopped by the cluster rather than
// by its own workload: eviction, preemption, node shutdown, or deletion by someone else.
var ErrPodDisrupted = errors.New("pod disrupted")

// MonitorPod waits for a pod to be in running state.
// The function is non-blocking, it returns 2 channels that will be used as event dispatchers:
// - When the pod reaches the running state, an empty struct is sent to readyChan.
// - When the pod reached completion, nil is sent to errChan on success, or an error if the pod failed.
// - If the pod failed with a DisruptionTarget condition or was deleted, the error wraps ErrPodDisrupted.
// - If the 1-hour timeout is reached, an error is sent to errChan.
// - If the passed context is cancelled or timeouts, an error is sent to errChan.
func MonitorPod(ctx context.Context, watcher watch.Interface) (chan struct{}, chan error) {
	readyChan := make(chan struct{})
	errChan := make(chan error)
	running := false

	go func() {
		defer close(errChan)
		defer close(readyChan)

		for {
			select {
			case event, chanOk := <-watcher.ResultChan():
				if !chanOk {
					return
				}

				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					// The Object of the event is not a Pod, we can ignore it.
					// Maybe the watcher is watching different types of resources, or the pod we are watching was
					// deleted before the watcher was stopped.
					// In both cases we don't care: we just want updates on the pod status.
					break
				}

				logger.Debugf("Pod %s/%s %s, status %s", pod.Namespace,
					pod.Name, event.Type, pod.Status.Phase)

				if event.Type == watch.Deleted {
					logger.Errorf("Pod %s/%s was deleted", pod.Namespace, pod.Name)

					// The pod is deleted by its creator only after MonitorPod returned: a deletion seen here
					// comes from someone else.
					errChan <- fmt.Errorf("pod %s was deleted: %w", pod.Name, ErrPodDisrupted)

					return
				}

				switch pod.Status.Phase {
				case corev1.PodRunning:
					if running {
						break
					}

					running = true

					logger.Infof("Pod %s/%s is running, ready to proceed", pod.Namespace, pod.Name)

					readyChan <- struct{}{}
				case corev1.PodSucceeded:
					logger.Infof("Pod %s/%s succeeded", pod.Namespace, pod.Name)

					errChan <- nil

					return
				case corev1.PodFailed:
					logger.Errorf("Pod %s/%s failed status: %+v", pod.Namespace, pod.Name, pod.Status)

					if reason, disrupted := disruptionReason(pod); disrupted {
						errChan <- fmt.Errorf("pod %s terminated (failed): %w (%s)", pod.Name, ErrPodDisrupted, reason)

						return
					}

					errChan <- fmt.Errorf("pod %s terminated (failed)", pod.Name)

					return
				case corev1.PodPending, corev1.PodUnknown:
				}
			case <-time.After(1 * time.Hour):
				errChan <- fmt.Errorf("timeout waiting for pod to run to completion")
				return
			case <-ctx.Done():
				errChan <- fmt.Errorf("stop wating for pod: %w", ctx.Err())
				return
			}
		}
	}()

	return readyChan, errChan
}

// disruptionReason tells whether a failed pod was stopped by the cluster rather than by its own workload, and why:
// the reason of its DisruptionTarget condition when it is true, or "deleted" when the pod is being deleted (since
// Kubernetes 1.27 the kubelet moves a deleted pod to the Failed phase before it is removed). The pod is deleted by its
// creator only after MonitorPod returned, so a deletion seen here comes from someone else.
func disruptionReason(pod *corev1.Pod) (string, bool) {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.DisruptionTarget && condition.Status == corev1.ConditionTrue {
			return condition.Reason, true
		}
	}

	if pod.DeletionTimestamp != nil {
		return "deleted", true
	}

	return "", false
}

// PrintPodLogs watches the logs a container in a pod and writes them to the giver io.Writer.
// The function is blocking, and will continue to print logs until the log stream is no longer readable,
// most likely because the container exited.
func PrintPodLogs(ctx context.Context, out io.Writer, k8s kubernetes.Interface,
	namespace, pod, container string,
) {
	req := k8s.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		logger.Errorf("Failed to stream logs for pod %s: %v", pod, err)
		return
	}

	defer func() {
		_ = podLogs.Close()
	}()

	for {
		buf := make([]byte, 2000)

		numBytes, err := podLogs.Read(buf)
		if errors.Is(err, io.EOF) {
			return
		}

		if numBytes == 0 {
			continue
		}

		if err != nil {
			logger.Errorf("Error reading logs buffer of pod %s: %v", pod, err)
			return
		}

		_, err = out.Write(buf[:numBytes])
		if err != nil {
			logger.Errorf("Error writing log to output: %v", err)
			return
		}
	}
}

// UniquePodName generates a unique pod name with random characters.
// An identifier string passed as an argument will be included in the generated pod name.
func UniquePodName(identifier string) func() string {
	return func() string {
		identifier = strings.ReplaceAll(identifier, ":", "-")
		identifier = strings.ReplaceAll(identifier, "/", "-")
		identifier = strings.ReplaceAll(identifier, "_", "-")
		base := identifier
		maxNameLength, randomLength := 63, 8

		maxGeneratedNameLength := maxNameLength - randomLength - 1
		if len(base) > maxGeneratedNameLength {
			base = base[:maxGeneratedNameLength]
		}

		return strings.ToLower(fmt.Sprintf("%s-%s", base, rand.String(randomLength)))
	}
}

// UniquePodNameWithImage generates a unique pod name that includes both an identifier and an image name.
// The image name is sanitized to comply with Kubernetes naming conventions.
func UniquePodNameWithImage(identifier string, imageName string) func() string {
	return func() string {
		// Sanitize the identifier
		identifier = strings.ReplaceAll(identifier, ":", "-")
		identifier = strings.ReplaceAll(identifier, "/", "-")
		identifier = strings.ReplaceAll(identifier, "_", "-")

		// Sanitize the image name
		imageName = strings.ReplaceAll(imageName, ":", "-")
		imageName = strings.ReplaceAll(imageName, "/", "-")
		imageName = strings.ReplaceAll(imageName, "_", "-")

		// Create the base name with both identifier and image name
		base := fmt.Sprintf("%s-%s", identifier, imageName)

		// Ensure the name respects Kubernetes naming conventions
		maxNameLength, randomLength := 63, 8

		maxGeneratedNameLength := maxNameLength - randomLength - 1
		if len(base) > maxGeneratedNameLength {
			base = base[:maxGeneratedNameLength]
		}

		return strings.ToLower(fmt.Sprintf("%s-%s", base, rand.String(randomLength)))
	}
}

// MergeObjectWithYaml unmarshalls the YAML from the yamlOverride argument into the provided object.
// The `obj` argument typically is a pointer to a kubernetes type (with `json` tags).
// Existing values inside the `obj` will be erased if the YAML explicitly overrides it.
// All values within the object that are not explicitly overridden will not be modified.
func MergeObjectWithYaml(obj any, yamlOverride string) error {
	if yamlOverride == "" {
		return nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(yamlOverride), 1024)

	err := decoder.Decode(&obj)
	if err != nil {
		return fmt.Errorf("invalid yaml override for type %T: %w", obj, err)
	}

	return nil
}
