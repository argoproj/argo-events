# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto](#github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto)
    - [ArtifactLocation](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ArtifactLocation)
    - [ConfigmapArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ConfigmapArtifact)
    - [Data](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Data)
    - [DataFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter)
    - [EscalationPolicy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy)
    - [Event](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Event)
    - [EventContext](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext)
    - [EventContext.ExtensionsEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext.ExtensionsEntry)
    - [EventDependency](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency)
    - [FileArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.FileArtifact)
    - [GroupVersionKind](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.GroupVersionKind)
    - [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.NodeStatus)
    - [ResourceObject](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject)
    - [ResourceObject.LabelsEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject.LabelsEntry)
    - [ResourceParameter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameter)
    - [ResourceParameterSource](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource)
    - [RetryStrategy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy)
    - [S3Artifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Artifact)
    - [S3Bucket](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Bucket)
    - [S3Filter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Filter)
    - [Sensor](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor)
    - [SensorList](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorList)
    - [SensorSpec](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec)
    - [SensorStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus)
    - [SensorStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry)
    - [SignalFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SignalFilter)
    - [TimeFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter)
    - [Trigger](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger)
    - [URI](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URI)
    - [URLArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URLArtifact)
  
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto



<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ArtifactLocation"></a>

### ArtifactLocation
ArtifactLocation describes the source location for an external artifact


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| s3 | [S3Artifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Artifact) | optional |  |
| inline | [string](#string) | optional |  |
| file | [FileArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.FileArtifact) | optional |  |
| url | [URLArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URLArtifact) | optional |  |
| configmap | [ConfigmapArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ConfigmapArtifact) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ConfigmapArtifact"></a>

### ConfigmapArtifact
ConfigmapArtifact contains information about artifact in k8 configmap


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the configmap |
| namespace | [string](#string) | optional | Namespace where configmap is deployed |
| key | [string](#string) | optional | Key within configmap data which contains trigger resource definition |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Data"></a>

### Data



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filters | [DataFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter) | repeated | filter constraints |
| escalationPolicy | [EscalationPolicy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy) | optional | EscalationPolicy is the escalation to trigger in case the signal filter fails |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter"></a>

### DataFilter
DataFilter describes constraints and filters for event data
Regular Expressions are purposefully not a feature as they are overkill for our uses here
See Rob Pike&#39;s Post: https://commandcenter.blogspot.com/2011/08/regular-expressions-in-lexing-and.html


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. |
| type | [string](#string) | optional | Type contains the JSON type of the data |
| value | [string](#string) | optional | Value is the expected string value for this key Booleans are pased using strconv.ParseBool() Numbers are parsed using as float64 using strconv.ParseFloat() Strings are taken as is Nils this value is ignored |
| escalationPolicy | [EscalationPolicy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy) | optional | EscalationPolicy is the escalation to trigger in case the signal filter fails |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy"></a>

### EscalationPolicy
EscalationPolicy describes the policy for escalating sensors in an Error state.
An escalation policy is associated with signal filter. Whenever a signal filter fails
escalation will be triggered


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is name of the escalation policy This is referred by signal filter/s |
| level | [string](#string) | optional | Level is the degree of importance |
| message | [string](#string) | optional | need someway to progressively get more serious notifications |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Event"></a>

### Event
Event is a data record expressing an occurrence and its context.
Adheres to the CloudEvents v0.1 specification


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [EventContext](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext) | optional |  |
| data | [bytes](#bytes) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext"></a>

### EventContext
EventContext contains metadata that provides circumstantial information about the occurrence.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| eventType | [string](#string) | optional | The type of occurrence which has happened. Often this attribute is used for routing, observability, policy enforcement, etc. should be prefixed with a reverse-DNS name. The prefixed domain dictates the organization which defines the semantics of this event type. ex: com.github.pull.create |
| eventTypeVersion | [string](#string) | optional | The version of the eventType. Enables the interpretation of data by eventual consumers, requires the consumer to be knowledgeable about the producer. |
| cloudEventsVersion | [string](#string) | optional | The version of the CloudEvents specification which the event uses. Enables the interpretation of the context. |
| source | [URI](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URI) | optional | This describes the event producer. |
| eventID | [string](#string) | optional | ID of the event. The semantics are explicitly undefined to ease the implementation of producers. Enables deduplication. Must be unique within scope of producer. |
| eventTime | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | Timestamp of when the event happened. Must adhere to format specified in RFC 3339. &#43;k8s:openapi-gen=false |
| schemaURL | [URI](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URI) | optional | A link to the schema that the data attribute adheres to. Must adhere to the format specified in RFC 3986. |
| contentType | [string](#string) | optional | Content type of the data attribute value. Enables the data attribute to carry any type of content, whereby format and encoding might differ from that of the chosen event format. For example, the data attribute may carry an XML or JSON payload and the consumer is informed by this attribute being set to &#34;application/xml&#34; or &#34;application/json&#34; respectively. |
| extensions | [EventContext.ExtensionsEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext.ExtensionsEntry) | repeated | This is for additional metadata and does not have a mandated structure. Enables a place for custom fields a producer or middleware might want to include and provides a place to test metadata before adding them to the CloudEvents specification. |
| escalationPolicy | [EscalationPolicy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy) | optional | EscalationPolicy is the name of escalation policy to trigger in case the signal filter fails |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext.ExtensionsEntry"></a>

### EventContext.ExtensionsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency"></a>

### EventDependency
EventDependency describes a dependency


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a unique name of this dependency |
| deadline | [int64](#int64) | optional | Deadline is the duration in seconds after the StartedAt time of the sensor after which this signal is terminated. Note: this functionality is not yet respected, but it&#39;s theoretical behavior is as follows: This trumps the recurrence patterns of calendar signals and allows any signal to have a strict defined life. After the deadline is reached and this signal has not in a Resolved state, this signal is marked as Failed and proper escalations should proceed. |
| filters | [SignalFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SignalFilter) | optional | Filters and rules governing tolerations of success and constraints on the context and data of an event |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.FileArtifact"></a>

### FileArtifact
FileArtifact contains information about an artifact in a filesystem


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.GroupVersionKind"></a>

### GroupVersionKind
GroupVersionKind unambiguously identifies a kind.  It doesn&#39;t anonymously include GroupVersion
to avoid automatic coercion.  It doesn&#39;t use a GroupVersion to avoid custom marshalling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |
| kind | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.NodeStatus"></a>

### NodeStatus
NodeStatus describes the status for an individual node in the sensor&#39;s FSM.
A single node can represent the status for signal or a trigger.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is a unique identifier of a node within a sensor It is a hash of the node name |
| name | [string](#string) | optional | Name is a unique name in the node tree used to generate the node ID |
| displayName | [string](#string) | optional | DisplayName is the human readable representation of the node |
| type | [string](#string) | optional | Type is the type of the node |
| phase | [string](#string) | optional | Phase of the node |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | StartedAt is the time at which this node started &#43;k8s:openapi-gen=false |
| completedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | CompletedAt is the time at which this node completed &#43;k8s:openapi-gen=false |
| message | [string](#string) | optional | store data or something to save for signal notifications or trigger events |
| latestEvent | [Event](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Event) | optional | Event stores the last seen event for this node |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject"></a>

### ResourceObject
ResourceObject is the resource object to create on kubernetes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groupVersionKind | [GroupVersionKind](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.GroupVersionKind) | optional | The unambiguous kind of this object - used in order to retrieve the appropriate kubernetes api client for this resource |
| namespace | [string](#string) | optional | Namespace in which to create this object optional defaults to the service account namespace |
| source | [ArtifactLocation](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ArtifactLocation) | optional | Source of the K8 resource file(s) |
| labels | [ResourceObject.LabelsEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject.LabelsEntry) | repeated | Map of string keys and values that can be used to organize and categorize (scope and select) objects. This overrides any labels in the unstructured object with the same key. |
| parameters | [ResourceParameter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameter) | repeated | Parameters is the list of resource parameters to pass in the object |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject.LabelsEntry"></a>

### ResourceObject.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameter"></a>

### ResourceParameter
ResourceParameter indicates a passed parameter to a service template


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| src | [ResourceParameterSource](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource) | optional | Src contains a source reference to the value of the resource parameter from a signal event |
| dest | [string](#string) | optional | Dest is the JSONPath of a resource key. A path is a series of keys separated by a dot. The colon character can be escaped with &#39;.&#39; The -1 key can be used to append a value to an existing array. See https://github.com/tidwall/sjson#path-syntax for more information about how this is used. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource"></a>

### ResourceParameterSource
ResourceParameterSource defines the source for a resource parameter from a signal event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| signal | [string](#string) | optional | EventDependency is the name of the signal for which to retrieve this event |
| path | [string](#string) | optional | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. |
| value | [string](#string) | optional | Value is the default literal value to use for this parameter source This is only used if the path is invalid. If the path is invalid and this is not defined, this param source will produce an error. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy"></a>

### RetryStrategy
RetryStrategy represents a strategy for retrying operations
TODO: implement me






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Artifact"></a>

### S3Artifact
S3Artifact contains information about an artifact in S3


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| s3Bucket | [S3Bucket](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Bucket) | optional |  |
| key | [string](#string) | optional |  |
| event | [string](#string) | optional |  |
| filter | [S3Filter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Filter) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Bucket"></a>

### S3Bucket
S3Bucket contains information for an S3 Bucket


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) | optional |  |
| bucket | [string](#string) | optional |  |
| region | [string](#string) | optional |  |
| insecure | [bool](#bool) | optional |  |
| accessKey | [k8s.io.api.core.v1.SecretKeySelector](#k8s.io.api.core.v1.SecretKeySelector) | optional | &#43;k8s:openapi-gen=false |
| secretKey | [k8s.io.api.core.v1.SecretKeySelector](#k8s.io.api.core.v1.SecretKeySelector) | optional | &#43;k8s:openapi-gen=false |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.S3Filter"></a>

### S3Filter
S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix | [string](#string) | optional |  |
| suffix | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor"></a>

### Sensor
Sensor is the definition of a sensor resource
&#43;genclient
&#43;genclient:noStatus
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
&#43;k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta) | optional | &#43;k8s:openapi-gen=false |
| spec | [SensorSpec](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec) | optional |  |
| status | [SensorStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorList"></a>

### SensorList
SensorList is the list of Sensor resources
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta) | optional | &#43;k8s:openapi-gen=false |
| items | [Sensor](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor) | repeated |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec"></a>

### SensorSpec
SensorSpec represents desired sensor state


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dependencies | [EventDependency](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency) | repeated | EventDependency is a list of the events that this sensor is dependent on. |
| triggers | [Trigger](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger) | repeated | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. |
| deploySpec | [k8s.io.api.core.v1.PodSpec](#k8s.io.api.core.v1.PodSpec) | optional | DeploySpec contains sensor pod specification. For more information, read https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#pod-v1-core |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus"></a>

### SensorStatus
SensorStatus contains information about the status of a sensor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | Phase is the high-level summary of the sensor |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s.io.apimachinery.pkg.apis.meta.v1.Time) | optional | StartedAt is the time at which this sensor was initiated &#43;k8s:openapi-gen=false |
| completedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s.io.apimachinery.pkg.apis.meta.v1.Time) | optional | CompletedAt is the time at which this sensor was completed &#43;k8s:openapi-gen=false |
| completionCount | [int32](#int32) | optional | CompletionCount is the count of sensor&#39;s successful runs. |
| message | [string](#string) | optional | Message is a human readable string indicating details about a sensor in its phase |
| nodes | [SensorStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry) | repeated | Nodes is a mapping between a node ID and the node&#39;s status it records the states for the FSM of this sensor. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry"></a>

### SensorStatus.NodesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.NodeStatus) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SignalFilter"></a>

### SignalFilter
SignalFilter defines filters and constraints for a signal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of signal filter |
| time | [TimeFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter) | optional | Time filter on the signal with escalation |
| context | [EventContext](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventContext) | optional | Context filter constraints with escalation |
| data | [Data](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Data) | optional | Data filter constraints with escalation |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter"></a>

### TimeFilter
TimeFilter describes a window in time.
Filters out signal events that occur outside the time limits.
In other words, only events that occur after Start and before Stop
will pass this filter.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [string](#string) | optional | Start is the beginning of a time window. Before this time, events for this signal are ignored and format is hh:mm:ss |
| stop | [string](#string) | optional | StopPattern is the end of a time window. After this time, events for this signal are ignored and format is hh:mm:ss |
| escalationPolicy | [EscalationPolicy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EscalationPolicy) | optional | EscalationPolicy is the escalation to trigger in case the signal filter fails |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger"></a>

### Trigger
Trigger is an action taken, output produced, an event created, a message sent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a unique name of the action to take |
| resource | [ResourceObject](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject) | optional | Resource describes the resource that will be created by this action |
| message | [string](#string) | optional | Message describes a message that will be sent on a queue |
| replyStrategy | [RetryStrategy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy) | optional | RetryStrategy is the strategy to retry a trigger if it fails |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URI"></a>

### URI
URI is a Uniform Resource Identifier based on RFC 3986


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scheme | [string](#string) | optional |  |
| user | [string](#string) | optional |  |
| password | [string](#string) | optional |  |
| host | [string](#string) | optional |  |
| port | [int32](#int32) | optional |  |
| path | [string](#string) | optional |  |
| query | [string](#string) | optional |  |
| fragment | [string](#string) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.URLArtifact"></a>

### URLArtifact
URLArtifact contains information about an artifact at an http endpoint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |
| verifyCert | [bool](#bool) | optional |  |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

