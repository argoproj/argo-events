package store

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestConfigmapReader_Read(t *testing.T) {
	kubeClientset := fake.NewSimpleClientset()
	key := "wf"
	cmArtifact := &v1alpha1.ConfigmapArtifact{
		Name:      "wf-configmap",
		Namespace: "argo-events",
		Key:       key,
	}

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmArtifact.Name,
			Namespace: cmArtifact.Namespace,
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
		cm, err := kubeClientset.CoreV1().ConfigMaps(cmArtifact.Namespace).Create(configmap)
		convey.So(err, convey.ShouldBeNil)
		convey.So(cm, convey.ShouldNotBeNil)

		convey.Convey("Make sure new configmap reader is not nil", func() {
			cmReader, err := NewConfigMapReader(kubeClientset, cmArtifact)
			convey.So(err, convey.ShouldBeNil)
			convey.So(cmReader, convey.ShouldNotBeNil)

			convey.Convey("Create a workflow from configmap minio", func() {
				resourceBody, err := cmReader.Read()
				convey.So(err, convey.ShouldBeNil)

				var wf *wfv1.Workflow
				err = yaml.Unmarshal(resourceBody, &wf)
				convey.So(err, convey.ShouldBeNil)
				convey.So(wf.Name, convey.ShouldEqual, "hello-world")
			})
		})
	})
}
