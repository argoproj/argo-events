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

package storagegrid

import (
	"time"

	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for storage grid events
type EventListener struct {
	EventSourceName        string
	EventName              string
	StorageGridEventSource v1alpha1.StorageGridEventSource
}

// Router manages route
type Router struct {
	// route contains configuration of a REST endpoint
	route *webhook.Route
	// storageGridEventSource refers to event source which contains configuration to consume events from storage grid
	storageGridEventSource *v1alpha1.StorageGridEventSource
}

type storageGridNotificationRequest struct {
	Notification string `json:"notification"`
}

type registerNotificationResponse struct {
	ResponseTime time.Time `json:"responseTime"`
	Status       string    `json:"status"`
	APIVersion   string    `json:"apiVersion"`
	Deprecated   bool      `json:"deprecated"`
	Code         int       `json:"code"`
	Message      struct {
		Text             string `json:"text"`
		Key              string `json:"key"`
		Context          string `json:"context"`
		DeveloperMessage string `json:"developerMessage"`
	} `json:"message"`
	Errors []struct {
		Text             string `json:"text"`
		Key              string `json:"key"`
		Context          string `json:"context"`
		DeveloperMessage string `json:"developerMessage"`
	} `json:"errors"`
}

type getEndpointResponse struct {
	ResponseTime time.Time `json:"responseTime"`
	Status       string    `json:"status"`
	APIVersion   string    `json:"apiVersion"`
	Deprecated   bool      `json:"deprecated"`
	Data         []struct {
		DisplayName string `json:"displayName"`
		EndpointURI string `json:"endpointURI"`
		EndpointURN string `json:"endpointURN"`
		AuthType    string `json:"authType"`
		CaCert      string `json:"caCert"`
		InsecureTLS bool   `json:"insecureTLS"`
		Credentials struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		} `json:"credentials"`
		BasicHTTPCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"basicHttpCredentials"`
		Error struct {
			Text string    `json:"text"`
			Time time.Time `json:"time"`
			Key  string    `json:"key"`
		} `json:"error"`
		ID string `json:"id"`
	} `json:"data"`
}

type createEndpointRequest struct {
	DisplayName string `json:"displayName"`
	EndpointURI string `json:"endpointURI"`
	EndpointURN string `json:"endpointURN"`
	AuthType    string `json:"authType"`
	InsecureTLS bool   `json:"insecureTLS"`
}

type genericResponse struct {
	ResponseTime time.Time `json:"responseTime"`
	Status       string    `json:"status"`
	APIVersion   string    `json:"apiVersion"`
	Deprecated   bool      `json:"deprecated"`
	Code         int       `json:"code"`
	Message      struct {
		Text             string `json:"text"`
		Key              string `json:"key"`
		Context          string `json:"context"`
		DeveloperMessage string `json:"developerMessage"`
	} `json:"message"`
	Errors []struct {
		Text             string `json:"text"`
		Key              string `json:"key"`
		Context          string `json:"context"`
		DeveloperMessage string `json:"developerMessage"`
	} `json:"errors"`
}
