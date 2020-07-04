/*
Copyright 2018 BlackRock, Inc.

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

package gateway

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

var testEventBus = &eventbusv1alpha1.EventBus{
	TypeMeta: metav1.TypeMeta{
		APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
		Kind:       "EventBus",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: common.DefaultControllerNamespace,
		Name:      "default",
	},
	Spec: eventbusv1alpha1.EventBusSpec{
		NATS: &eventbusv1alpha1.NATSBus{
			Native: &eventbusv1alpha1.NativeStrategy{
				Replicas: 3,
				Auth:     &eventbusv1alpha1.AuthStrategyToken,
			},
		},
	},
}

var gatewayObj = &v1alpha1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-gateway-1",
		Namespace: common.DefaultControllerNamespace,
	},
	Spec: v1alpha1.GatewaySpec{
		EventSourceRef: &v1alpha1.EventSourceRef{
			Name: "fake-event-source",
		},
		Replica:       1,
		Type:          apicommon.WebhookEvent,
		ProcessorPort: "8080",
		Template: v1alpha1.Template{
			ServiceAccountName: "fake-sa",
			Container: &corev1.Container{
				Image: "argoproj/fake-image",
			},
		},
		Service: &v1alpha1.Service{
			Ports: []corev1.ServicePort{
				{
					Name:       "server-port",
					Port:       12000,
					TargetPort: intstr.FromInt(12000),
				},
			},
		},
	},
}

var gatewayObjNoTemplate = &v1alpha1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-gateway-2",
		Namespace: common.DefaultControllerNamespace,
	},
	Spec: v1alpha1.GatewaySpec{
		EventSourceRef: &v1alpha1.EventSourceRef{
			Name: "fake-event-source",
		},
		Replica:       1,
		Type:          apicommon.WebhookEvent,
		ProcessorPort: "8080",
		Service: &v1alpha1.Service{
			Ports: []corev1.ServicePort{
				{
					Name:       "server-port",
					Port:       12000,
					TargetPort: intstr.FromInt(12000),
				},
			},
		},
	},
}
