package eventsource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common/logging"
)

const (
	testImage = "test-image"
)

var (
	testLabels = map[string]string{"controller": "test-controller"}
)

func Test_BuildDeployment(t *testing.T) {
	testEventSource := fakeEmptyEventSource()
	testEventSource.Spec.HDFS = fakeHDFSEventSourceMap("test")
	t.Run("test build HDFS", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:       testImage,
			EventSource: testEventSource,
			Labels:      testLabels,
		}
		deployment, err := buildDeployment(args, fakeEventBus)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		volumes := deployment.Spec.Template.Spec.Volumes
		assert.True(t, len(volumes) > 0)
		hasAuthVolume := false
		for _, vol := range volumes {
			if vol.Name == "auth-volume" {
				hasAuthVolume = true
				break
			}
		}
		assert.True(t, hasAuthVolume)
		envFroms := deployment.Spec.Template.Spec.Containers[0].EnvFrom
		assert.True(t, len(envFroms) > 0)
		cmRefs, secretRefs := 0, 0
		for _, ef := range envFroms {
			if ef.ConfigMapRef != nil {
				cmRefs++
			} else if ef.SecretRef != nil {
				secretRefs++
			}
		}
		assert.True(t, cmRefs > 0)
		assert.True(t, secretRefs > 0)
	})
}

func TestResourceReconcile(t *testing.T) {
	testEventSource := fakeEmptyEventSource()
	testEventSource.Spec.HDFS = fakeHDFSEventSourceMap("test")
	t.Run("test resource reconcile without eventbus", func(t *testing.T) {
		cl := fake.NewFakeClient(testEventSource)
		args := &AdaptorArgs{
			Image:       testImage,
			EventSource: testEventSource,
			Labels:      testLabels,
		}
		err := Reconcile(cl, args, logging.NewArgoEventsLogger())
		assert.Error(t, err)
		assert.False(t, testEventSource.Status.IsReady())
	})

	t.Run("test resource reconcile with eventbus", func(t *testing.T) {
		ctx := context.TODO()
		cl := fake.NewFakeClient(testEventSource)
		testBus := fakeEventBus.DeepCopy()
		testBus.Status.MarkDeployed("test", "test")
		testBus.Status.MarkConfigured()
		err := cl.Create(ctx, testBus)
		assert.Nil(t, err)
		args := &AdaptorArgs{
			Image:       testImage,
			EventSource: testEventSource,
			Labels:      testLabels,
		}
		err = Reconcile(cl, args, logging.NewArgoEventsLogger())
		assert.Nil(t, err)
		assert.True(t, testEventSource.Status.IsReady())

		deployList := &appv1.DeploymentList{}
		err = cl.List(ctx, deployList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(deployList.Items))

		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(svcList.Items))
	})
}
