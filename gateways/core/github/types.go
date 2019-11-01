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
	"github.com/argoproj/argo-events/gateways/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for GitHub event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
	// K8sClient is the Kubernetes client
	K8sClient kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// Router contains information about the route
type Router struct {
	// route contains configuration for an API endpoint
	route *webhook.Route
	// githubEventSource is the event source that holds information to consume events from GitHub
	githubEventSource *v1alpha1.GithubEventSource
	// githubClient is the client to connect to GitHub
	githubClient *github.Client
	// hook represents a GitHub (web and service) hook for a repository.
	hook *github.Hook
	// K8sClient is the Kubernetes client
	k8sClient kubernetes.Interface
}

// cred stores the api access token or webhook secret
type cred struct {
	secret string
}
