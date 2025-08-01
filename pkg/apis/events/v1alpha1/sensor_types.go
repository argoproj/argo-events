package v1alpha1

import (
	"encoding/base64"
	"fmt"
	"mime"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// KubernetesResourceOperation refers to the type of operation performed on the K8s resource
type KubernetesResourceOperation string

// possible values for KubernetesResourceOperation
const (
	Create KubernetesResourceOperation = "create" // create the resource
	Update KubernetesResourceOperation = "update" // updates the resource
	Patch  KubernetesResourceOperation = "patch"  // patch resource
	Delete KubernetesResourceOperation = "delete" // deletes the resource
)

// ArgoWorkflowOperation refers to the type of the operation performed on the Argo Workflow
type ArgoWorkflowOperation string

// possible values for ArgoWorkflowOperation
const (
	Submit     ArgoWorkflowOperation = "submit"      // submit a workflow
	SubmitFrom ArgoWorkflowOperation = "submit-from" // submit from existing resource
	Suspend    ArgoWorkflowOperation = "suspend"     // suspends a workflow
	Resubmit   ArgoWorkflowOperation = "resubmit"    // resubmit a workflow
	Retry      ArgoWorkflowOperation = "retry"       // retry a workflow
	Resume     ArgoWorkflowOperation = "resume"      // resume a workflow
	Terminate  ArgoWorkflowOperation = "terminate"   // terminate a workflow
	Stop       ArgoWorkflowOperation = "stop"        // stop a workflow
)

// Comparator refers to the comparator operator for a data filter
type Comparator string

const (
	GreaterThanOrEqualTo Comparator = ">=" // Greater than or equal to value provided in data filter
	GreaterThan          Comparator = ">"  // Greater than value provided in data filter
	EqualTo              Comparator = "="  // Equal to value provided in data filter
	NotEqualTo           Comparator = "!=" // Not equal to value provided in data filter
	LessThan             Comparator = "<"  // Less than value provided in data filter
	LessThanOrEqualTo    Comparator = "<=" // Less than or equal to value provided in data filter
	EmptyComparator                 = ""   // Equal to value provided in data filter
)

// Sensor is the definition of a sensor resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=sn
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type Sensor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              SensorSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// +optional
	Status SensorStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// SensorList is the list of Sensor resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SensorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Sensor `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// SensorSpec represents desired sensor state
type SensorSpec struct {
	// Dependencies is a list of the events that this sensor is dependent on.
	Dependencies []EventDependency `json:"dependencies" protobuf:"bytes,1,rep,name=dependencies"`
	// Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor.
	Triggers []Trigger `json:"triggers" protobuf:"bytes,2,rep,name=triggers"`
	// Template is the pod specification for the sensor
	// +optional
	Template *Template `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`
	// ErrorOnFailedRound if set to true, marks sensor state as `error` if the previous trigger round fails.
	// Once sensor state is set to `error`, no further triggers will be processed.
	ErrorOnFailedRound bool `json:"errorOnFailedRound,omitempty" protobuf:"varint,4,opt,name=errorOnFailedRound"`
	// EventBusName references to a EventBus name. By default the value is "default"
	EventBusName string `json:"eventBusName,omitempty" protobuf:"bytes,5,opt,name=eventBusName"`
	// Replicas is the sensor deployment replicas
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,6,opt,name=replicas"`
	// RevisionHistoryLimit specifies how many old deployment revisions to retain
	// +optional
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty" protobuf:"varint,7,opt,name=revisionHistoryLimit"`
	// LoggingFields add additional key-value pairs when logging happens
	// +optional
	LoggingFields map[string]string `json:"loggingFields" protobuf:"bytes,8,rep,name=loggingFields"`
}

func (s SensorSpec) GetReplicas() int32 {
	if s.Replicas == nil {
		return 1
	}
	replicas := *s.Replicas
	if replicas < 1 {
		replicas = 1
	}
	return replicas
}

type LogicalOperator string

const (
	AndLogicalOperator   LogicalOperator = "and" // Equal to &&
	OrLogicalOperator    LogicalOperator = "or"  // Equal to ||
	EmptyLogicalOperator LogicalOperator = ""    // Empty will default to AND (&&)
)

// EventDependency describes a dependency
type EventDependency struct {
	// Name is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// EventSourceName is the name of EventSource that Sensor depends on
	EventSourceName string `json:"eventSourceName" protobuf:"bytes,2,name=eventSourceName"`
	// EventName is the name of the event
	EventName string `json:"eventName" protobuf:"bytes,3,name=eventName"`
	// Filters and rules governing toleration of success and constraints on the context and data of an event
	Filters *EventDependencyFilter `json:"filters,omitempty" protobuf:"bytes,4,opt,name=filters"`
	// Transform transforms the event data
	Transform *EventDependencyTransformer `json:"transform,omitempty" protobuf:"bytes,5,opt,name=transform"`
	// FiltersLogicalOperator defines how different filters are evaluated together.
	// Available values: and (&&), or (||)
	// Is optional and if left blank treated as and (&&).
	FiltersLogicalOperator LogicalOperator `json:"filtersLogicalOperator,omitempty" protobuf:"bytes,6,opt,name=filtersLogicalOperator,casttype=LogicalOperator"`
}

// EventDependencyTransformer transforms the event
type EventDependencyTransformer struct {
	// JQ holds the jq command applied for transformation
	// +optional
	JQ string `json:"jq,omitempty" protobuf:"bytes,1,opt,name=jq"`
	// Script refers to a Lua script used to transform the event
	// +optional
	Script string `json:"script,omitempty" protobuf:"bytes,2,opt,name=script"`
}

// EventDependencyFilter defines filters and constraints for a event.
type EventDependencyFilter struct {
	// Time filter on the event with escalation
	Time *TimeFilter `json:"time,omitempty" protobuf:"bytes,1,opt,name=time"`
	// Context filter constraints
	Context *EventContext `json:"context,omitempty" protobuf:"bytes,2,opt,name=context"`
	// Data filter constraints with escalation
	Data []DataFilter `json:"data,omitempty" protobuf:"bytes,3,rep,name=data"`
	// Exprs contains the list of expressions evaluated against the event payload.
	Exprs []ExprFilter `json:"exprs,omitempty" protobuf:"bytes,4,rep,name=exprs"`
	// DataLogicalOperator defines how multiple Data filters (if defined) are evaluated together.
	// Available values: and (&&), or (||)
	// Is optional and if left blank treated as and (&&).
	DataLogicalOperator LogicalOperator `json:"dataLogicalOperator,omitempty" protobuf:"bytes,5,opt,name=dataLogicalOperator,casttype=DataLogicalOperator"`
	// ExprLogicalOperator defines how multiple Exprs filters (if defined) are evaluated together.
	// Available values: and (&&), or (||)
	// Is optional and if left blank treated as and (&&).
	ExprLogicalOperator LogicalOperator `json:"exprLogicalOperator,omitempty" protobuf:"bytes,6,opt,name=exprLogicalOperator,casttype=ExprLogicalOperator"`
	// Script refers to a Lua script evaluated to determine the validity of an event.
	Script string `json:"script,omitempty" protobuf:"bytes,7,opt,name=script"`
}

type ExprFilter struct {
	// Expr refers to the expression that determines the outcome of the filter.
	Expr string `json:"expr" protobuf:"bytes,1,opt,name=expr"`
	// Fields refers to set of keys that refer to the paths within event payload.
	Fields []PayloadField `json:"fields" protobuf:"bytes,2,rep,name=fields"`
}

// PayloadField binds a value at path within the event payload against a name.
type PayloadField struct {
	// Path is the JSONPath of the event's (JSON decoded) data key
	// Path is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`
	// Name acts as key that holds the value at the path.
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

// TimeFilter describes a window in time.
// It filters out events that occur outside the time limits.
// In other words, only events that occur after Start and before Stop
// will pass this filter.
type TimeFilter struct {
	// Start is the beginning of a time window in UTC.
	// Before this time, events for this dependency are ignored.
	// Format is hh:mm:ss.
	Start string `json:"start" protobuf:"bytes,1,opt,name=start"`
	// Stop is the end of a time window in UTC.
	// After or equal to this time, events for this dependency are ignored and
	// Format is hh:mm:ss.
	// If it is smaller than Start, it is treated as next day of Start
	// (e.g.: 22:00:00-01:00:00 means 22:00:00-25:00:00).
	Stop string `json:"stop" protobuf:"bytes,2,opt,name=stop"`
}

// JSONType contains the supported JSON types for data filtering
type JSONType string

// the various supported JSONTypes
const (
	JSONTypeBool   JSONType = "bool"
	JSONTypeNumber JSONType = "number"
	JSONTypeString JSONType = "string"
)

// DataFilter describes constraints and filters for event data.
type DataFilter struct {
	// Path is the JSONPath of the event's (JSON decoded) data key.
	// Path is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`
	// Type contains the JSON type of the data
	Type JSONType `json:"type" protobuf:"bytes,2,opt,name=type,casttype=JSONType"`
	// Value is the allowed string values for this key.
	// Booleans are parsed using strconv.ParseBool(),
	// Numbers are parsed as float64 using strconv.ParseFloat(),
	// Strings are treated as regular expressions,
	// Nils value is ignored.
	Value []string `json:"value" protobuf:"bytes,3,rep,name=value"`
	// Comparator compares the event data with a user given value.
	// Can be ">=", ">", "=", "!=", "<", or "<=".
	// Is optional, and if left blank treated as equality "=".
	Comparator Comparator `json:"comparator,omitempty" protobuf:"bytes,4,opt,name=comparator,casttype=Comparator"`
	// Template is a go-template for extracting a string from the event's data.
	// A Template is evaluated with provided path, type and value.
	// The templating follows the standard go-template syntax as well as sprig's extra functions.
	// See https://pkg.go.dev/text/template and https://masterminds.github.io/sprig/
	Template string `json:"template,omitempty" protobuf:"bytes,5,opt,name=template"`
}

// Trigger is an action taken, output produced, an event created, a message sent
type Trigger struct {
	// Template describes the trigger specification.
	Template *TriggerTemplate `json:"template,omitempty" protobuf:"bytes,1,opt,name=template"`
	// Parameters is the list of parameters applied to the trigger template definition
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`
	// Policy to configure backoff and execution criteria for the trigger
	// +optional
	Policy *TriggerPolicy `json:"policy,omitempty" protobuf:"bytes,3,opt,name=policy"`
	// Retry strategy, defaults to no retry
	// +optional
	RetryStrategy *Backoff `json:"retryStrategy,omitempty" protobuf:"bytes,4,opt,name=retryStrategy"`
	// Rate limit, default unit is Second
	// +optional
	RateLimit *RateLimit `json:"rateLimit,omitempty" protobuf:"bytes,5,opt,name=rateLimit"`
	// AtLeastOnce determines the trigger execution semantics.
	// Defaults to false. Trigger execution will use at-most-once semantics.
	// If set to true, Trigger execution will switch to at-least-once semantics.
	// +kubebuilder:default=false
	// +optional
	AtLeastOnce bool `json:"atLeastOnce,omitempty" protobuf:"varint,6,opt,name=atLeastOnce"`
	// If the trigger fails, it will retry up to the configured number of
	// retries. If the maximum retries are reached and the trigger is set to
	// execute atLeastOnce, the dead letter queue (DLQ) trigger will be invoked if
	// specified.  Invoking the dead letter queue trigger helps prevent data
	// loss.
	// +optional
	DlqTrigger *Trigger `json:"dlqTrigger,omitempty" protobuf:"bytes,7,opt,name=dlqTrigger"`
}

type RateLimiteUnit string

const (
	Second RateLimiteUnit = "Second"
	Minute RateLimiteUnit = "Minute"
	Hour   RateLimiteUnit = "Hour"
)

type RateLimit struct {
	// Defaults to Second
	Unit            RateLimiteUnit `json:"unit,omitempty" protobuf:"bytes,1,opt,name=unit"`
	RequestsPerUnit int32          `json:"requestsPerUnit,omitempty" protobuf:"bytes,2,opt,name=requestsPerUnit"`
}

// TriggerTemplate is the template that describes trigger specification.
type TriggerTemplate struct {
	// Name is a unique name of the action to take.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Conditions is the conditions to execute the trigger.
	// For example: "(dep01 || dep02) && dep04"
	// +optional
	Conditions string `json:"conditions,omitempty" protobuf:"bytes,2,opt,name=conditions"`
	// StandardK8STrigger refers to the trigger designed to create or update a generic Kubernetes resource.
	// +optional
	K8s *StandardK8STrigger `json:"k8s,omitempty" protobuf:"bytes,3,opt,name=k8s"`
	// ArgoWorkflow refers to the trigger that can perform various operations on an Argo workflow.
	// +optional
	ArgoWorkflow *ArgoWorkflowTrigger `json:"argoWorkflow,omitempty" protobuf:"bytes,4,opt,name=argoWorkflow"`
	// HTTP refers to the trigger designed to dispatch a HTTP request with on-the-fly constructable payload.
	// +optional
	HTTP *HTTPTrigger `json:"http,omitempty" protobuf:"bytes,5,opt,name=http"`
	// AWSLambda refers to the trigger designed to invoke AWS Lambda function with with on-the-fly constructable payload.
	// +optional
	AWSLambda *AWSLambdaTrigger `json:"awsLambda,omitempty" protobuf:"bytes,6,opt,name=awsLambda"`
	// CustomTrigger refers to the trigger designed to connect to a gRPC trigger server and execute a custom trigger.
	// +optional
	CustomTrigger *CustomTrigger `json:"custom,omitempty" protobuf:"bytes,7,opt,name=custom"`
	// Kafka refers to the trigger designed to place messages on Kafka topic.
	// +optional.
	Kafka *KafkaTrigger `json:"kafka,omitempty" protobuf:"bytes,8,opt,name=kafka"`
	// NATS refers to the trigger designed to place message on NATS subject.
	// +optional.
	NATS *NATSTrigger `json:"nats,omitempty" protobuf:"bytes,9,opt,name=nats"`
	// Slack refers to the trigger designed to send slack notification message.
	// +optional
	Slack *SlackTrigger `json:"slack,omitempty" protobuf:"bytes,10,opt,name=slack"`
	// OpenWhisk refers to the trigger designed to invoke OpenWhisk action.
	// +optional
	OpenWhisk *OpenWhiskTrigger `json:"openWhisk,omitempty" protobuf:"bytes,11,opt,name=openWhisk"`
	// Log refers to the trigger designed to invoke log the event.
	// +optional
	Log *LogTrigger `json:"log,omitempty" protobuf:"bytes,12,opt,name=log"`
	// AzureEventHubs refers to the trigger send an event to an Azure Event Hub.
	// +optional
	AzureEventHubs *AzureEventHubsTrigger `json:"azureEventHubs,omitempty" protobuf:"bytes,13,opt,name=azureEventHubs"`
	// Pulsar refers to the trigger designed to place messages on Pulsar topic.
	// +optional
	Pulsar *PulsarTrigger `json:"pulsar,omitempty" protobuf:"bytes,14,opt,name=pulsar"`
	// Criteria to reset the conditons
	// +optional
	ConditionsReset []ConditionsResetCriteria `json:"conditionsReset,omitempty" protobuf:"bytes,15,rep,name=conditionsReset"`
	// AzureServiceBus refers to the trigger designed to place messages on Azure Service Bus
	// +optional
	AzureServiceBus *AzureServiceBusTrigger `json:"azureServiceBus,omitempty" protobuf:"bytes,16,opt,name=azureServiceBus"`
	// Email refers to the trigger designed to send an email notification
	// +optional
	Email *EmailTrigger `json:"email,omitempty" protobuf:"bytes,17,opt,name=email"`
}

type ConditionsResetCriteria struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	ByTime *ConditionsResetByTime `json:"byTime,omitempty" protobuf:"bytes,1,opt,name=byTime"`
}

type ConditionsResetByTime struct {
	// Cron is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Cron string `json:"cron,omitempty" protobuf:"bytes,1,opt,name=cron"`
	// +optional
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,2,opt,name=timezone"`
}

// StandardK8STrigger is the standard Kubernetes resource trigger
type StandardK8STrigger struct {
	// Source of the K8s resource file(s)
	Source *ArtifactLocation `json:"source,omitempty" protobuf:"bytes,1,opt,name=source"`
	// Operation refers to the type of operation performed on the k8s resource.
	// Default value is Create.
	// +optional
	Operation KubernetesResourceOperation `json:"operation,omitempty" protobuf:"bytes,2,opt,name=operation,casttype=KubernetesResourceOperation"`
	// Parameters is the list of parameters that is applied to resolved K8s trigger object.
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`
	// PatchStrategy controls the K8s object patching strategy when the trigger operation is specified as patch.
	// possible values:
	// "application/json-patch+json"
	// "application/merge-patch+json"
	// "application/strategic-merge-patch+json"
	// "application/apply-patch+yaml".
	// Defaults to "application/merge-patch+json"
	// +optional
	PatchStrategy k8stypes.PatchType `json:"patchStrategy,omitempty" protobuf:"bytes,4,opt,name=patchStrategy,casttype=k8s.io/apimachinery/pkg/types.PatchType"`
	// LiveObject specifies whether the resource should be directly fetched from K8s instead
	// of being marshaled from the resource artifact. If set to true, the resource artifact
	// must contain the information required to uniquely identify the resource in the cluster,
	// that is, you must specify "apiVersion", "kind" as well as "name" and "namespace" meta
	// data.
	// Only valid for operation type `update`
	// +optional
	LiveObject bool `json:"liveObject,omitempty" protobuf:"varint,5,opt,name=liveObject"`
}

// ArgoWorkflowTrigger is the trigger for the Argo Workflow
type ArgoWorkflowTrigger struct {
	// Source of the K8s resource file(s)
	Source *ArtifactLocation `json:"source,omitempty" protobuf:"bytes,1,opt,name=source"`
	// Operation refers to the type of operation performed on the argo workflow resource.
	// Default value is Submit.
	// +optional
	Operation ArgoWorkflowOperation `json:"operation,omitempty" protobuf:"bytes,2,opt,name=operation,casttype=ArgoWorkflowOperation"`
	// Parameters is the list of parameters to pass to resolved Argo Workflow object
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`
	// Args is the list of arguments to pass to the argo CLI
	Args []string `json:"args,omitempty" protobuf:"bytes,4,rep,name=args"`
}

// HTTPTrigger is the trigger for the HTTP request
type HTTPTrigger struct {
	// URL refers to the URL to send HTTP request to.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Payload is the list of key-value extracted from an event payload to construct the HTTP request payload.

	Payload []TriggerParameter `json:"payload" protobuf:"bytes,2,rep,name=payload"`
	// TLS configuration for the HTTP client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,3,opt,name=tls"`
	// Method refers to the type of the HTTP request.
	// Refer https://golang.org/src/net/http/method.go for more info.
	// Default value is POST.
	// +optional
	Method string `json:"method,omitempty" protobuf:"bytes,4,opt,name=method"`
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the HTTP trigger resource.
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,5,rep,name=parameters"`
	// Timeout refers to the HTTP request timeout in seconds.
	// Default value is 60 seconds.
	// +optional
	Timeout int64 `json:"timeout,omitempty" protobuf:"varint,6,opt,name=timeout"`
	// BasicAuth configuration for the http request.
	// +optional
	BasicAuth *BasicAuth `json:"basicAuth,omitempty" protobuf:"bytes,7,opt,name=basicAuth"`
	// Headers for the HTTP request.
	// +optional
	Headers map[string]string `json:"headers,omitempty" protobuf:"bytes,8,rep,name=headers"`
	// Secure Headers stored in Kubernetes Secrets for the HTTP requests.
	// +optional
	SecureHeaders []*SecureHeader `json:"secureHeaders,omitempty" protobuf:"bytes,9,rep,name=secureHeaders"`
	// Dynamic Headers for the request, sourced from the event. Same spec as Parameters.
	// +optional
	DynamicHeaders []TriggerParameter `json:"dynamicHeaders,omitempty" protobuf:"bytes,10,rep,name=dynamicHeaders"`
}

// SecureHeader refers to HTTP Headers with auth tokens as values
type SecureHeader struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Values can be read from either secrets or configmaps
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty" protobuf:"bytes,2,opt,name=valueFrom"`
}

// ValueFromSource allows you to reference keys from either a Configmap or Secret
type ValueFromSource struct {
	SecretKeyRef    *corev1.SecretKeySelector    `json:"secretKeyRef,omitempty" protobuf:"bytes,1,opt,name=secretKeyRef"`
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty" protobuf:"bytes,2,opt,name=configMapKeyRef"`
}

// AWSLambdaTrigger refers to specification of the trigger to invoke an AWS Lambda function
type AWSLambdaTrigger struct {
	// FunctionName refers to the name of the function to invoke.
	FunctionName string `json:"functionName" protobuf:"bytes,1,opt,name=functionName"`
	// AccessKey refers K8s secret containing aws access key
	// +optional
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,2,opt,name=accessKey"`
	// SecretKey refers K8s secret containing aws secret key
	// +optional
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,3,opt,name=secretKey"`
	// Region is AWS region
	Region string `json:"region" protobuf:"bytes,4,opt,name=region"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,5,rep,name=payload"`
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,6,rep,name=parameters"`
	// Choose from the following options.
	//
	//    * RequestResponse (default) - Invoke the function synchronously. Keep
	//    the connection open until the function returns a response or times out.
	//    The API response includes the function response and additional data.
	//
	//    * Event - Invoke the function asynchronously. Send events that fail multiple
	//    times to the function's dead-letter queue (if it's configured). The API
	//    response only includes a status code.
	//
	//    * DryRun - Validate parameter values and verify that the user or role
	//    has permission to invoke the function.
	// +optional
	InvocationType *string `json:"invocationType,omitempty" protobuf:"bytes,7,opt,name=invocationType"`
	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	// +optional
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,8,opt,name=roleARN"`
}

// AzureEventHubsTrigger refers to specification of the Azure Event Hubs Trigger
type AzureEventHubsTrigger struct {
	// FQDN refers to the namespace dns of Azure Event Hubs to be used i.e. <namespace>.servicebus.windows.net
	FQDN string `json:"fqdn" protobuf:"bytes,1,opt,name=fqdn"`
	// HubName refers to the Azure Event Hub to send events to
	HubName string `json:"hubName" protobuf:"bytes,2,opt,name=hubName"`
	// SharedAccessKeyName refers to the name of the Shared Access Key
	SharedAccessKeyName *corev1.SecretKeySelector `json:"sharedAccessKeyName" protobuf:"bytes,3,opt,name=sharedAccessKeyName"`
	// SharedAccessKey refers to a K8s secret containing the primary key for the
	SharedAccessKey *corev1.SecretKeySelector `json:"sharedAccessKey,omitempty" protobuf:"bytes,4,opt,name=sharedAccessKey"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,5,rep,name=payload"`
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,6,rep,name=parameters"`
}

type AzureServiceBusTrigger struct {
	// ConnectionString is the connection string for the Azure Service Bus
	ConnectionString *corev1.SecretKeySelector `json:"connectionString,omitempty" protobuf:"bytes,1,opt,name=connectionString"`
	// QueueName is the name of the Azure Service Bus Queue
	QueueName string `json:"queueName" protobuf:"bytes,2,opt,name=queueName"`
	// TopicName is the name of the Azure Service Bus Topic
	TopicName string `json:"topicName" protobuf:"bytes,3,opt,name=topicName"`
	// SubscriptionName is the name of the Azure Service Bus Topic Subscription
	SubscriptionName string `json:"subscriptionName" protobuf:"bytes,4,opt,name=subscriptionName"`
	// TLS configuration for the service bus client
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,6,rep,name=payload"`
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,7,rep,name=parameters"`
}

// KafkaTrigger refers to the specification of the Kafka trigger.
type KafkaTrigger struct {
	// URL of the Kafka broker, multiple URLs separated by comma.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Name of the topic.
	// More info at https://kafka.apache.org/documentation/#intro_topics
	Topic string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	// +optional
	// DEPRECATED
	Partition int32 `json:"partition" protobuf:"varint,3,opt,name=partition"`
	// Parameters is the list of parameters that is applied to resolved Kafka trigger object.
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,4,rep,name=parameters"`
	// RequiredAcks used in producer to tell the broker how many replica acknowledgements
	// Defaults to 1 (Only wait for the leader to ack).
	// +optional.
	RequiredAcks int32 `json:"requiredAcks,omitempty" protobuf:"varint,5,opt,name=requiredAcks"`
	// Compress determines whether to compress message or not.
	// Defaults to false.
	// If set to true, compresses message using snappy compression.
	// +optional
	Compress bool `json:"compress,omitempty" protobuf:"varint,6,opt,name=compress"`
	// FlushFrequency refers to the frequency in milliseconds to flush batches.
	// Defaults to 500 milliseconds.
	// +optional
	FlushFrequency int32 `json:"flushFrequency,omitempty" protobuf:"varint,7,opt,name=flushFrequency"`
	// TLS configuration for the Kafka producer.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,8,opt,name=tls"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,9,rep,name=payload"`
	// The partitioning key for the messages put on the Kafka topic.
	// +optional.
	PartitioningKey *string `json:"partitioningKey,omitempty" protobuf:"bytes,10,opt,name=partitioningKey"`
	// Specify what kafka version is being connected to enables certain features in sarama, defaults to 1.0.0
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,11,opt,name=version"`
	// SASL configuration for the kafka client
	// +optional
	SASL *SASLConfig `json:"sasl,omitempty" protobuf:"bytes,12,opt,name=sasl"`
	// Schema Registry configuration to producer message with avro format
	// +optional
	SchemaRegistry *SchemaRegistryConfig `json:"schemaRegistry,omitempty" protobuf:"bytes,13,opt,name=schemaRegistry"`
	// Headers for the Kafka Messages.
	// +optional
	Headers map[string]string `json:"headers,omitempty" protobuf:"bytes,14,rep,name=headers"`
	// Secure Headers stored in Kubernetes Secrets for the Kafka messages.
	// +optional
	SecureHeaders []*SecureHeader `json:"secureHeaders,omitempty" protobuf:"bytes,15,rep,name=secureHeaders"`
}

// SchemaRegistryConfig refers to configuration for a client
type SchemaRegistryConfig struct {
	// Schema Registry URL.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Schema ID
	// +optional
	SchemaID int32 `json:"schemaId" protobuf:"varint,2,name=schemaId"`
	// SchemaRegistry - basic authentication
	// +optional
	Auth BasicAuth `json:"auth,omitempty" protobuf:"bytes,3,opt,name=auth"`
}

// PulsarTrigger refers to the specification of the Pulsar trigger.
type PulsarTrigger struct {
	// Configure the service URL for the Pulsar service.
	// +required
	URL string `json:"url" protobuf:"bytes,1,name=url"`
	// Name of the topic.
	// See https://pulsar.apache.org/docs/en/concepts-messaging/
	Topic string `json:"topic" protobuf:"bytes,2,name=topic"`
	// Parameters is the list of parameters that is applied to resolved Kafka trigger object.
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,4,rep,name=payload"`
	// Trusted TLS certificate secret.
	// +optional
	TLSTrustCertsSecret *corev1.SecretKeySelector `json:"tlsTrustCertsSecret,omitempty" protobuf:"bytes,5,opt,name=tlsTrustCertsSecret"`
	// Whether the Pulsar client accept untrusted TLS certificate from broker.
	// +optional
	TLSAllowInsecureConnection bool `json:"tlsAllowInsecureConnection,omitempty" protobuf:"bytes,6,opt,name=tlsAllowInsecureConnection"`
	// Whether the Pulsar client verify the validity of the host name from broker.
	// +optional
	TLSValidateHostname bool `json:"tlsValidateHostname,omitempty" protobuf:"bytes,7,opt,name=tlsValidateHostname"`
	// TLS configuration for the pulsar client.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,8,opt,name=tls"`
	// Authentication token for the pulsar client.
	// Either token or athenz can be set to use auth.
	// +optional
	AuthTokenSecret *corev1.SecretKeySelector `json:"authTokenSecret,omitempty" protobuf:"bytes,9,opt,name=authTokenSecret"`
	// Backoff holds parameters applied to connection.
	// +optional
	ConnectionBackoff *Backoff `json:"connectionBackoff,omitempty" protobuf:"bytes,10,opt,name=connectionBackoff"`
	// Authentication athenz parameters for the pulsar client.
	// Refer https://github.com/apache/pulsar-client-go/blob/master/pulsar/auth/athenz.go
	// Either token or athenz can be set to use auth.
	// +optional
	AuthAthenzParams map[string]string `json:"authAthenzParams,omitempty" protobuf:"bytes,11,rep,name=authAthenzParams"`
	// Authentication athenz privateKey secret for the pulsar client.
	// AuthAthenzSecret must be set if AuthAthenzParams is used.
	// +optional
	AuthAthenzSecret *corev1.SecretKeySelector `json:"authAthenzSecret,omitempty" protobuf:"bytes,12,opt,name=authAthenzSecret"`
}

// NATSTrigger refers to the specification of the NATS trigger.
type NATSTrigger struct {
	// URL of the NATS cluster.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Name of the subject to put message on.
	Subject string `json:"subject" protobuf:"bytes,2,opt,name=subject"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.

	Payload []TriggerParameter `json:"payload" protobuf:"bytes,3,rep,name=payload"`
	// Parameters is the list of parameters that is applied to resolved NATS trigger object.

	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,4,rep,name=parameters"`
	// TLS configuration for the NATS producer.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
	// AuthInformation
	// +optional
	Auth *NATSAuth `json:"auth,omitempty" protobuf:"bytes,6,opt,name=auth"`
}

// CustomTrigger refers to the specification of the custom trigger.
type CustomTrigger struct {
	// ServerURL is the url of the gRPC server that executes custom trigger
	ServerURL string `json:"serverURL" protobuf:"bytes,1,opt,name=serverURL"`
	// Secure refers to type of the connection between sensor to custom trigger gRPC
	Secure bool `json:"secure" protobuf:"varint,2,opt,name=secure"`
	// CertSecret refers to the secret that contains cert for secure connection between sensor and custom trigger gRPC server.
	CertSecret *corev1.SecretKeySelector `json:"certSecret,omitempty" protobuf:"bytes,3,opt,name=certSecret"`
	// ServerNameOverride for the secure connection between sensor and custom trigger gRPC server.
	ServerNameOverride string `json:"serverNameOverride,omitempty" protobuf:"bytes,4,opt,name=serverNameOverride"`
	// Spec is the custom trigger resource specification that custom trigger gRPC server knows how to interpret.
	Spec map[string]string `json:"spec" protobuf:"bytes,5,rep,name=spec"`
	// Parameters is the list of parameters that is applied to resolved custom trigger trigger object.
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,6,rep,name=parameters"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,7,rep,name=payload"`
}

// EmailTrigger refers to the specification of the email notification trigger.
type EmailTrigger struct {
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,1,rep,name=parameters"`
	// Username refers to the username used to connect to the smtp server.
	// +optional
	Username string `json:"username,omitempty" protobuf:"bytes,2,opt,name=username"`
	// SMTPPassword refers to the Kubernetes secret that holds the smtp password used to connect to smtp server.
	// +optional
	SMTPPassword *corev1.SecretKeySelector `json:"smtpPassword,omitempty" protobuf:"bytes,3,opt,name=smtpPassword"`
	// Host refers to the smtp host url to which email is send.
	Host string `json:"host,omitempty" protobuf:"bytes,4,opt,name=host"`
	// Port refers to the smtp server port to which email is send.
	// Defaults to 0.
	// +optional
	Port int32 `json:"port,omitempty" protobuf:"varint,5,opt,name=port"`
	// To refers to the email addresses to which the emails are send.
	// +optional
	To []string `json:"to,omitempty" protobuf:"bytes,6,rep,name=to"`
	// From refers to the address from which the email is send from.
	// +optional
	From string `json:"from,omitempty" protobuf:"bytes,7,opt,name=from"`
	// Subject refers to the subject line for the email send.
	// +optional
	Subject string `json:"subject,omitempty" protobuf:"bytes,8,opt,name=subject"`
	// Body refers to the body/content of the email send.
	// +optional
	Body string `json:"body,omitempty" protobuf:"bytes,9,opt,name=body"`
}

// SlackTrigger refers to the specification of the slack notification trigger.
type SlackTrigger struct {
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,1,rep,name=parameters"`
	// SlackToken refers to the Kubernetes secret that holds the slack token required to send messages.
	SlackToken *corev1.SecretKeySelector `json:"slackToken,omitempty" protobuf:"bytes,2,opt,name=slackToken"`
	// Channel refers to which Slack channel to send Slack message.
	// +optional
	Channel string `json:"channel,omitempty" protobuf:"bytes,3,opt,name=channel"`
	// Message refers to the message to send to the Slack channel.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	// Attachments is a JSON format string that represents an array of Slack attachments according to the attachments API: https://api.slack.com/reference/messaging/attachments .
	// +optional
	Attachments string `json:"attachments,omitempty" protobuf:"bytes,5,opt,name=attachments"`
	// Blocks is a JSON format string that represents an array of Slack blocks according to the blocks API: https://api.slack.com/reference/block-kit/blocks .
	// +optional
	Blocks string `json:"blocks,omitempty" protobuf:"bytes,6,opt,name=blocks"`
	// Thread refers to additional options for sending messages to a Slack thread.
	// +optional
	Thread SlackThread `json:"thread,omitempty" protobuf:"bytes,7,opt,name=thread"`
	// Sender refers to additional configuration of the Slack application that sends the message.
	// +optional
	Sender SlackSender `json:"sender,omitempty" protobuf:"bytes,8,opt,name=sender"`
}

type SlackSender struct {
	// Username is the Slack application's username
	// +optional
	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	// Icon is the Slack application's icon, e.g. :robot_face: or https://example.com/image.png
	// +optional
	Icon string `json:"icon,omitempty" protobuf:"bytes,2,opt,name=icon"`
}

type SlackThread struct {
	// MessageAggregationKey allows to aggregate the messages to a thread by some key.
	// +optional
	MessageAggregationKey string `json:"messageAggregationKey,omitempty" protobuf:"bytes,1,opt,name=messageAggregationKey"`
	// BroadcastMessageToChannel allows to also broadcast the message from the thread to the channel
	// +optional
	BroadcastMessageToChannel bool `json:"broadcastMessageToChannel,omitempty" protobuf:"bytes,2,opt,name=broadcastMessageToChannel"`
}

// OpenWhiskTrigger refers to the specification of the OpenWhisk trigger.
type OpenWhiskTrigger struct {
	// Host URL of the OpenWhisk.
	Host string `json:"host" protobuf:"bytes,1,opt,name=host"`
	// Version for the API.
	// Defaults to v1.
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	// Namespace for the action.
	// Defaults to "_".
	// +optional.
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	// AuthToken for authentication.
	// +optional
	AuthToken *corev1.SecretKeySelector `json:"authToken,omitempty" protobuf:"bytes,4,opt,name=authToken"`
	// Name of the action/function.
	ActionName string `json:"actionName" protobuf:"bytes,5,opt,name=actionName"`
	// Payload is the list of key-value extracted from an event payload to construct the request payload.
	Payload []TriggerParameter `json:"payload" protobuf:"bytes,6,rep,name=payload"`
	// Parameters is the list of key-value extracted from event's payload that are applied to
	// the trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,7,rep,name=parameters"`
}

type LogTrigger struct {
	// Only print messages every interval. Useful to prevent logging too much data for busy events.
	// +optional
	IntervalSeconds uint64 `json:"intervalSeconds,omitempty" protobuf:"varint,1,opt,name=intervalSeconds"`
}

func (in *LogTrigger) GetInterval() time.Duration {
	return time.Duration(in.IntervalSeconds) * time.Second
}

// TriggerParameterOperation represents how to set a trigger destination
// resource key
type TriggerParameterOperation string

const (
	// TriggerParameterOpNone is the zero value of TriggerParameterOperation
	TriggerParameterOpNone TriggerParameterOperation = ""
	// TriggerParameterOpAppend means append the new value to the existing
	TriggerParameterOpAppend TriggerParameterOperation = "append"
	// TriggerParameterOpOverwrite means overwrite the existing value with the new
	TriggerParameterOpOverwrite TriggerParameterOperation = "overwrite"
	// TriggerParameterOpPrepend means prepend the new value to the existing
	TriggerParameterOpPrepend TriggerParameterOperation = "prepend"
)

// TriggerParameter indicates a passed parameter to a service template
type TriggerParameter struct {
	// Src contains a source reference to the value of the parameter from a dependency
	Src *TriggerParameterSource `json:"src,omitempty" protobuf:"bytes,1,opt,name=src"`
	// Dest is the JSONPath of a resource key.
	// A path is a series of keys separated by a dot. The colon character can be escaped with '.'
	// The -1 key can be used to append a value to an existing array.
	// See https://github.com/tidwall/sjson#path-syntax for more information about how this is used.
	Dest string `json:"dest" protobuf:"bytes,2,opt,name=dest"`
	// Operation is what to do with the existing value at Dest, whether to
	// 'prepend', 'overwrite', or 'append' it.
	Operation TriggerParameterOperation `json:"operation,omitempty" protobuf:"bytes,3,opt,name=operation,casttype=TriggerParameterOperation"`
}

// TriggerParameterSource defines the source for a parameter from a event event
type TriggerParameterSource struct {
	// DependencyName refers to the name of the dependency. The event which is stored for this dependency is used as payload
	// for the parameterization. Make sure to refer to one of the dependencies you have defined under Dependencies list.
	DependencyName string `json:"dependencyName" protobuf:"bytes,1,opt,name=dependencyName"`
	// ContextKey is the JSONPath of the event's (JSON decoded) context key
	// ContextKey is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	ContextKey string `json:"contextKey,omitempty" protobuf:"bytes,2,opt,name=contextKey"`
	// ContextTemplate is a go-template for extracting a string from the event's context.
	// If a ContextTemplate is provided with a ContextKey, the template will be evaluated first and fallback to the ContextKey.
	// The templating follows the standard go-template syntax as well as sprig's extra functions.
	// See https://pkg.go.dev/text/template and https://masterminds.github.io/sprig/
	ContextTemplate string `json:"contextTemplate,omitempty" protobuf:"bytes,3,opt,name=contextTemplate"`
	// DataKey is the JSONPath of the event's (JSON decoded) data key
	// DataKey is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'.
	// To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\'.
	// See https://github.com/tidwall/gjson#path-syntax for more information on how to use this.
	DataKey string `json:"dataKey,omitempty" protobuf:"bytes,4,opt,name=dataKey"`
	// DataTemplate is a go-template for extracting a string from the event's data.
	// If a DataTemplate is provided with a DataKey, the template will be evaluated first and fallback to the DataKey.
	// The templating follows the standard go-template syntax as well as sprig's extra functions.
	// See https://pkg.go.dev/text/template and https://masterminds.github.io/sprig/
	DataTemplate string `json:"dataTemplate,omitempty" protobuf:"bytes,5,opt,name=dataTemplate"`
	// Value is the default literal value to use for this parameter source
	// This is only used if the DataKey is invalid.
	// If the DataKey is invalid and this is not defined, this param source will produce an error.
	Value *string `json:"value,omitempty" protobuf:"bytes,6,opt,name=value"`
	// UseRawData indicates if the value in an event at data key should be used without converting to string.
	// When true, a number, boolean, json or string parameter may be extracted. When the field is unspecified, or explicitly
	// false, the behavior is to turn the extracted field into a string. (e.g. when set to true, the parameter
	// 123 will resolve to the numerical type, but when false, or not provided, the string "123" will be resolved)
	// +optional
	UseRawData bool `json:"useRawData,omitempty" protobuf:"bytes,7,opt,name=useRawData"`
}

// TriggerPolicy dictates the policy for the trigger retries
type TriggerPolicy struct {
	// K8SResourcePolicy refers to the policy used to check the state of K8s based triggers using using labels
	K8s *K8SResourcePolicy `json:"k8s,omitempty" protobuf:"bytes,1,opt,name=k8s"`
	// Status refers to the policy used to check the state of the trigger using response status
	Status *StatusPolicy `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

// K8SResourcePolicy refers to the policy used to check the state of K8s based triggers using labels
type K8SResourcePolicy struct {
	// Labels required to identify whether a resource is in success state
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,1,rep,name=labels"`
	// Backoff before checking resource state
	Backoff *Backoff `json:"backoff" protobuf:"bytes,2,opt,name=backoff"`
	// ErrorOnBackoffTimeout determines whether sensor should transition to error state if the trigger policy is unable to determine
	// the state of the resource
	ErrorOnBackoffTimeout bool `json:"errorOnBackoffTimeout" protobuf:"varint,3,opt,name=errorOnBackoffTimeout"`
}

// StatusPolicy refers to the policy used to check the state of the trigger using response status
type StatusPolicy struct {
	// Allow refers to the list of allowed response statuses. If the response status of the trigger is within the list,
	// the trigger will marked as successful else it will result in trigger failure.

	Allow []int32 `json:"allow" protobuf:"varint,1,rep,name=allow"`
}

func (in *StatusPolicy) GetAllow() []int {
	statuses := make([]int, len(in.Allow))
	for i, s := range in.Allow {
		statuses[i] = int(s)
	}
	return statuses
}

// SensorStatus contains information about the status of a sensor.
type SensorStatus struct {
	Status `json:",inline" protobuf:"bytes,1,opt,name=status"`
}

const (
	// SensorConditionDepencencyProvided has the status True when the
	// Sensor has valid dependencies provided.
	SensorConditionDepencencyProvided ConditionType = "DependenciesProvided"
	// SensorConditionTriggersProvided has the status True when the
	// Sensor has valid triggers provided.
	SensorConditionTriggersProvided ConditionType = "TriggersProvided"
	// SensorConditionDeployed has the status True when the Sensor
	// has its Deployment created.
	SensorConditionDeployed ConditionType = "Deployed"
)

// InitConditions sets conditions to Unknown state.
func (s *SensorStatus) InitConditions() {
	s.InitializeConditions(SensorConditionDepencencyProvided, SensorConditionTriggersProvided, SensorConditionDeployed)
}

// MarkDependenciesProvided set the sensor has valid dependencies provided.
func (s *SensorStatus) MarkDependenciesProvided() {
	s.MarkTrue(SensorConditionDepencencyProvided)
}

// MarkDependenciesNotProvided set the sensor has invalid dependencies provided.
func (s *SensorStatus) MarkDependenciesNotProvided(reason, message string) {
	s.MarkFalse(SensorConditionDepencencyProvided, reason, message)
}

// MarkTriggersProvided set the sensor has valid triggers provided.
func (s *SensorStatus) MarkTriggersProvided() {
	s.MarkTrue(SensorConditionTriggersProvided)
}

// MarkTriggersNotProvided set the sensor has invalid triggers provided.
func (s *SensorStatus) MarkTriggersNotProvided(reason, message string) {
	s.MarkFalse(SensorConditionTriggersProvided, reason, message)
}

// MarkDeployed set the sensor has been deployed.
func (s *SensorStatus) MarkDeployed() {
	s.MarkTrue(SensorConditionDeployed)
}

// MarkDeployFailed set the sensor deploy failed
func (s *SensorStatus) MarkDeployFailed(reason, message string) {
	s.MarkFalse(SensorConditionDeployed, reason, message)
}

// ArtifactLocation describes the source location for an external artifact
type ArtifactLocation struct {
	// S3 compliant artifact
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`
	// Inline artifact is embedded in sensor spec as a string
	Inline *string `json:"inline,omitempty" protobuf:"bytes,2,opt,name=inline"`
	// File artifact is artifact stored in a file
	File *FileArtifact `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`
	// URL to fetch the artifact from
	URL *URLArtifact `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`
	// Configmap that stores the artifact
	Configmap *corev1.ConfigMapKeySelector `json:"configmap,omitempty" protobuf:"bytes,5,opt,name=configmap"`
	// Git repository hosting the artifact
	Git *GitArtifact `json:"git,omitempty" protobuf:"bytes,6,opt,name=git"`
	// Resource is generic template for K8s resource
	Resource *K8SResource `json:"resource,omitempty" protobuf:"bytes,7,opt,name=resource"`
}

// FileArtifact contains information about an artifact in a filesystem
type FileArtifact struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

// URLArtifact contains information about an artifact at an http endpoint.
type URLArtifact struct {
	// Path is the complete URL
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`
	// VerifyCert decides whether the connection is secure or not
	VerifyCert bool `json:"verifyCert,omitempty" protobuf:"varint,2,opt,name=verifyCert"`
}

// GitArtifact contains information about an artifact stored in git
type GitArtifact struct {
	// Git URL
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Directory to clone the repository. We clone complete directory because GitArtifact is not limited to any specific Git service providers.
	// Hence we don't use any specific git provider client.
	CloneDirectory string `json:"cloneDirectory" protobuf:"bytes,2,opt,name=cloneDirectory"`
	// Creds contain reference to git username and password
	// +optional
	Creds *GitCreds `json:"creds,omitempty" protobuf:"bytes,3,opt,name=creds"`
	// SSHKeySecret refers to the secret that contains SSH key
	SSHKeySecret *corev1.SecretKeySelector `json:"sshKeySecret,omitempty" protobuf:"bytes,4,opt,name=sshKeySecret"`
	// Path to file that contains trigger resource definition
	FilePath string `json:"filePath" protobuf:"bytes,5,opt,name=filePath"`
	// Branch to use to pull trigger resource
	// +optional
	Branch string `json:"branch,omitempty" protobuf:"bytes,6,opt,name=branch"`
	// Tag to use to pull trigger resource
	// +optional
	Tag string `json:"tag,omitempty" protobuf:"bytes,7,opt,name=tag"`
	// Ref to use to pull trigger resource. Will result in a shallow clone and
	// fetch.
	// +optional
	Ref string `json:"ref,omitempty" protobuf:"bytes,8,opt,name=ref"`
	// Remote to manage set of tracked repositories. Defaults to "origin".
	// Refer https://git-scm.com/docs/git-remote
	// +optional
	Remote *GitRemoteConfig `json:"remote,omitempty" protobuf:"bytes,9,opt,name=remote"`
	// Whether to ignore host key
	// +optional
	InsecureIgnoreHostKey bool `json:"insecureIgnoreHostKey,omitempty" protobuf:"bytes,10,opt,name=insecureIgnoreHostKey"`
}

// GitRemoteConfig contains the configuration of a Git remote
type GitRemoteConfig struct {
	// Name of the remote to fetch from.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// URLs the URLs of a remote repository. It must be non-empty. Fetch will
	// always use the first URL, while push will use all of them.
	URLS []string `json:"urls" protobuf:"bytes,2,rep,name=urls"`
}

// GitCreds contain reference to git username and password
type GitCreds struct {
	Username *corev1.SecretKeySelector `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`
}

// Event represents the cloudevent received from an event source.
// +protobuf.options.(gogoproto.goproto_stringer)=false
type Event struct {
	Context *EventContext `json:"context,omitempty" protobuf:"bytes,1,opt,name=context"`
	Data    []byte        `json:"data" protobuf:"bytes,2,opt,name=data"`
}

// returns a string representation of the data, either as the text (e.g. if it is text) or as base 64 encoded string
func (e Event) DataString() string {
	if e.Data == nil {
		return ""
	}
	mediaType := e.getMediaType()
	switch mediaType {
	case "application/json", "text/plain":
		return string(e.Data)
	default:
		return base64.StdEncoding.EncodeToString(e.Data)
	}
}

func (e Event) getMediaType() string {
	dataContentType := ""
	if e.Context != nil {
		dataContentType = e.Context.DataContentType
	}
	mediaType, _, _ := mime.ParseMediaType(dataContentType)
	return mediaType
}

func (e Event) String() string {
	return fmt.Sprintf(`{"context:" "%v", "data": "%v"}`, e.Context, e.DataString())
}

// EventContext holds the context of the cloudevent received from an event source.
// +protobuf.options.(gogoproto.goproto_stringer)=false
type EventContext struct {
	// ID of the event; must be non-empty and unique within the scope of the producer.
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`
	// Source - A URI describing the event producer.
	Source string `json:"source" protobuf:"bytes,2,opt,name=source"`
	// SpecVersion - The version of the CloudEvents specification used by the event.
	SpecVersion string `json:"specversion" protobuf:"bytes,3,opt,name=specversion"`
	// Type - The type of the occurrence which has happened.
	Type string `json:"type" protobuf:"bytes,4,opt,name=type"`
	// DataContentType - A MIME (RFC2046) string describing the media type of `data`.
	DataContentType string `json:"datacontenttype" protobuf:"bytes,5,opt,name=datacontenttype"`
	// Subject - The subject of the event in the context of the event producer
	Subject string `json:"subject" protobuf:"bytes,6,opt,name=subject"`
	// Time - A Timestamp when the event happened.
	Time metav1.Time `json:"time" protobuf:"bytes,7,opt,name=time"`
}

func (e EventContext) String() string {
	return fmt.Sprintf(`{"id": "%s", "source": "%s", "specversion": "%s", "type": "%s", "datacontenttype": "%s", "subject": "%s", "time": "%s"}`, e.ID, e.Source, e.SpecVersion, e.Type, e.DataContentType, e.Subject, e.Time)
}

// HasLocation whether or not an artifact has a location defined
func (a *ArtifactLocation) HasLocation() bool {
	return a.S3 != nil || a.Inline != nil || a.File != nil || a.URL != nil
}
