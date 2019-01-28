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

import "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	EventTime v1.MicroTime `json:"eventTime" protobuf:"bytes,6,opt,name=eventTime"`

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
