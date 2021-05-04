package eventsource

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
	testEventSource.Spec.Template = &v1alpha1.Template{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test",
			},
		},
		PriorityClassName: "test-class",
	}
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
		hasTmpVolume := false
		cmRefs, secretRefs := 0, 0
		for _, vol := range volumes {
			if vol.Name == "auth-volume" {
				hasAuthVolume = true
			}
			if vol.Name == "tmp" {
				hasTmpVolume = true
			}
			if strings.Contains(vol.Name, testEventSource.Spec.HDFS["test"].KrbCCacheSecret.Name) {
				secretRefs++
			}
			if strings.Contains(vol.Name, testEventSource.Spec.HDFS["test"].KrbConfigConfigMap.Name) {
				cmRefs++
			}
		}
		assert.True(t, hasAuthVolume)
		assert.True(t, hasTmpVolume)
		assert.True(t, len(deployment.Spec.Template.Spec.ImagePullSecrets) > 0)
		assert.True(t, cmRefs > 0)
		assert.True(t, secretRefs > 0)
		assert.Equal(t, deployment.Spec.Template.Spec.PriorityClassName, "test-class")
	})
}

func TestResourceReconcile(t *testing.T) {
	testEventSource := fakeEmptyEventSource()
	testEventSource.Spec.HDFS = fakeHDFSEventSourceMap("test")
	t.Run("test resource reconcile without eventbus", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
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
		cl := fake.NewClientBuilder().Build()
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
