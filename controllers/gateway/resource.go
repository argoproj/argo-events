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
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// createGatewayResources creates gateway deployment and service
func (ctx *gatewayContext) createGatewayResources() error {
	return ctx.updateGatewayResources()
}

// updateGatewayResources updates gateway deployment and service
func (ctx *gatewayContext) updateGatewayResources() error {
	svc, err := ctx.getService()
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if svc != nil {
		err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	deploy, err := ctx.getDeployment()
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if deploy != nil {
		err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).Delete(deploy.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *gatewayContext) getService() (*corev1.Service, error) {
	sl, err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, ctx.gateway) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (ctx *gatewayContext) getDeployment() (*appv1.Deployment, error) {
	dl, err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range dl.Items {
		if metav1.IsControlledBy(&d, ctx.gateway) {
			return &d, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}
