/*
Copyright 2020 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package argo_workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// ArgoWorkflowTrigger implements Trigger interface for Argo workflow
type ArgoWorkflowTrigger struct {
	// K8sClient is Kubernetes client
	K8sClient kubernetes.Interface
	// ArgoClient is Argo Workflow client
	DynamicClient dynamic.Interface
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *zap.SugaredLogger

	namespableDynamicClient dynamic.NamespaceableResourceInterface
	cmdRunner               func(cmd *exec.Cmd) error
}

// NewArgoWorkflowTrigger returns a new Argo workflow trigger
func NewArgoWorkflowTrigger(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) *ArgoWorkflowTrigger {
	return &ArgoWorkflowTrigger{
		K8sClient:     k8sClient,
		DynamicClient: dynamicClient,
		Sensor:        sensor,
		Trigger:       trigger,
		Logger:        logger.With(logging.LabelTriggerType, apicommon.ArgoWorkflowTrigger),
		cmdRunner: func(cmd *exec.Cmd) error {
			return cmd.Run()
		},
	}
}

// GetTriggerType returns the type of the trigger
func (t *ArgoWorkflowTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.ArgoWorkflowTrigger
}

// FetchResource fetches the trigger resource from external source
func (t *ArgoWorkflowTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	trigger := t.Trigger
	return triggers.FetchKubernetesResource(trigger.Template.ArgoWorkflow.Source)
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *ArgoWorkflowTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}
	if err := triggers.ApplyResourceParameters(events, t.Trigger.Template.ArgoWorkflow.Parameters, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// Execute executes the trigger
func (t *ArgoWorkflowTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger := t.Trigger

	op := v1alpha1.Submit
	if trigger.Template.ArgoWorkflow.Operation != "" {
		op = trigger.Template.ArgoWorkflow.Operation
	}

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	name := obj.GetName()
	if name == "" {
		if op != v1alpha1.Submit {
			return nil, errors.Errorf("failed to execute the workflow %v operation, no name is given", op)
		}
		if obj.GetGenerateName() == "" {
			return nil, errors.New("failed to trigger the workflow, neither name nor generateName is given")
		}
	}

	submittedWFLabels := make(map[string]string)
	if op == v1alpha1.Submit {
		submittedWFLabels["events.argoproj.io/sensor"] = t.Sensor.Name
		submittedWFLabels["events.argoproj.io/trigger"] = trigger.Template.Name
		submittedWFLabels["events.argoproj.io/action-timestamp"] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
	}

	namespace := obj.GetNamespace()
	if namespace == "" {
		namespace = t.Sensor.Namespace
	}

	var cmd *exec.Cmd

	switch op {
	case v1alpha1.Submit:
		file, err := os.CreateTemp("", fmt.Sprintf("%s%s", name, obj.GetGenerateName()))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create a temp file for the workflow %s", obj.GetName())
		}
		defer os.Remove(file.Name())

		// Add labels
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for k, v := range submittedWFLabels {
			labels[k] = v
		}
		obj.SetLabels(labels)

		jObj, err := obj.MarshalJSON()
		if err != nil {
			return nil, err
		}

		if _, err := file.Write(jObj); err != nil {
			return nil, errors.Wrapf(err, "failed to write workflow json %s to the temp file %s", name, file.Name())
		}
		cmd = exec.Command("argo", "-n", namespace, "submit", file.Name())
	case v1alpha1.SubmitFrom:
		wf := obj.GetKind()
		switch strings.ToLower(wf) {
		case "cronworkflow":
			wf = "cronwf"
		case "workflow":
			wf = "wf"
		default:
			return nil, errors.Errorf("invalid kind %s", wf)
		}
		fromArg := fmt.Sprintf("%s/%s", wf, name)
		cmd = exec.Command("argo", "-n", namespace, "submit", "--from", fromArg)
	case v1alpha1.Resubmit:
		cmd = exec.Command("argo", "-n", namespace, "resubmit", name)
	case v1alpha1.Resume:
		cmd = exec.Command("argo", "-n", namespace, "resume", name)
	case v1alpha1.Retry:
		cmd = exec.Command("argo", "-n", namespace, "retry", name)
	case v1alpha1.Suspend:
		cmd = exec.Command("argo", "-n", namespace, "suspend", name)
	case v1alpha1.Terminate:
		cmd = exec.Command("argo", "-n", namespace, "terminate", name)
	case v1alpha1.Stop:
		cmd = exec.Command("argo", "-n", namespace, "stop", name)
	default:
		return nil, errors.Errorf("unknown operation type %s", string(op))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Args = append(cmd.Args, trigger.Template.ArgoWorkflow.Args...)
	if err := t.cmdRunner(cmd); err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s command for workflow %s", string(op), name)
	}

	t.namespableDynamicClient = t.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "workflows",
	})

	if op != v1alpha1.Submit {
		return t.namespableDynamicClient.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	l, err := t.namespableDynamicClient.Namespace(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(submittedWFLabels).String()})
	if err != nil {
		return nil, err
	}
	if len(l.Items) == 0 {
		return nil, errors.New("failed to list created workflows for unknown reason")
	}
	return l.Items[0], nil
}

// ApplyPolicy applies the policy on the trigger
func (t *ArgoWorkflowTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	trigger := t.Trigger

	if trigger.Policy == nil || trigger.Policy.K8s == nil || trigger.Policy.K8s.Labels == nil {
		return nil
	}

	obj, ok := resource.(*unstructured.Unstructured)
	if !ok {
		return errors.New("failed to interpret the trigger resource")
	}

	p := policy.NewResourceLabels(trigger, t.namespableDynamicClient, obj)
	if p == nil {
		return nil
	}

	err := p.ApplyPolicy(ctx)
	if err != nil {
		switch err {
		case wait.ErrWaitTimeout:
			if trigger.Policy.K8s.ErrorOnBackoffTimeout {
				return errors.Errorf("failed to determine status of the triggered resource. setting trigger state as failed")
			}
			return nil
		default:
			return err
		}
	}

	return nil
}
