package common

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// FetchResource fetches the K8s resource
func FetchResource(kubeClient kubernetes.Interface, namespace string, trigger *v1alpha1.Trigger) (*unstructured.Unstructured, error) {
	if trigger.Template != nil {
		creds, err := store.GetCredentials(kubeClient, namespace, trigger.Template.Source)
		if err != nil {
			return nil, err
		}
		reader, err := store.GetArtifactReader(trigger.Template.Source, creds, kubeClient)
		if err != nil {
			return nil, err
		}
		uObj, err := store.FetchArtifact(reader, trigger.Template.GroupVersionResource)
		if err != nil {
			return nil, err
		}
		return uObj, nil
	}
	return nil, nil
}
