package kaniko

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func printPodLog(ctx context.Context, ready chan struct{}, k8s kubernetes.Interface, ns string, podName string) {
	<-ready
	req := k8s.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		logrus.Errorf("Failed to stream logs for pod %s: %v", podName, err)
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
			logrus.Errorf("Error reading logs buffer of pod %s: %v", podName, err)
			return
		}
		message := string(buf[:numBytes])
		fmt.Print(message) // nolint: forbidigo
	}
}
