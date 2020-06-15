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

	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for storage grid events
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
	// k8sClient is Kubernetes client
	K8sClient kubernetes.Interface
	Namespace string
}

// Router manages route
type Router struct {
	// route contains configuration of a REST endpoint
	route *webhook.Route
	// storageGridEventSource refers to event source which contains configuration to consume events from storage grid
	storageGridEventSource *v1alpha1.StorageGridEventSource
	// k8sClient is Kubernetes client
	k8sClient kubernetes.Interface
	namespace string
}

type storageGridNotificationRequest struct {
	Notification string `json:"notification"`
}

// storageGridNotification is the bucket notification received from storage grid
type storageGridNotification struct {
	Action  string `json:"Action"`
	Message struct {
		Records []struct {
			EventVersion string    `json:"eventVersion"`
			EventSource  string    `json:"storageGridEventSource"`
			EventTime    time.Time `json:"eventTime"`
			EventName    string    `json:"eventName"`
			UserIdentity struct {
				PrincipalID string `json:"principalId"`
			} `json:"userIdentity"`
			RequestParameters struct {
				SourceIPAddress string `json:"sourceIPAddress"`
			} `json:"requestParameters"`
			ResponseElements struct {
				XAmzRequestID string `json:"x-amz-request-id"`
			} `json:"responseElements"`
			S3 struct {
				S3SchemaVersion string `json:"s3SchemaVersion"`
				ConfigurationID string `json:"configurationId"`
				Bucket          struct {
					Name          string `json:"name"`
					OwnerIdentity struct {
						PrincipalID string `json:"principalId"`
					} `json:"ownerIdentity"`
					Arn string `json:"arn"`
				} `json:"bucket"`
				Object struct {
					Key       string `json:"key"`
					Size      int    `json:"size"`
					ETag      string `json:"eTag"`
					Sequencer string `json:"sequencer"`
				} `json:"object"`
			} `json:"s3"`
		} `json:"Records"`
	} `json:"Message"`
	TopicArn string `json:"TopicArn"`
	Version  string `json:"Version"`
}

type registerNotificationResponse struct {
	ResponseTime time.Time `json:“responseTime”`
	Status       string    `json:“status”`
	APIVersion   string    `json:“apiVersion”`
	Deprecated   bool      `json:“deprecated”`
	Code         int       `json:“code”`
	Message      struct {
		Text             string `json:“text”`
		Key              string `json:“key”`
		Context          string `json:“context”`
		DeveloperMessage string `json:“developerMessage”`
	} `json:“message”`
	Errors []struct {
		Text             string `json:“text”`
		Key              string `json:“key”`
		Context          string `json:“context”`
		DeveloperMessage string `json:“developerMessage”`
	} `json:“errors”`
}

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

	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for storage grid events
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
	// k8sClient is Kubernetes client
	K8sClient kubernetes.Interface
	Namespace string
}

// Router manages route
type Router struct {
	// route contains configuration of a REST endpoint
	route *webhook.Route
	// storageGridEventSource refers to event source which contains configuration to consume events from storage grid
	storageGridEventSource *v1alpha1.StorageGridEventSource
	// k8sClient is Kubernetes client
	k8sClient kubernetes.Interface
	namespace string
}

type storageGridNotificationRequest struct {
	Notification string `json:"notification"`
}

// storageGridNotification is the bucket notification received from storage grid
type storageGridNotification struct {
	Action  string `json:"Action"`
	Message struct {
		Records []struct {
			EventVersion string    `json:"eventVersion"`
			EventSource  string    `json:"storageGridEventSource"`
			EventTime    time.Time `json:"eventTime"`
			EventName    string    `json:"eventName"`
			UserIdentity struct {
				PrincipalID string `json:"principalId"`
			} `json:"userIdentity"`
			RequestParameters struct {
				SourceIPAddress string `json:"sourceIPAddress"`
			} `json:"requestParameters"`
			ResponseElements struct {
				XAmzRequestID string `json:"x-amz-request-id"`
			} `json:"responseElements"`
			S3 struct {
				S3SchemaVersion string `json:"s3SchemaVersion"`
				ConfigurationID string `json:"configurationId"`
				Bucket          struct {
					Name          string `json:"name"`
					OwnerIdentity struct {
						PrincipalID string `json:"principalId"`
					} `json:"ownerIdentity"`
					Arn string `json:"arn"`
				} `json:"bucket"`
				Object struct {
					Key       string `json:"key"`
					Size      int    `json:"size"`
					ETag      string `json:"eTag"`
					Sequencer string `json:"sequencer"`
				} `json:"object"`
			} `json:"s3"`
		} `json:"Records"`
	} `json:"Message"`
	TopicArn string `json:"TopicArn"`
	Version  string `json:"Version"`
}

type registerNotificationResponse struct {
	ResponseTime time.Time `json:“responseTime”`
	Status       string    `json:“status”`
	APIVersion   string    `json:“apiVersion”`
	Deprecated   bool      `json:“deprecated”`
	Code         int       `json:“code”`
	Message      struct {
		Text             string `json:“text”`
		Key              string `json:“key”`
		Context          string `json:“context”`
		DeveloperMessage string `json:“developerMessage”`
	} `json:“message”`
	Errors []struct {
		Text             string `json:“text”`
		Key              string `json:“key”`
		Context          string `json:“context”`
		DeveloperMessage string `json:“developerMessage”`
	} `json:“errors”`
}

type getEndpointRequest struct {
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
