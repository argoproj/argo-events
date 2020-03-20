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

package gitlab

import (
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	Logger    *logrus.Logger
	Namespace string
	// K8sClient is kubernetes client
	K8sClient kubernetes.Interface
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// K8sClient is the Kubernetes client
	k8sClient kubernetes.Interface
	// gitlabClient is the client to connect to GitLab
	gitlabClient *gitlab.Client
	// hook is gitlab project hook
	// GitLab API docs:
	// https://docs.gitlab.com/ce/api/projects.html#list-project-hooks
	hook *gitlab.ProjectHook
	// gitlabEventSource is the event source that contains configuration necessary to consume events from GitLab
	gitlabEventSource *v1alpha1.GitlabEventSource
}

// cred stores the api access token
type cred struct {
	// token is gitlab api access token
	token string
}
