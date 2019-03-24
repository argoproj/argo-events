package core

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"github.com/argoproj/argo-events/common"
	e2ecommon "github.com/argoproj/argo-events/test/e2e/common"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	client       *e2ecommon.E2EClient
	e2eID        string
	tmpNamespace string
)

func setup() error {
	e2eID = e2ecommon.GetE2EID()
	cli, err := e2ecommon.NewE2EClient(e2eID)
	if err != nil {
		return err
	}
	client = cli
	namespace, err := client.CreateTmpNamespace()
	if err != nil {
		return err
	}
	tmpNamespace = namespace
	return nil
}

func teardown() {
	if !e2ecommon.KeepNamespace() {
		if client != nil {
			err := client.DeleteNamespaces()
			if err != nil {
				fmt.Printf("%+v\n", err)
			}
		}
	}
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		teardown()
		panic(err)
	}
	ret := m.Run()
	teardown()
	os.Exit(ret)
}

func TestGeneralUseCase(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		t.Fatal(err)
	}
	manifestsDir := filepath.Join(dir, "manifests")

	convey.Convey("Test the general use case", t, func() {
		convey.Convey("Create a gateway.", func() {
			_, err := client.CreateResourceFromYaml(tmpNamespace, filepath.Join(manifestsDir, "webhook-gateway.yaml"), func(obj *unstructured.Unstructured) error {
				e2ecommon.SetLabel(obj, common.LabelKeyGatewayControllerInstanceID, e2eID)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.CreateResourceFromYaml(tmpNamespace, filepath.Join(manifestsDir, "webhook-gateway-configmap.yaml"), nil)
			if err != nil {
				t.Fatal(err)
			}
		})

		convey.Convey("Create a sensor.", func() {
			_, err := client.CreateResourceFromYaml(tmpNamespace, filepath.Join(manifestsDir, "webhook-sensor.yaml"), func(obj *unstructured.Unstructured) error {
				e2ecommon.SetLabel(obj, common.LabelKeySensorControllerInstanceID, e2eID)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})

		convey.Convey("Wait for corresponding resources.", func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			var gwpod, spod *corev1.Pod
			var gwsvc *corev1.Service
		L:
			for {
				select {
				case _ = <-ticker.C:
					if gwpod == nil {
						pod, err := client.GetPod(tmpNamespace, "webhook-gateway")
						if err != nil && !apierr.IsNotFound(err) {
							t.Error(err)
							break L
						}
						if pod != nil && pod.Status.Phase == corev1.PodRunning {
							gwpod = pod
						}
					}
					if gwsvc == nil {
						svc, err := client.GetService(tmpNamespace, "webhook-gateway-svc")
						if err != nil && !apierr.IsNotFound(err) {
							t.Error(err)
							break L
						}
						gwsvc = svc
					}
					if spod == nil {
						pod, err := client.GetPod(tmpNamespace, "webhook-sensor")
						if err != nil && !apierr.IsNotFound(err) {
							t.Error(err)
							break L
						}
						if pod != nil && pod.Status.Phase == corev1.PodRunning {
							spod = pod
						}
					}
					if gwpod != nil && gwsvc != nil && spod != nil {
						break L
					}
				case <-time.After(10 * time.Second):
					t.Error("timed out gateway and sensor startup")
					break L
				}
			}
			if t.Failed() {
				t.FailNow()
			}
		})

		convey.Convey("Make a request to the gateway.", func() {
			// Avoid too early access
			time.Sleep(1 * time.Second)

			// Use available port
			l, _ := net.Listen("tcp", ":0")
			port := l.Addr().(*net.TCPAddr).Port
			l.Close()

			// Use port forwarding to access pods in minikube
			stopChan, err := client.ForwardServicePort(tmpNamespace, "webhook-gateway", port, 12000)
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
			ticker2 := time.NewTicker(time.Second)
			defer ticker2.Stop()
		L:
			for {
				select {
				case _ = <-ticker2.C:
					pod, err := client.GetPod(tmpNamespace, "webhook-sensor-triggered-pod")
					if err != nil && !apierr.IsNotFound(err) {
						t.Error(err)
						break L
					}
					if pod != nil && pod.Status.Phase == corev1.PodSucceeded {
						convey.So(pod.Spec.Containers[0].Args[0], convey.ShouldEqual, "e2e")
						break L
					}
				case <-time.After(10 * time.Second):
					t.Error("timed out gateway and sensor startup")
					break L
				}
			}
		})
	})
}
