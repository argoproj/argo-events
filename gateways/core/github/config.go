/*
Copyright 2018 KompiTech GmbH

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

package github

import (
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventSourceListener implements ConfigExecutor
type EventSourceListener struct {
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// RouteConfig contains information about the route
type RouteConfig struct {
	route             *gwcommon.Route
	githubEventSource *v1alpha1.GithubEventSource
	client            *github.Client
	hook              *github.Hook
	clientset         kubernetes.Interface
	namespace         string
}

// cred stores the api access token or webhook secret
type cred struct {
	secret string
}
