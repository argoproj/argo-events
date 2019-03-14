# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto](#github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1/generated.proto)
    - [ArtifactLocation](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ArtifactLocation)
    - [ConfigmapArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ConfigmapArtifact)
    - [DataFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter)
    - [DependencyGroup](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DependencyGroup)
    - [EventDependency](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency)
    - [EventDependencyFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependencyFilter)
    - [FileArtifact](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.FileArtifact)
    - [GroupVersionKind](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.GroupVersionKind)
    - [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.NodeStatus)
    - [ResourceObject](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject)
    - [ResourceObject.LabelsEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject.LabelsEntry)
    - [ResourceParameter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameter)
    - [ResourceParameterSource](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource)
    - [RetryStrategy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy)
    - [Sensor](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor)
    - [SensorList](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorList)
    - [SensorSpec](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec)
    - [SensorStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus)
    - [SensorStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry)
    - [TimeFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter)
    - [Trigger](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger)
    - [TriggerCondition](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TriggerCondition)
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
| s3 | [github.com.argoproj.argo_events.pkg.apis.common.S3Artifact](#github.com.argoproj.argo_events.pkg.apis.common.S3Artifact) | optional |  |
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






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter"></a>

### DataFilter
DataFilter describes constraints and filters for event data
Regular Expressions are purposefully not a feature as they are overkill for our uses here
See Rob Pike&#39;s Post: https://commandcenter.blogspot.com/2011/08/regular-expressions-in-lexing-and.html


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. |
| type | [string](#string) | optional | Type contains the JSON type of the data |
| value | [string](#string) | repeated | Value is the allowed string values for this key Booleans are pased using strconv.ParseBool() Numbers are parsed using as float64 using strconv.ParseFloat() Strings are taken as is Nils this value is ignored |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DependencyGroup"></a>

### DependencyGroup
DependencyGroup is the group of dependencies


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the group |
| dependencies | [string](#string) | repeated | Dependencies of events |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency"></a>

### EventDependency
EventDependency describes a dependency


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a unique name of this dependency |
| deadline | [int64](#int64) | optional | Deadline is the duration in seconds after the StartedAt time of the sensor after which this event is terminated. Note: this functionality is not yet respected, but it&#39;s theoretical behavior is as follows: This trumps the recurrence patterns of calendar events and allows any event to have a strict defined life. After the deadline is reached and this event has not in a Resolved state, this event is marked as Failed and proper escalations should proceed. |
| filters | [EventDependencyFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependencyFilter) | optional | Filters and rules governing tolerations of success and constraints on the context and data of an event |
| connected | [bool](#bool) | optional | Connected tells if subscription is already setup in case of nats protocol. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependencyFilter"></a>

### EventDependencyFilter
EventDependencyFilter defines filters and constraints for a event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of event filter |
| time | [TimeFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter) | optional | Time filter on the event with escalation |
| context | [github.com.argoproj.argo_events.pkg.apis.common.EventContext](#github.com.argoproj.argo_events.pkg.apis.common.EventContext) | optional | Context filter constraints with escalation |
| data | [DataFilter](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DataFilter) | repeated | Data filter constraints with escalation |






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
A single node can represent the status for event or a trigger.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is a unique identifier of a node within a sensor It is a hash of the node name |
| name | [string](#string) | optional | Name is a unique name in the node tree used to generate the node ID |
| displayName | [string](#string) | optional | DisplayName is the human readable representation of the node |
| type | [string](#string) | optional | Type is the type of the node |
| phase | [string](#string) | optional | Phase of the node |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | StartedAt is the time at which this node started |
| completedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | CompletedAt is the time at which this node completed |
| message | [string](#string) | optional | store data or something to save for event notifications or trigger events |
| event | [github.com.argoproj.argo_events.pkg.apis.common.Event](#github.com.argoproj.argo_events.pkg.apis.common.Event) | optional | Event stores the last seen event for this node |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject"></a>

### ResourceObject
ResourceObject is the resource object to create on kubernetes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groupVersionKind | [GroupVersionKind](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.GroupVersionKind) | optional | The unambiguous kind of this object - used in order to retrieve the appropriate kubernetes api client for this resource |
| namespace | [string](#string) | optional | Namespace in which to create this object defaults to the service account namespace |
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
| src | [ResourceParameterSource](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource) | optional | Src contains a source reference to the value of the resource parameter from a event event |
| dest | [string](#string) | optional | Dest is the JSONPath of a resource key. A path is a series of keys separated by a dot. The colon character can be escaped with &#39;.&#39; The -1 key can be used to append a value to an existing array. See https://github.com/tidwall/sjson#path-syntax for more information about how this is used. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceParameterSource"></a>

### ResourceParameterSource
ResourceParameterSource defines the source for a resource parameter from a event event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event | [string](#string) | optional | Event is the name of the event for which to retrieve this event |
| path | [string](#string) | optional | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. |
| value | [string](#string) | optional | Value is the default literal value to use for this parameter source This is only used if the path is invalid. If the path is invalid and this is not defined, this param source will produce an error. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy"></a>

### RetryStrategy
RetryStrategy represents a strategy for retrying operations
TODO: implement me






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor"></a>

### Sensor
Sensor is the definition of a sensor resource
&#43;genclient
&#43;genclient:noStatus
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
&#43;k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta) | optional |  |
| spec | [SensorSpec](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec) | optional |  |
| status | [SensorStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorList"></a>

### SensorList
SensorList is the list of Sensor resources
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta) | optional |  |
| items | [Sensor](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Sensor) | repeated |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorSpec"></a>

### SensorSpec
SensorSpec represents desired sensor state


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dependencies | [EventDependency](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.EventDependency) | repeated | Dependencies is a list of the events that this sensor is dependent on. |
| triggers | [Trigger](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger) | repeated | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. |
| deploySpec | [k8s.io.api.core.v1.PodSpec](#k8s.io.api.core.v1.PodSpec) | optional | Template contains sensor pod specification. For more information, read https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#pod-v1-core |
| eventProtocol | [github.com.argoproj.argo_events.pkg.apis.common.EventProtocol](#github.com.argoproj.argo_events.pkg.apis.common.EventProtocol) | optional | EventProtocol is the protocol through which sensor receives events from gateway |
| circuit | [string](#string) | optional | Circuit is a boolean expression of dependency groups |
| dependencyGroups | [DependencyGroup](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.DependencyGroup) | repeated | DependencyGroups is a list of the groups of events. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus"></a>

### SensorStatus
SensorStatus contains information about the status of a sensor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | Phase is the high-level summary of the sensor |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s.io.apimachinery.pkg.apis.meta.v1.Time) | optional | StartedAt is the time at which this sensor was initiated |
| completedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s.io.apimachinery.pkg.apis.meta.v1.Time) | optional | CompletedAt is the time at which this sensor was completed |
| completionCount | [int32](#int32) | optional | CompletionCount is the count of sensor&#39;s successful runs. |
| message | [string](#string) | optional | Message is a human readable string indicating details about a sensor in its phase |
| nodes | [SensorStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry) | repeated | Nodes is a mapping between a node ID and the node&#39;s status it records the states for the FSM of this sensor. |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.SensorStatus.NodesEntry"></a>

### SensorStatus.NodesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.NodeStatus) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TimeFilter"></a>

### TimeFilter
TimeFilter describes a window in time.
DataFilters out event events that occur outside the time limits.
In other words, only events that occur after Start and before Stop
will pass this filter.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [string](#string) | optional | Start is the beginning of a time window. Before this time, events for this event are ignored and format is hh:mm:ss |
| stop | [string](#string) | optional | StopPattern is the end of a time window. After this time, events for this event are ignored and format is hh:mm:ss |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.Trigger"></a>

### Trigger
Trigger is an action taken, output produced, an event created, a message sent


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a unique name of the action to take |
| resource | [ResourceObject](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.ResourceObject) | optional | Resource describes the resource that will be created by this action |
| message | [string](#string) | optional | Message describes a message that will be sent on a queue |
| replyStrategy | [RetryStrategy](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.RetryStrategy) | optional | RetryStrategy is the strategy to retry a trigger if it fails |
| when | [TriggerCondition](#github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TriggerCondition) | optional | When is the condition to execute the trigger |






<a name="github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1.TriggerCondition"></a>

### TriggerCondition
TriggerCondition describes condition which must be satisfied in order to execute a trigger.
Depending upon condition type, status of dependency groups is used to evaluate the result.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| any | [string](#string) | repeated | Any acts as a OR operator between dependencies |
| all | [string](#string) | repeated | All acts as a AND operator between dependencies |






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

