<p>

Packages:

</p>

<ul>

<li>

<a href="#argoproj.io%2fv1alpha1">argoproj.io/v1alpha1</a>

</li>

</ul>

<h2 id="argoproj.io/v1alpha1">

argoproj.io/v1alpha1

</h2>

<p>

<p>

Package v1alpha1 is the v1alpha1 version of the API.

</p>

</p>

Resource Types:

<ul>

</ul>

<h3 id="argoproj.io/v1alpha1.EventSourceRef">

EventSourceRef

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewaySpec">GatewaySpec</a>)

</p>

<p>

<p>

EventSourceRef holds information about the EventSourceRef custom
resource

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>name</code></br> <em> string </em>

</td>

<td>

<p>

Name of the event source

</p>

</td>

</tr>

<tr>

<td>

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace of the event source Default value is the namespace where
referencing gateway is deployed

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Gateway">

Gateway

</h3>

<p>

<p>

Gateway is the definition of a gateway resource

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>metadata</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>

</td>

<td>

Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="#argoproj.io/v1alpha1.GatewayStatus"> GatewayStatus </a> </em>

</td>

<td>

</td>

</tr>

<tr>

<td>

<code>spec</code></br> <em> <a href="#argoproj.io/v1alpha1.GatewaySpec">
GatewaySpec </a> </em>

</td>

<td>

<br/> <br/>

<table>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Template is the pod specification for the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>eventSourceRef</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceRef"> EventSourceRef </a>
</em>

</td>

<td>

<p>

EventSourceRef refers to event-source that stores event source
configurations for the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em>
github.com/argoproj/argo-events/pkg/apis/common.EventSourceType </em>

</td>

<td>

<p>

Type is the type of gateway. Used as metadata.

</p>

</td>

</tr>

<tr>

<td>

<code>service</code></br> <em> <a href="#argoproj.io/v1alpha1.Service">
Service </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Service is the specifications of the service to expose the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>subscribers</code></br> <em>
<a href="#argoproj.io/v1alpha1.Subscribers"> Subscribers </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Subscribers holds the contexts of the subscribers/sinks to send events
to.

</p>

</td>

</tr>

<tr>

<td>

<code>processorPort</code></br> <em> string </em>

</td>

<td>

<p>

Port on which the gateway event source processor is running on.

</p>

</td>

</tr>

<tr>

<td>

<code>replica</code></br> <em> int </em>

</td>

<td>

<p>

Replica is the gateway deployment replicas

</p>

</td>

</tr>

</table>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GatewayResource">

GatewayResource

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewayStatus">GatewayStatus</a>)

</p>

<p>

<p>

GatewayResource holds the metadata about the gateway resources

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>deployment</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>

</td>

<td>

<p>

Metadata of the deployment for the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>service</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Metadata of the service for the gateway

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GatewaySpec">

GatewaySpec

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Gateway">Gateway</a>)

</p>

<p>

<p>

GatewaySpec represents gateway specifications

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Template is the pod specification for the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>eventSourceRef</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceRef"> EventSourceRef </a>
</em>

</td>

<td>

<p>

EventSourceRef refers to event-source that stores event source
configurations for the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em>
github.com/argoproj/argo-events/pkg/apis/common.EventSourceType </em>

</td>

<td>

<p>

Type is the type of gateway. Used as metadata.

</p>

</td>

</tr>

<tr>

<td>

<code>service</code></br> <em> <a href="#argoproj.io/v1alpha1.Service">
Service </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Service is the specifications of the service to expose the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>subscribers</code></br> <em>
<a href="#argoproj.io/v1alpha1.Subscribers"> Subscribers </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Subscribers holds the contexts of the subscribers/sinks to send events
to.

</p>

</td>

</tr>

<tr>

<td>

<code>processorPort</code></br> <em> string </em>

</td>

<td>

<p>

Port on which the gateway event source processor is running on.

</p>

</td>

</tr>

<tr>

<td>

<code>replica</code></br> <em> int </em>

</td>

<td>

<p>

Replica is the gateway deployment replicas

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GatewayStatus">

GatewayStatus

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Gateway">Gateway</a>)

</p>

<p>

<p>

GatewayStatus contains information about the status of a gateway.

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>phase</code></br> <em> <a href="#argoproj.io/v1alpha1.NodePhase">
NodePhase </a> </em>

</td>

<td>

<p>

Phase is the high-level summary of the gateway

</p>

</td>

</tr>

<tr>

<td>

<code>startedAt</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time </a> </em>

</td>

<td>

<p>

StartedAt is the time at which this gateway was initiated

</p>

</td>

</tr>

<tr>

<td>

<code>message</code></br> <em> string </em>

</td>

<td>

<p>

Message is a human readable string indicating details about a gateway in
its phase

</p>

</td>

</tr>

<tr>

<td>

<code>nodes</code></br> <em> <a href="#argoproj.io/v1alpha1.NodeStatus">
map\[string\]github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1.NodeStatus
</a> </em>

</td>

<td>

<p>

Nodes is a mapping between a node ID and the node’s status it records
the states for the configurations of gateway.

</p>

</td>

</tr>

<tr>

<td>

<code>resources</code></br> <em>
<a href="#argoproj.io/v1alpha1.GatewayResource"> GatewayResource </a>
</em>

</td>

<td>

<p>

Resources refers to the metadata about the gateway resources

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSSubscriber">

NATSSubscriber

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Subscribers">Subscribers</a>)

</p>

<p>

<p>

NATSSubscriber holds the context of subscriber over NATS.

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>serverURL</code></br> <em> string </em>

</td>

<td>

<p>

ServerURL refers to the NATS server URL.

</p>

</td>

</tr>

<tr>

<td>

<code>subject</code></br> <em> string </em>

</td>

<td>

<p>

Subject refers to the NATS subject name.

</p>

</td>

</tr>

<tr>

<td>

<code>name</code></br> <em> string </em>

</td>

<td>

<p>

Name of the subscription. Must be unique.

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NodePhase">

NodePhase (<code>string</code> alias)

</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewayStatus">GatewayStatus</a>,
<a href="#argoproj.io/v1alpha1.NodeStatus">NodeStatus</a>)

</p>

<p>

<p>

NodePhase is the label for the condition of a node.

</p>

</p>

<h3 id="argoproj.io/v1alpha1.NodeStatus">

NodeStatus

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewayStatus">GatewayStatus</a>)

</p>

<p>

<p>

NodeStatus describes the status for an individual node in the gateway
configurations. A single node can represent one configuration.

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>id</code></br> <em> string </em>

</td>

<td>

<p>

ID is a unique identifier of a node within a sensor It is a hash of the
node name

</p>

</td>

</tr>

<tr>

<td>

<code>name</code></br> <em> string </em>

</td>

<td>

<p>

Name is a unique name in the node tree used to generate the node ID

</p>

</td>

</tr>

<tr>

<td>

<code>displayName</code></br> <em> string </em>

</td>

<td>

<p>

DisplayName is the human readable representation of the node

</p>

</td>

</tr>

<tr>

<td>

<code>phase</code></br> <em> <a href="#argoproj.io/v1alpha1.NodePhase">
NodePhase </a> </em>

</td>

<td>

<p>

Phase of the node

</p>

</td>

</tr>

<tr>

<td>

<code>startedAt</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#microtime-v1-meta">
Kubernetes meta/v1.MicroTime </a> </em>

</td>

<td>

<p>

StartedAt is the time at which this node started

</p>

</td>

</tr>

<tr>

<td>

<code>message</code></br> <em> string </em>

</td>

<td>

<p>

Message store data or something to save for configuration

</p>

</td>

</tr>

<tr>

<td>

<code>updateTime</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#microtime-v1-meta">
Kubernetes meta/v1.MicroTime </a> </em>

</td>

<td>

<p>

UpdateTime is the time when node(gateway configuration) was updated

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Service">

Service

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewaySpec">GatewaySpec</a>)

</p>

<p>

<p>

Service holds the service information gateway exposes

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>ports</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#serviceport-v1-core">
\[\]Kubernetes core/v1.ServicePort </a> </em>

</td>

<td>

<p>

The list of ports that are exposed by this service.

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Subscribers">

Subscribers

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewaySpec">GatewaySpec</a>)

</p>

<p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>http</code></br> <em> \[\]string </em>

</td>

<td>

<em>(Optional)</em>

<p>

HTTP subscribers are HTTP endpoints to send events to.

</p>

</td>

</tr>

<tr>

<td>

<code>nats</code></br> <em>
<a href="#argoproj.io/v1alpha1.NATSSubscriber"> \[\]NATSSubscriber </a>
</em>

</td>

<td>

<em>(Optional)</em>

<p>

NATS refers to the subscribers over NATS protocol.

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Template">

Template

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GatewaySpec">GatewaySpec</a>)

</p>

<p>

<p>

Template holds the information of a Gateway deployment template

</p>

</p>

<table>

<thead>

<tr>

<th>

Field

</th>

<th>

Description

</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>serviceAccountName</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

ServiceAccountName is the name of the ServiceAccount to use to run
gateway pod. More info:
<a href="https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/">https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/</a>

</p>

</td>

</tr>

<tr>

<td>

<code>container</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#container-v1-core">
Kubernetes core/v1.Container </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Container is the main container image to run in the gateway pod

</p>

</td>

</tr>

<tr>

<td>

<code>volumes</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#volume-v1-core">
\[\]Kubernetes core/v1.Volume </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Volumes is a list of volumes that can be mounted by containers in a
workflow.

</p>

</td>

</tr>

<tr>

<td>

<code>securityContext</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext </a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

SecurityContext holds pod-level security attributes and common container
settings. Optional: Defaults to empty. See type description for default
values of each field.

</p>

</td>

</tr>

</tbody>

</table>

<hr/>

<p>

<em> Generated with <code>gen-crd-api-reference-docs</code> on git
commit <code>09ad9fa</code>. </em>

</p>
