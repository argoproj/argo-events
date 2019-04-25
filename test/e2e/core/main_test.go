package core

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gwalpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	snv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	e2ecommon "github.com/argoproj/argo-events/test/e2e/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const NAMESPACE = "argo-events"

func TestGeneralUseCase(t *testing.T) {
	client, err := e2ecommon.NewE2EClient()
	if err != nil {
		t.Fatal(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		t.Fatal(err)
	}
	manifestsDir := filepath.Join(dir, "manifests", "general-use-case")

	convey.Convey("Test the general use case", t, func() {

		convey.Convey("Create event source", func() {
			esBytes, err := ioutil.ReadFile(filepath.Join(manifestsDir, "webhook-gateway-event-source.yaml"))
			if err != nil {
				convey.ShouldPanic(err)
			}
			var cm *corev1.ConfigMap
			if err := yaml.Unmarshal(esBytes, &cm); err != nil {
				convey.ShouldPanic(err)
			}
			if _, err = client.KubeClient.CoreV1().ConfigMaps(NAMESPACE).Create(cm); err != nil {
				convey.ShouldPanic(err)
			}
		})

		convey.Convey("Create a gateway.", func() {
			gwBytes, err := ioutil.ReadFile(filepath.Join(manifestsDir, "webhook-gateway.yaml"))
			if err != nil {
				convey.ShouldPanic(err)
			}
			var gw *gwalpha1.Gateway
			if err := yaml.Unmarshal(gwBytes, &gw); err != nil {
				convey.ShouldPanic(err)
			}
			if _, err = client.GwClient.ArgoprojV1alpha1().Gateways(NAMESPACE).Create(gw); err != nil {
				convey.ShouldPanic(err)
			}
		})

		convey.Convey("Create a sensor.", func() {
			swBytes, err := ioutil.ReadFile(filepath.Join(manifestsDir, "webhook-sensor.yaml"))
			if err != nil {
				convey.ShouldPanic(err)
			}
			var sn *snv1alpha1.Sensor
			if err := yaml.Unmarshal(swBytes, &sn); err != nil {
				convey.ShouldPanic(err)
			}
			if _, err = client.SnClient.ArgoprojV1alpha1().Sensors(NAMESPACE).Create(sn); err != nil {
				convey.ShouldPanic(err)
			}
		})

		convey.Convey("Wait for corresponding resources.", func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			var gwpod, spod *corev1.Pod
			var gwsvc *corev1.Service
			for {
				if gwpod == nil {
					pod, err := client.KubeClient.CoreV1().Pods(NAMESPACE).Get("webhook-gateway", metav1.GetOptions{})
					if err != nil && !apierr.IsNotFound(err) {
						t.Fatal(err)
					}
					_, _ = yaml.Marshal(pod)
					if pod != nil && pod.Status.Phase == corev1.PodRunning {
						gwpod = pod
					}
				}

				if gwsvc == nil {
					svc, err := client.KubeClient.CoreV1().Services(NAMESPACE).Get("webhook-gateway-svc", metav1.GetOptions{})
					if err != nil && !apierr.IsNotFound(err) {
						t.Fatal(err)
					}
					gwsvc = svc
				}
				if spod == nil {
					pod, err := client.KubeClient.CoreV1().Pods(NAMESPACE).Get("webhook-sensor", metav1.GetOptions{})
					if err != nil && !apierr.IsNotFound(err) {
						t.Fatal(err)
					}
					if pod != nil && pod.Status.Phase == corev1.PodRunning {
						spod = pod
					}
				}
				if gwpod != nil && gwsvc != nil && spod != nil {
					break
				}
			}
		})

		convey.Convey("Make a request to the gateway.", func() {
			// Avoid too early access
			time.Sleep(5 * time.Second)

			// Use available port
			l, _ := net.Listen("tcp", ":0")
			port := l.Addr().(*net.TCPAddr).Port
			l.Close()

			// Use port forwarding to access pods in minikube
			stopChan, err := client.ForwardServicePort(NAMESPACE, "webhook-gateway", port, 12000)
			if err != nil {
				t.Fatal(err)
			}
			defer close(stopChan)

			url := fmt.Sprintf("http://localhost:%d/foo", port)
			req, err := http.NewRequest("POST", url, strings.NewReader("e2e"))
			if err != nil {
				t.Fatal(err)
			}

			resp, err := new(http.Client).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if t.Failed() {
				t.FailNow()
			}
		})

		convey.Convey("Check if the sensor trigggered a pod.", func() {
			pod, err := client.KubeClient.CoreV1().Pods(NAMESPACE).Get("webhook-sensor-triggered-pod", metav1.GetOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				t.Error(err)
			}
			if pod != nil && pod.Status.Phase == corev1.PodSucceeded {
				convey.So(pod.Spec.Containers[0].Args[0], convey.ShouldEqual, "e2e")
			}
		})
	})
}
