package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// WaitPodReady waits for a pod to be in running state.
// The function is non-blocking, it returns 2 channels that will be used as event dispatchers:
// - When the pod reaches the running state, an empty struct is sent to readyChan.
// - When the pod reached completion, nil is sent to errChan on success, or an error if the pod failed.
// - If the 1-hour timeout is reached, an error is sent to errChan.
// - If the passed context is cancelled or timeouts, an error is sent to errChan.
func WaitPodReady(ctx context.Context, watcher watch.Interface) (readyChan chan struct{}, errChan chan error) {
	readyChan = make(chan struct{})
	errChan = make(chan error)
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

				logrus.Debugf("Pod %s/%s %s, status %s", pod.ObjectMeta.Namespace,
					pod.ObjectMeta.Name, event.Type, pod.Status.Phase)

				if event.Type == watch.Deleted {
					logrus.Errorf("Pod %s/%s was deleted", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					errChan <- fmt.Errorf("pod %s was deleted", pod.ObjectMeta.Name)
					return
				}

				switch pod.Status.Phase { //nolint: exhaustive
				case corev1.PodRunning:
					if running {
						break
					}
					running = true
					logrus.Infof("Pod %s/%s is running, ready to proceed", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					readyChan <- struct{}{}
				case corev1.PodSucceeded:
					logrus.Infof("Pod %s/%s succeeded", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					errChan <- nil
					return
				case corev1.PodFailed:
					logrus.Infof("Pod %s/%s failed", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					errChan <- fmt.Errorf("pod %s terminated (failed)", pod.Name)
					return
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

// PrintPodLogs watches the logs a container in a pod and writes them to the giver io.Writer.
// The function is blocking, and will continue to print logs until the log stream is no longer readable,
// most likely because the container exited.
func PrintPodLogs(ctx context.Context, out io.Writer, k8s kubernetes.Interface, namespace, pod, container string) {
	req := k8s.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
	})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		logrus.Errorf("Failed to stream logs for pod %s: %v", pod, err)
		return
	}
	defer podLogs.Close()
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
			logrus.Errorf("Error reading logs buffer of pod %s: %v", pod, err)
			return
		}
		if _, err := out.Write(buf[:numBytes]); err != nil {
			logrus.Errorf("Error writing log to output: %v", err)
			return
		}
	}
}

// UniquePodName generates a unique pod name with random characters.
// An identifier string passed as argument will be included in the generated pod name.
func UniquePodName(identifier string) func() string {
	return func() string {
		identifier = strings.ReplaceAll(identifier, ":", "-")
		identifier = strings.ReplaceAll(identifier, "/", "-")
		base := identifier
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
func MergeObjectWithYaml(obj interface{}, yamlOverride string) error {
	if yamlOverride == "" {
		return nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(yamlOverride), 1024)
	if err := decoder.Decode(&obj); err != nil {
		return fmt.Errorf("invalid yaml override for type %T: %w", obj, err)
	}

	return nil
}
