package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

var (
	testContainer = &Container{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"cpu": resource.MustParse("100m"),
			},
		},
	}
)

func Test_ApplyToContainer(t *testing.T) {
	c := &corev1.Container{}
	testContainer.ApplyToContainer(c)
	assert.Equal(t, "", string(c.ImagePullPolicy))
	assert.Equal(t, testContainer.Resources, c.Resources)
	c.Resources.Limits = corev1.ResourceList{
		"cpu": resource.MustParse("200m"),
	}
	testContainer.ApplyToContainer(c)
	assert.Equal(t, resource.MustParse("200m"), *c.Resources.Limits.Cpu())
	c.Resources.Limits = corev1.ResourceList{
		"memory": resource.MustParse("32Mi"),
	}
	assert.Equal(t, resource.MustParse("32Mi"), *c.Resources.Limits.Memory())
	testContainer.ImagePullPolicy = corev1.PullAlways
	testContainer.ApplyToContainer(c)
	assert.Equal(t, corev1.PullAlways, c.ImagePullPolicy)
	c.ImagePullPolicy = corev1.PullIfNotPresent
	testContainer.ApplyToContainer(c)
	assert.Equal(t, corev1.PullIfNotPresent, c.ImagePullPolicy)
	testContainer.SecurityContext = &corev1.SecurityContext{}
	testContainer.ApplyToContainer(c)
	assert.NotNil(t, c.SecurityContext)
	testContainer.Env = []corev1.EnvVar{{Name: "a", Value: "b"}}
	testContainer.EnvFrom = []corev1.EnvFromSource{{Prefix: "a", ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "b"}}}}
	testContainer.ApplyToContainer(c)
	envs := []string{}
	for _, e := range c.Env {
		envs = append(envs, e.Name)
	}
	assert.Contains(t, envs, "a")
	envFroms := []string{}
	for _, e := range c.EnvFrom {
		envFroms = append(envFroms, e.Prefix)
	}
	assert.Contains(t, envFroms, "a")
	testContainer.VolumeMounts = []corev1.VolumeMount{{Name: "test", MountPath: "/test"}}
	testContainer.ApplyToContainer(c)
	assert.Equal(t, 1, len(c.VolumeMounts))
	assert.Equal(t, "test", c.VolumeMounts[0].Name)
}
