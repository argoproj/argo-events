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

// EventSourceType is the type of event source supported by the gateway
type EventSourceType string

// possible event source types
var (
	MinioEvent       EventSourceType = "minio"
	CalendarEvent    EventSourceType = "calendar"
	FileEvent        EventSourceType = "file"
	ResourceEvent    EventSourceType = "resource"
	WebhookEvent     EventSourceType = "webhook"
	AMQPEvent        EventSourceType = "amqp"
	KafkaEvent       EventSourceType = "kafka"
	MQTTEvent        EventSourceType = "mqtt"
	NATSEvent        EventSourceType = "nats"
	SNSEvent         EventSourceType = "sns"
	SQSEvent         EventSourceType = "sqs"
	PubSubEvent      EventSourceType = "pubsub"
	GitHubEvent      EventSourceType = "github"
	GitLabEvent      EventSourceType = "gitlab"
	HDFSEvent        EventSourceType = "hdfs"
	SlackEvent       EventSourceType = "slack"
	StorageGridEvent EventSourceType = "storagegrid"
	AzureEventsHub   EventSourceType = "azureEventsHub"
	StripeEvent      EventSourceType = "stripe"
	EmitterEvent     EventSourceType = "emitter"
	RedisEvent       EventSourceType = "redis"
	NSQEvent         EventSourceType = "nsq"
	GenericEvent     EventSourceType = "generic"
)

// EventBusType is the type of event bus
type EventBusType string

// possible event bus types
var (
	EventBusNATS EventBusType = "nats"
)
