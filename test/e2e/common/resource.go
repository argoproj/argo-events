package common

import (
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	gwv1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func GetE2EID() string {
	e2eID := os.Getenv("E2E_ID")
	if e2eID == "" {
		e2eID = "argo-events-e2e"
	}
	return e2eID
}

func KeepNamespace() bool {
	return os.Getenv("KEEP_NAMESPACE") != ""
}

func ReadResource(path string) (*unstructured.Unstructured, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: uObj}, nil
}

func ToGateway(obj *unstructured.Unstructured) (*gwv1.Gateway, error) {
	var newObj *gwv1.Gateway
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, newObj)
	if err != nil {
		return nil, err
	}
	return newObj, nil
}

func ToSensor(obj *unstructured.Unstructured) (*sv1.Sensor, error) {
	var newObj *sv1.Sensor
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, newObj)
	if err != nil {
		return nil, err
	}
	return newObj, nil
}

func SetLabel(obj *unstructured.Unstructured, key, val string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = val
	obj.SetLabels(labels)
}
