# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [pkg/apis/gateway/v1alpha1/generated.proto](#pkg/apis/gateway/v1alpha1/generated.proto)
    - [Gateway](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.Gateway)
    - [GatewayList](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayList)
    - [GatewayNotificationWatcher](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayNotificationWatcher)
    - [GatewaySpec](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewaySpec)
    - [GatewayStatus](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus)
    - [GatewayStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus.NodesEntry)
    - [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NodeStatus)
    - [NotificationWatchers](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NotificationWatchers)
    - [SensorNotificationWatcher](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.SensorNotificationWatcher)
  
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="pkg/apis/gateway/v1alpha1/generated.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## pkg/apis/gateway/v1alpha1/generated.proto



<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.Gateway"></a>

### Gateway
Gateway is the definition of a gateway resource
&#43;genclient
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
&#43;k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta) | optional |  |
| status | [GatewayStatus](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus) | optional |  |
| spec | [GatewaySpec](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewaySpec) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayList"></a>

### GatewayList
GatewayList is the list of Gateway resources
&#43;k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta) | optional |  |
| items | [Gateway](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.Gateway) | repeated |  |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayNotificationWatcher"></a>

### GatewayNotificationWatcher
GatewayNotificationWatcher is the gateway interested in listening to notifications from this gateway


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the gateway name |
| port | [string](#string) | optional | Port is http server port on which gateway is running |
| endpoint | [string](#string) | optional | Endpoint is REST API endpoint to post event to. Events are sent using HTTP POST method to this endpoint. |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewaySpec"></a>

### GatewaySpec
GatewaySpec represents gateway specifications


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Template | [k8s.io.api.core.v1.Pod](#k8s.io.api.core.v1.Pod) | optional | Template is the pod specification for the gateway Refer https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#pod-v1-core |
| configmap | [string](#string) | optional | ConfigMap is name of the configmap for gateway. This configmap contains event sources. |
| type | [string](#string) | optional | Type is the type of gateway. Used as metadata. |
| eventVersion | [string](#string) | optional | Version is used for marking event version |
| serviceSpec | [k8s.io.api.core.v1.Service](#k8s.io.api.core.v1.Service) | optional | Service is the specifications of the service to expose the gateway Refer https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#service-v1-core |
| watchers | [NotificationWatchers](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NotificationWatchers) | optional | Watchers are components which are interested listening to notifications from this gateway These only need to be specified when gateway dispatch mechanism is through HTTP POST notifications. In future, support for NATS, KAFKA will be added as a means to dispatch notifications in which case specifying watchers would be unnecessary. |
| processorPort | [string](#string) | optional | Port on which the gateway event source processor is running on. |
| eventProtocol | [github.com.argoproj.argo_events.pkg.apis.common.EventProtocol](#github.com.argoproj.argo_events.pkg.apis.common.EventProtocol) | optional | EventProtocol is the underlying protocol used to send events from gateway to watchers(components interested in listening to event from this gateway) |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus"></a>

### GatewayStatus
GatewayStatus contains information about the status of a gateway.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | Phase is the high-level summary of the gateway |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s.io.apimachinery.pkg.apis.meta.v1.Time) | optional | StartedAt is the time at which this gateway was initiated |
| message | [string](#string) | optional | Message is a human readable string indicating details about a gateway in its phase |
| nodes | [GatewayStatus.NodesEntry](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus.NodesEntry) | repeated | Nodes is a mapping between a node ID and the node&#39;s status it records the states for the configurations of gateway. |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayStatus.NodesEntry"></a>

### GatewayStatus.NodesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [NodeStatus](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NodeStatus) | optional |  |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NodeStatus"></a>

### NodeStatus
NodeStatus describes the status for an individual node in the gateway configurations.
A single node can represent one configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is a unique identifier of a node within a sensor It is a hash of the node name |
| name | [string](#string) | optional | Name is a unique name in the node tree used to generate the node ID |
| displayName | [string](#string) | optional | DisplayName is the human readable representation of the node |
| phase | [string](#string) | optional | Phase of the node |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | StartedAt is the time at which this node started &#43;k8s:openapi-gen=false |
| message | [string](#string) | optional | Message store data or something to save for configuration |
| updateTime | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime) | optional | UpdateTime is the time when node(gateway configuration) was updated |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.NotificationWatchers"></a>

### NotificationWatchers
NotificationWatchers are components which are interested listening to notifications from this gateway


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gateways | [GatewayNotificationWatcher](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.GatewayNotificationWatcher) | repeated | Gateways is the list of gateways interested in listening to notifications from this gateway |
| sensors | [SensorNotificationWatcher](#github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.SensorNotificationWatcher) | repeated | Sensors is the list of sensors interested in listening to notifications from this gateway |






<a name="github.com.argoproj.argo_events.pkg.apis.gateway.v1alpha1.SensorNotificationWatcher"></a>

### SensorNotificationWatcher
SensorNotificationWatcher is the sensor interested in listening to notifications from this gateway


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is name of the sensor |





 

 

 

 



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

