package fixtures

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
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

func PortForward(config *rest.Config, namespace, podName string, localPort, remotePort int, stopCh <-chan struct{}) error {
	readyCh := make(chan struct{})

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		namespace, podName)
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}
	var scheme string
	var host string
	if strings.HasPrefix(config.Host, "https://") {
		scheme = "https"
		host = strings.TrimPrefix(config.Host, "https://")
	} else {
		scheme = "http"
		host = strings.TrimPrefix(config.Host, "http://")
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: scheme, Path: path, Host: host})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopCh, readyCh, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			panic(err)
		}
	}()

	<-readyCh

	return nil
}
