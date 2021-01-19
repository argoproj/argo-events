package artifacts

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConfigmapReader_Read(t *testing.T) {
	kubeClientset := fake.NewSimpleClientset()
	key := "wf"

	cmArtifact := &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "fake-cm",
		},
		Key: key,
	}
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-cm",
			Namespace: "fake-ns",
		},
		Data: map[string]string{
			key: `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
       args:
       - "hello world"	
       command:
       - cowsay
       image: "docker/whalesay:latest"`,
		},
	}

	convey.Convey("Given a configmap", t, func() {
		cm, err := kubeClientset.CoreV1().ConfigMaps("fake-ns").Create(context.TODO(), configmap, metav1.CreateOptions{})
		convey.So(err, convey.ShouldBeNil)
		convey.So(cm, convey.ShouldNotBeNil)

		convey.Convey("Make sure new configmap reader is not nil", func() {
			cmReader, err := NewConfigMapReader(cmArtifact)
			convey.So(err, convey.ShouldBeNil)
			convey.So(cmReader, convey.ShouldNotBeNil)
		})
	})
}
