package gateway

import (
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

var (
	configmapName = "gateway-controller-configmap-test"
)

func getController() *GatewayController {
	return &GatewayController{
		ConfigMap:     configmapName,
		ConfigMapNS:   "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}
}

func TestWatchControllerConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc := getController()
	_, err := gc.watchControllerConfigMap(ctx)
	assert.Nil(t, err)
}

func TestNewControllerConfigMapWatch(t *testing.T) {
	gc := getController()
	watcher := gc.newControllerConfigMapWatch()
	assert.NotNil(t, watcher)
}

func TestGatewayController_ResyncConfig(t *testing.T) {
	gc := getController()
	cmObj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gc.ConfigMapNS,
			Name:      gc.ConfigMap,
		},
		Data: map[string]string{
			common.GatewayControllerConfigMapKey: `instanceID: argo-events`,
		},
	}

	cm, err := gc.kubeClientset.CoreV1().ConfigMaps(gc.ConfigMapNS).Create(cmObj)
	assert.Nil(t, err)

	err = gc.ResyncConfig(gc.ConfigMapNS)
	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, gc.Config)
	assert.NotEqual(t, gc.Config.Namespace, gc.ConfigMapNS)
	assert.Equal(t, gc.Config.Namespace, common.DefaultGatewayControllerNamespace)
}
