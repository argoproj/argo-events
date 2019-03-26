package common

import (
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func SetLabel(obj *unstructured.Unstructured, key, val string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = val
	obj.SetLabels(labels)
}
