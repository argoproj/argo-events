package common

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	gwv1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var (
	kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube/config")
)

func init() {
	rand.Seed(time.Now().UnixNano())

	// Add custom schemes
	if err := sv1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := gwv1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

type E2EClient struct {
	dynamic.ClientPool
	discovery.DiscoveryClient
	E2EID    string
	ClientID string
}

func NewE2EClient(e2eID string) (*E2EClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientPool := dynamic.NewDynamicClientPool(config)

	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	clientID := strconv.FormatUint(rand.Uint64(), 16)

	return &E2EClient{
		ClientPool:      clientPool,
		DiscoveryClient: *disco,
		E2EID:           e2eID,
		ClientID:        clientID,
	}, nil
}

func (clpl *E2EClient) CreateTmpNamespace() (string, error) {
	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", clpl.E2EID),
			Labels: map[string]string{
				ArgoEventsE2ETestLabelKey:         clpl.E2EID,
				ArgoEventsE2ETestClientIDLabelKey: clpl.ClientID,
			},
		},
	}
	ns, err := clpl.Create("", namespace)
	if err != nil {
		return "", err
	}

	nsName := ns.GetName()
	binding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: nsName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: nsName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "edit",
		},
	}

	_, err = clpl.Create(nsName, binding)
	if err != nil {
		return "", err
	}

	return nsName, nil
}

func (clpl *E2EClient) DeleteNamespaces() error {
	client, err := clpl.GetClient("", schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})
	if err != nil {
		return err
	}
	labelSelector := labels.Set(map[string]string{ArgoEventsE2ETestClientIDLabelKey: clpl.ClientID}).String()
	return client.DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector})
}

func (clpl *E2EClient) GetClient(namespace string, gvk schema.GroupVersionKind) (dynamic.ResourceInterface, error) {
	client, err := clpl.ClientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return nil, err
	}

	resources, err := clpl.DiscoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, err
	}

	for _, resource := range resources.APIResources {
		if resource.Kind == gvk.Kind {
			return client.Resource(&resource, namespace), nil
		}
	}

	return nil, fmt.Errorf("resource not found: %v", gvk)
}

func (clpl *E2EClient) Create(namespace string, obj runtime.Object) (*unstructured.Unstructured, error) {
	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	unstructuredObj := &unstructured.Unstructured{Object: uObj}

	client, err := clpl.GetClient(namespace, obj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, err
	}

	return client.Create(unstructuredObj)
}
