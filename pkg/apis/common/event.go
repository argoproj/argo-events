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

package common

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventProtocolType is type of the event dispatch protocol. Used for dispatching events
type EventProtocolType string

// possible types of event dispatch protocol
const (
	HTTP EventProtocolType = "HTTP"
	NATS EventProtocolType = "NATS"
)

// Type of nats connection.
type NatsType string

// possible values of nats connection type
const (
	Standard  NatsType = "Standard"
	Streaming NatsType = "Streaming"
)

// Event is a data record expressing an occurrence and its context.
// Adheres to the CloudEvents v0.1 specification
type Event struct {
	Context EventContext `json:"context" protobuf:"bytes,1,opt,name=context"`
	Payload []byte       `json:"payload" protobuf:"bytes,2,opt,name=data"`
}

// EventContext contains metadata that provides circumstantial information about the occurrence.
type EventContext struct {
	// The type of occurrence which has happened. Often this attribute is used for
	// routing, observability, policy enforcement, etc.
	// should be prefixed with a reverse-DNS name. The prefixed domain dictates
	// the organization which defines the semantics of this event type. ex: com.github.pull.create
	EventType string `json:"eventType" protobuf:"bytes,1,opt,name=eventType"`

	// The version of the eventType. Enables the interpretation of data by eventual consumers,
	// requires the consumer to be knowledgeable about the producer.
	EventTypeVersion string `json:"eventTypeVersion" protobuf:"bytes,2,opt,name=eventTypeVersion"`

	// The version of the CloudEvents specification which the event uses.
	// Enables the interpretation of the context.
	CloudEventsVersion string `json:"cloudEventsVersion" protobuf:"bytes,3,opt,name=cloudEventsVersion"`

	// This describes the event producer.
	Source *URI `json:"source" protobuf:"bytes,4,opt,name=source"`

	// ID of the event. The semantics are explicitly undefined to ease the implementation of producers.
	// Enables deduplication. Must be unique within scope of producer.
	EventID string `json:"eventID" protobuf:"bytes,5,opt,name=eventID"`

	// Timestamp of when the event happened. Must adhere to format specified in RFC 3339.

	EventTime metav1.MicroTime `json:"eventTime" protobuf:"bytes,6,opt,name=eventTime"`

	// A link to the schema that the data attribute adheres to.
	// Must adhere to the format specified in RFC 3986.
	SchemaURL *URI `json:"schemaURL" protobuf:"bytes,7,opt,name=schemaURL"`

	// Content type of the data attribute value. Enables the data attribute to carry any type of content,
	// whereby format and encoding might differ from that of the chosen event format.
	// For example, the data attribute may carry an XML or JSON payload and the consumer is informed
	// by this attribute being set to "application/xml" or "application/json" respectively.
	ContentType string `json:"contentType" protobuf:"bytes,8,opt,name=contentType"`

	// This is for additional metadata and does not have a mandated structure.
	// Enables a place for custom fields a producer or middleware might want to include and provides a place
	// to test metadata before adding them to the CloudEvents specification.
	Extensions map[string]string `json:"extensions,omitempty" protobuf:"bytes,9,rep,name=extensions"`
}

// URI is a Uniform Resource Identifier based on RFC 3986
type URI struct {
	Scheme   string `json:"scheme" protobuf:"bytes,1,opt,name=scheme"`
	User     string `json:"user" protobuf:"bytes,2,opt,name=user"`
	Password string `json:"password" protobuf:"bytes,3,opt,name=password"`
	Host     string `json:"host" protobuf:"bytes,4,opt,name=host"`
	Port     int32  `json:"port" protobuf:"bytes,5,opt,name=port"`
	Path     string `json:"path" protobuf:"bytes,6,opt,name=path"`
	Query    string `json:"query" protobuf:"bytes,7,opt,name=query"`
	Fragment string `json:"fragment" protobuf:"bytes,8,opt,name=fragment"`
}

// Dispatch protocol contains configuration necessary to dispatch an event to sensor over different communication protocols
type EventProtocol struct {
	Type EventProtocolType `json:"type" protobuf:"bytes,1,opt,name=type"`

	Http Http `json:"http" protobuf:"bytes,2,opt,name=http"`

	Nats Nats `json:"nats" protobuf:"bytes,3,opt,name=nats"`
}

// Http contains the information required to setup a http server and listen to incoming events
type Http struct {
	// Port on which server will run
	Port string `json:"port" protobuf:"bytes,1,opt,name=port"`
}

// Nats contains the information required to connect to nats server and get subscriptions
type Nats struct {
	// URL is nats server/service URL
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`

	// Subscribe starting with most recently published value. Refer https://github.com/nats-io/go-nats-streaming
	StartWithLastReceived bool `json:"startWithLastReceived,omitempty" protobuf:"bytes,2,opt,name=startWithLastReceived"`

	// Receive all stored values in order.
	DeliverAllAvailable bool `json:"deliverAllAvailable,omitempty" protobuf:"bytes,3,opt,name=deliverAllAvailable"`

	// Receive messages starting at a specific sequence number
	StartAtSequence string `json:"startAtSequence,omitempty" protobuf:"bytes,4,opt,name=startAtSequence"`

	// Subscribe starting at a specific time
	StartAtTime string `json:"startAtTime,omitempty" protobuf:"bytes,5,opt,name=startAtTime"`

	// Subscribe starting a specific amount of time in the past (e.g. 30 seconds ago)
	StartAtTimeDelta string `json:"startAtTimeDelta,omitempty" protobuf:"bytes,6,opt,name=startAtTimeDelta"`

	// Durable subscriptions allow clients to assign a durable name to a subscription when it is created
	Durable bool `json:"durable,omitempty" protobuf:"bytes,7,opt,name=durable"`

	// The NATS Streaming cluster ID
	ClusterId string `json:"clusterId,omitempty" protobuf:"bytes,8,opt,name=clusterId"`

	// The NATS Streaming cluster ID
	ClientId string `json:"clientId,omitempty" protobuf:"bytes,9,opt,name=clientId"`

	// Type of the connection. either standard or streaming
	Type NatsType `json:"type" protobuf:"bytes,10,opt,name=type"`
}

// ServiceTemplateSpec is the template spec contains metadata and service spec.
type ServiceTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec corev1.ServiceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
