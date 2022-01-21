package kaniko

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	secondsToSleepEachTime = 1
	maxRetryToGetJobOrPod  = 90
)

const (
	Unknown jobState = iota
	Active
	Succeeded
	Failed
)

type jobState int

func (j jobState) String() string {
	return [...]string{"Active", "Succeeded", "Failed", "Unknown"}[j]
}

var errNotFound = errors.New("job or pod not found")

func getJob(ctx context.Context, k8s kubernetes.Interface, namespace string, jobName string) (*batchv1.Job, error) {
	job, err := k8s.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get job %s: %w", jobName, err)
	}

	return job, nil
}

func getPod(ctx context.Context, k8s kubernetes.Interface, namespace string, jobName string) (*corev1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		Limit:         1,
	}

	podList, err := k8s.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("can't get pod name for job %s: %w", jobName, err)
	}

	pods := podList.Items

	if len(pods) == 0 {
		return nil, errNotFound
	}

	return &pods[0], nil
}

func printPodLog(
	ctx context.Context,
	k8s kubernetes.Interface,
	logBuf *bytes.Buffer,
	ns string,
	podName string,
) (*bytes.Buffer, error) {
	var logsToDisplayBytes []byte

	podLogsReq := k8s.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{})
	podLogsStream, err := podLogsReq.Stream(ctx)
	if err != nil {
		return logBuf, fmt.Errorf("error streaming logs from pod: %w", err)
	}

	defer func() {
		if err := podLogsStream.Close(); err != nil {
			logrus.Errorf("can't close current log stream for pod %s, err is %v", podName, err)
		}
	}()

	streamLogsBuf := new(bytes.Buffer)
	streamLogsBufLen, streamLogsBufErr := io.Copy(streamLogsBuf, podLogsStream)
	if streamLogsBufErr != nil {
		logrus.Errorf(
			"can't get Pod logs stream copied to buffer for Kaniko Pod %s: %v",
			podName,
			streamLogsBufErr,
		)
	}

	totalPodLogsBufLen := logBuf.Len()

	if int(streamLogsBufLen) > totalPodLogsBufLen {
		logBuf = streamLogsBuf
		streamLogsBytes := streamLogsBuf.Bytes()
		logsToDisplayBytes = make([]byte, int(streamLogsBufLen)-totalPodLogsBufLen+1)
		for i := totalPodLogsBufLen; i < int(streamLogsBufLen); i++ {
			logsToDisplayBytes[i-totalPodLogsBufLen] = streamLogsBytes[i]
		}
		for _, line := range strings.Split(string(logsToDisplayBytes), "\n") {
			if len(line) > 1 { // We sometimes have strange lines of len 1, with some special characters, we skip it
				logrus.Info(line)
			}
		}
	}

	return logBuf, nil
}

func isJobAlive(job *batchv1.Job) (bool, jobState) {
	switch {
	case job.Status.Active == 1:
		return true, Active
	case job.Status.Succeeded == 1:
		return false, Succeeded
	case job.Status.Failed == 1:
		return false, Failed
	default:
		return false, Unknown
	}
}

func waitForJobToBeReady(
	ctx context.Context,
	k8s kubernetes.Interface,
	namespace string,
	jobName string,
) (*batchv1.Job, error) {
	count := 0
	for {
		job, err := getJob(ctx, k8s, namespace, jobName)
		if job != nil {
			return job, nil
		}
		if err != nil && !errors.Is(err, errNotFound) {
			return nil, err
		}
		if count > maxRetryToGetJobOrPod {
			return nil, fmt.Errorf("the job %s doesn't exist", jobName)
		}
		count++
		logrus.Infof("Waiting for Job %s to be Ready, retrying", jobName)
		time.Sleep(secondsToSleepEachTime * time.Second)
	}
}

func waitForPodToBeReady(
	ctx context.Context,
	k8s kubernetes.Interface,
	namespace string,
	jobName string,
) (*corev1.Pod, error) {
	count := 0
	for {
		count++

		time.Sleep(secondsToSleepEachTime * time.Second)

		pod, err := getPod(ctx, k8s, namespace, jobName)
		if pod != nil {
			return pod, nil
		}
		logrus.Infof("Waited for %d seconds for Pod created by Job %s to be ready", count, jobName)
		if err != nil && !errors.Is(err, errNotFound) {
			return nil, err
		}
		if count > maxRetryToGetJobOrPod {
			return nil, fmt.Errorf("no pod found related to job %s, maximum retry count reached", jobName)
		}
	}
}

func printLogAndGetLogAndStatus(
	ctx context.Context,
	k8s kubernetes.Interface,
	logBuf *bytes.Buffer,
	namespace string,
	jobName string,
) (bool, jobState, error) {
	var updatedJobState jobState
	var isActive, wasActiveOnce bool
	countWaitState := 0
	countUnknownState := 0
	for {
		job, err := waitForJobToBeReady(ctx, k8s, namespace, jobName)
		if err != nil {
			return false, Unknown, err
		}
		isActive, updatedJobState = isJobAlive(job)
		if isActive {
			wasActiveOnce = true
			pod, err := waitForPodToBeReady(ctx, k8s, namespace, jobName)
			if err != nil {
				return false, Unknown, err
			}
			logBuf, err = printPodLog(ctx, k8s, logBuf, namespace, pod.GetName())
			if err != nil {
				if countWaitState > maxRetryToGetJobOrPod {
					return false, Unknown, fmt.Errorf(
						"pod %s is stuck in waiting state for the maximum allowed time of %d second(s): %w",
						pod.GetName(),
						maxRetryToGetJobOrPod,
						err,
					)
				}
				countWaitState++
				logrus.Info(err)
				time.Sleep(secondsToSleepEachTime * time.Second)
			}
		} else {
			// It will only retry in case of the Job start in 'Unknown' state, when he never was active at all
			if updatedJobState == Failed || wasActiveOnce || countUnknownState > maxRetryToGetJobOrPod {
				return wasActiveOnce, updatedJobState, nil
			}
			countUnknownState++
			logrus.Infof("Job %s is in '%s' state, retrying...", jobName, Unknown.String())
			time.Sleep(secondsToSleepEachTime * time.Second)
		}
	}
}

// watchJob waits for the job to be completed and prints the logs from the pod.
// Once the job completes, it is deleted.
func watchJob(ctx context.Context, k8s kubernetes.Interface, namespace string, jobName string) error {
	defer func() {
		logrus.Infof("Deleting Job %s", jobName)
		if err := deleteJob(ctx, k8s, namespace, jobName); err != nil {
			logrus.Errorf("Failed to delete Job: %v, skipping", err)
		}
	}()

	logrus.Infof("Fetching logs for job %s", jobName)
	if err := logPod(ctx, k8s, namespace, jobName); err != nil {
		return err
	}

	return nil
}

func logPod(ctx context.Context, k8s kubernetes.Interface, ns string, jobName string) error {
	totalPodLogsBuf := new(bytes.Buffer)
	wasActiveOnce, jobState, err := printLogAndGetLogAndStatus(ctx, k8s, totalPodLogsBuf, ns, jobName)
	if err != nil {
		return err
	}
	// Here it takes as assumption when the job is done and came from 'Active' state to 'Unknown', it's valid
	if jobState == Failed || !wasActiveOnce {
		return fmt.Errorf("kaniko pod is in '%s' state", jobState.String())
	}

	logrus.Infof("Kaniko Job %s is %s", jobName, jobState.String())

	return nil
}

func deleteJob(ctx context.Context, k8s kubernetes.Interface, ns string, jobName string) error {
	deletePolicy := metav1.DeletePropagationBackground
	if err := k8s.BatchV1().Jobs(ns).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}
