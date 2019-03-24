package common

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

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
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

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
	Config *restclient.Config
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
		Config:          config,
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
			Name:     "argo-events-role",
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
	// DeleteCollection is not supported for namespaces,
	// so list namespaces and delete them one by one.
	labelSelector := labels.Set(map[string]string{ArgoEventsE2ETestClientIDLabelKey: clpl.ClientID}).String()
	obj, err := client.List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	for _, ns := range (unstructured.UnstructuredList{Object: uObj}).Items {
		err := client.Delete(ns.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
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

func (clpl *E2EClient) CreateResourceFromYaml(namespace, path string, modFunc func(*unstructured.Unstructured) error) (*unstructured.Unstructured, error) {
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

	unstructuredObj := &unstructured.Unstructured{Object: uObj}

	if modFunc != nil {
		err = modFunc(unstructuredObj)
		if err != nil {
			return nil, err
		}
	}

	return clpl.Create(namespace, unstructuredObj)
}

func (clpl *E2EClient) Get(namespace string, gvk schema.GroupVersionKind, name string) (*unstructured.Unstructured, error) {
	client, err := clpl.GetClient(namespace, gvk)
	if err != nil {
		return nil, err
	}

	return client.Get(name, metav1.GetOptions{})
}

func (clpl *E2EClient) GetPod(namespace string, name string) (*corev1.Pod, error) {
	client, err := clpl.GetClient(namespace, schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	if err != nil {
		return nil, err
	}

	uObj, err := client.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	newObj := &corev1.Pod{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(uObj.Object, newObj)
	if err != nil {
		return nil, err
	}
	return newObj, nil
}

func (clpl *E2EClient) GetService(namespace string, name string) (*corev1.Service, error) {
	client, err := clpl.GetClient(namespace, schema.GroupVersionKind{Version: "v1", Kind: "Service"})
	if err != nil {
		return nil, err
	}

	uObj, err := client.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	newObj := &corev1.Service{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(uObj.Object, newObj)
	if err != nil {
		return nil, err
	}
	return newObj, nil
}

func (clpl *E2EClient) ForwardServicePort(tmpNamespace, podName string, localPort, targetPort int) (chan struct{}, error) {
	// Implementation ref: https://github.com/kubernetes/client-go/issues/51#issuecomment-436200428
	roundTripper, upgrader, err := spdy.RoundTripperFor(clpl.Config)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", tmpNamespace, podName)
	hostIP := strings.TrimLeft(clpl.Config.Host, "https://")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)

	portDesc := fmt.Sprintf("%d:%d", localPort, targetPort)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, []string{portDesc}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		err = forwarder.ForwardPorts()
		if err != nil {
			fmt.Printf("%+v\n", err)
		}
	}()

	err = nil
L:
	for {
		select {
		case <-time.After(10 * time.Second):
			err = errors.New("timed out port forwarding")
			break L
		case <-readyChan:
			break L
		default:
		}
	}

	return stopChan, err
}
