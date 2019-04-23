package common

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gwv1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func init() {
	// Add custom schemes
	if err := sv1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	if err := gwv1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

type E2EClient struct {
	Config     *restclient.Config
	KubeClient kubernetes.Interface
	GwClient   gwclient.Interface
	SnClient   sensorclient.Interface
	E2EID      string
	ClientID   string
}

func NewE2EClient() (*E2EClient, error) {
	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube/config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	gwClient, err := gwclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	sensorClient, err := sensorclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	myrand := rand.New(rand.NewSource(time.Now().UnixNano()))
	clientID := strconv.FormatUint(myrand.Uint64(), 16)

	return &E2EClient{
		Config:     config,
		KubeClient: kubeClient,
		GwClient:   gwClient,
		SnClient:   sensorClient,
		ClientID:   clientID,
	}, nil
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
