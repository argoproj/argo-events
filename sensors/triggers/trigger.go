package triggers

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	standard_k8s "github.com/argoproj/argo-events/sensors/triggers/standard-k8s"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Trigger interface
type Trigger interface {
	// FetchResource fetches the trigger resource from external source
	FetchResource() (interface{}, error)
	// ApplyResourceParameters applies parameters to the trigger resource
	ApplyResourceParameters(sensor *v1alpha1.Sensor, parameters []v1alpha1.TriggerParameter, resource interface{}) error
	// Execute executes the trigger
	Execute(resource interface{}) (interface{}, error)
	// ApplyPolicy applies the policy on the trigger
	ApplyPolicy(resource interface{}) error
}

// GetTrigger returns a trigger
func GetTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, logger *logrus.Logger) Trigger {
	if trigger.Template.K8s != nil {
		return standard_k8s.NewStandardK8sTrigger(k8sClient, dynamicClient, sensor, trigger, logger)
	}
	return nil
}
