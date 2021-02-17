package fixtures

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Exec(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	println(cmd.String())
	output, err := runWithTimeout(cmd)
	// Command completed before timeout. Print output and error if it exists.
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
	}
	for _, s := range strings.Split(output, "\n") {
		println(s)
	}
	return output, err
}

func runWithTimeout(cmd *exec.Cmd) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Start()
	if err != nil {
		return "", err
	}
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	timeout := time.After(60 * time.Second)
	select {
	case <-timeout:
		_ = cmd.Process.Kill()
		return buf.String(), fmt.Errorf("timeout")
	case err := <-done:
		return buf.String(), err
	}
}

func podNameByLabelSelectors(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) (string, error) {
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	l, err := client.CoreV1().Pods(Namespace).List(ctx, opts)
	if err != nil {
		return "", err
	}
	return l.Items[0].GetName(), nil
}
