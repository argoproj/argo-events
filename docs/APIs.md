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

</p>

Resource Types:
<ul>

</ul>

<h3 id="argoproj.io/v1alpha1.AMQPConsumeConfig">

AMQPConsumeConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>)
</p>

<p>

<p>

AMQPConsumeConfig holds the configuration to immediately starts
delivering queued messages
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

<code>consumerTag</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ConsumerTag is the identity of the consumer included in every delivery
</p>

</td>

</tr>

<tr>

<td>

<code>autoAck</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

AutoAck when true, the server will acknowledge deliveries to this
consumer prior to writing the delivery to the network
</p>

</td>

</tr>

<tr>

<td>

<code>exclusive</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Exclusive when true, the server will ensure that this is the sole
consumer from this queue
</p>

</td>

</tr>

<tr>

<td>

<code>noLocal</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

NoLocal flag is not supported by RabbitMQ
</p>

</td>

</tr>

<tr>

<td>

<code>noWait</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

NowWait when true, do not wait for the server to confirm the request and
immediately begin deliveries
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AMQPEventSource">

AMQPEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

AMQPEventSource refers to an event-source for AMQP stream events
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL for rabbitmq service
</p>

</td>

</tr>

<tr>

<td>

<code>exchangeName</code></br> <em> string </em>
</td>

<td>

<p>

ExchangeName is the exchange name For more information, visit
<a href="https://www.rabbitmq.com/tutorials/amqp-concepts.html">https://www.rabbitmq.com/tutorials/amqp-concepts.html</a>
</p>

</td>

</tr>

<tr>

<td>

<code>exchangeType</code></br> <em> string </em>
</td>

<td>

<p>

ExchangeType is rabbitmq exchange type
</p>

</td>

</tr>

<tr>

<td>

<code>routingKey</code></br> <em> string </em>
</td>

<td>

<p>

Routing key for bindings
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the amqp client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>exchangeDeclare</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPExchangeDeclareConfig">
AMQPExchangeDeclareConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ExchangeDeclare holds the configuration for the exchange on the server
For more information, visit
<a href="https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.ExchangeDeclare">https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.ExchangeDeclare</a>
</p>

</td>

</tr>

<tr>

<td>

<code>queueDeclare</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPQueueDeclareConfig">
AMQPQueueDeclareConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

QueueDeclare holds the configuration of a queue to hold messages and
deliver to consumers. Declaring creates a queue if it doesn’t already
exist, or ensures that an existing queue matches the same parameters For
more information, visit
<a href="https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.QueueDeclare">https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.QueueDeclare</a>
</p>

</td>

</tr>

<tr>

<td>

<code>queueBind</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPQueueBindConfig"> AMQPQueueBindConfig
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

QueueBind holds the configuration that binds an exchange to a queue so
that publishings to the exchange will be routed to the queue when the
publishing routing key matches the binding routing key For more
information, visit
<a href="https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.QueueBind">https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.QueueBind</a>
</p>

</td>

</tr>

<tr>

<td>

<code>consume</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPConsumeConfig"> AMQPConsumeConfig
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Consume holds the configuration to immediately starts delivering queued
messages For more information, visit
<a href="https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.Consume">https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.Consume</a>
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em> <a href="#argoproj.io/v1alpha1.BasicAuth">
BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Auth hosts secret selectors for username and password
</p>

</td>

</tr>

<tr>

<td>

<code>urlSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

URLSecret is secret reference for rabbitmq service URL
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AMQPExchangeDeclareConfig">

AMQPExchangeDeclareConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>)
</p>

<p>

<p>

AMQPExchangeDeclareConfig holds the configuration for the exchange on
the server
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

<code>durable</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Durable keeps the exchange also after the server restarts
</p>

</td>

</tr>

<tr>

<td>

<code>autoDelete</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

AutoDelete removes the exchange when no bindings are active
</p>

</td>

</tr>

<tr>

<td>

<code>internal</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Internal when true does not accept publishings
</p>

</td>

</tr>

<tr>

<td>

<code>noWait</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

NowWait when true does not wait for a confirmation from the server
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AMQPQueueBindConfig">

AMQPQueueBindConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>)
</p>

<p>

<p>

AMQPQueueBindConfig holds the configuration that binds an exchange to a
queue so that publishings to the exchange will be routed to the queue
when the publishing routing key matches the binding routing key
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

<code>noWait</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

NowWait false and the queue could not be bound, the channel will be
closed with an error
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AMQPQueueDeclareConfig">

AMQPQueueDeclareConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>)
</p>

<p>

<p>

AMQPQueueDeclareConfig holds the configuration of a queue to hold
messages and deliver to consumers. Declaring creates a queue if it
doesn’t already exist, or ensures that an existing queue matches the
same parameters
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

<em>(Optional)</em>
<p>

Name of the queue. If empty the server auto-generates a unique name for
this queue
</p>

</td>

</tr>

<tr>

<td>

<code>durable</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Durable keeps the queue also after the server restarts
</p>

</td>

</tr>

<tr>

<td>

<code>autoDelete</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

AutoDelete removes the queue when no consumers are active
</p>

</td>

</tr>

<tr>

<td>

<code>exclusive</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Exclusive sets the queues to be accessible only by the connection that
declares them and will be deleted wgen the connection closes
</p>

</td>

</tr>

<tr>

<td>

<code>noWait</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

NowWait when true, the queue assumes to be declared on the server
</p>

</td>

</tr>

<tr>

<td>

<code>arguments</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Arguments of a queue (also known as “x-arguments”) used for optional
features and plugins
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AWSLambdaTrigger">

AWSLambdaTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

AWSLambdaTrigger refers to specification of the trigger to invoke an AWS
Lambda function
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

<code>functionName</code></br> <em> string </em>
</td>

<td>

<p>

FunctionName refers to the name of the function to invoke.
</p>

</td>

</tr>

<tr>

<td>

<code>accessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AccessKey refers K8s secret containing aws access key
</p>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SecretKey refers K8s secret containing aws secret key
</p>

</td>

</tr>

<tr>

<td>

<code>region</code></br> <em> string </em>
</td>

<td>

<p>

Region is AWS region
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

<tr>

<td>

<code>invocationType</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Choose from the following options.
</p>

<ul>

<li>

<p>

RequestResponse (default) - Invoke the function synchronously. Keep the
connection open until the function returns a response or times out. The
API response includes the function response and additional data.
</p>

</li>

<li>

<p>

Event - Invoke the function asynchronously. Send events that fail
multiple times to the function’s dead-letter queue (if it’s configured).
The API response only includes a status code.
</p>

</li>

<li>

<p>

DryRun - Validate parameter values and verify that the user or role has
permission to invoke the function.
</p>

</li>

</ul>

</td>

</tr>

<tr>

<td>

<code>roleARN</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

RoleARN is the Amazon Resource Name (ARN) of the role to assume.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Amount">

Amount
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Backoff">Backoff</a>)
</p>

<p>

<p>

Amount represent a numeric amount.
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

<code>value</code></br> <em> \[\]byte </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ArgoWorkflowOperation">

ArgoWorkflowOperation (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArgoWorkflowTrigger">ArgoWorkflowTrigger</a>)
</p>

<p>

<p>

ArgoWorkflowOperation refers to the type of the operation performed on
the Argo Workflow
</p>

</p>

<h3 id="argoproj.io/v1alpha1.ArgoWorkflowTrigger">

ArgoWorkflowTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

ArgoWorkflowTrigger is the trigger for the Argo Workflow
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

<code>source</code></br> <em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation"> ArtifactLocation </a>
</em>
</td>

<td>

<p>

Source of the K8s resource file(s)
</p>

</td>

</tr>

<tr>

<td>

<code>operation</code></br> <em>
<a href="#argoproj.io/v1alpha1.ArgoWorkflowOperation">
ArgoWorkflowOperation </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Operation refers to the type of operation performed on the argo workflow
resource. Default value is Submit.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters to pass to resolved Argo Workflow
object
</p>

</td>

</tr>

<tr>

<td>

<code>args</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Args is the list of arguments to pass to the argo CLI
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ArtifactLocation">

ArtifactLocation
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArgoWorkflowTrigger">ArgoWorkflowTrigger</a>,
<a href="#argoproj.io/v1alpha1.StandardK8STrigger">StandardK8STrigger</a>)
</p>

<p>

<p>

ArtifactLocation describes the source location for an external artifact
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

<code>s3</code></br> <em> <a href="#argoproj.io/v1alpha1.S3Artifact">
S3Artifact </a> </em>
</td>

<td>

<p>

S3 compliant artifact
</p>

</td>

</tr>

<tr>

<td>

<code>inline</code></br> <em> string </em>
</td>

<td>

<p>

Inline artifact is embedded in sensor spec as a string
</p>

</td>

</tr>

<tr>

<td>

<code>file</code></br> <em>
<a href="#argoproj.io/v1alpha1.FileArtifact"> FileArtifact </a> </em>
</td>

<td>

<p>

File artifact is artifact stored in a file
</p>

</td>

</tr>

<tr>

<td>

<code>url</code></br> <em> <a href="#argoproj.io/v1alpha1.URLArtifact">
URLArtifact </a> </em>
</td>

<td>

<p>

URL to fetch the artifact from
</p>

</td>

</tr>

<tr>

<td>

<code>configmap</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#configmapkeyselector-v1-core">
Kubernetes core/v1.ConfigMapKeySelector </a> </em>
</td>

<td>

<p>

Configmap that stores the artifact
</p>

</td>

</tr>

<tr>

<td>

<code>git</code></br> <em> <a href="#argoproj.io/v1alpha1.GitArtifact">
GitArtifact </a> </em>
</td>

<td>

<p>

Git repository hosting the artifact
</p>

</td>

</tr>

<tr>

<td>

<code>resource</code></br> <em>
<a href="#argoproj.io/v1alpha1.K8SResource"> K8SResource </a> </em>
</td>

<td>

<p>

Resource is generic template for K8s resource
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AuthStrategy">

AuthStrategy (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.NATSConfig">NATSConfig</a>,
<a href="#argoproj.io/v1alpha1.NativeStrategy">NativeStrategy</a>)
</p>

<p>

<p>

AuthStrategy is the auth strategy of native nats installaion
</p>

</p>

<h3 id="argoproj.io/v1alpha1.AzureEventHubsTrigger">

AzureEventHubsTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

AzureEventHubsTrigger refers to specification of the Azure Event Hubs
Trigger
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

<code>fqdn</code></br> <em> string </em>
</td>

<td>

<p>

FQDN refers to the namespace dns of Azure Event Hubs to be used
i.e. <namespace>.servicebus.windows.net
</p>

</td>

</tr>

<tr>

<td>

<code>hubName</code></br> <em> string </em>
</td>

<td>

<p>

HubName refers to the Azure Event Hub to send events to
</p>

</td>

</tr>

<tr>

<td>

<code>sharedAccessKeyName</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SharedAccessKeyName refers to the name of the Shared Access Key
</p>

</td>

</tr>

<tr>

<td>

<code>sharedAccessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SharedAccessKey refers to a K8s secret containing the primary key for
the
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AzureEventsHubEventSource">

AzureEventsHubEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

AzureEventsHubEventSource describes the event source for azure events
hub More info at
<a href="https://docs.microsoft.com/en-us/azure/event-hubs/">https://docs.microsoft.com/en-us/azure/event-hubs/</a>
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

<code>fqdn</code></br> <em> string </em>
</td>

<td>

<p>

FQDN of the EventHubs namespace you created More info at
<a href="https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string">https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string</a>
</p>

</td>

</tr>

<tr>

<td>

<code>sharedAccessKeyName</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SharedAccessKeyName is the name you chose for your application’s SAS
keys
</p>

</td>

</tr>

<tr>

<td>

<code>sharedAccessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SharedAccessKey is the generated value of the key
</p>

</td>

</tr>

<tr>

<td>

<code>hubName</code></br> <em> string </em>
</td>

<td>

<p>

Event Hub path/name
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AzureQueueStorageEventSource">

AzureQueueStorageEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

AzureQueueStorageEventSource describes the event source for azure queue
storage more info at
<a href="https://learn.microsoft.com/en-us/azure/storage/queues/">https://learn.microsoft.com/en-us/azure/storage/queues/</a>
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

<code>storageAccountName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

StorageAccountName is the name of the storage account where the queue
is. This field is necessary to access via Azure AD (managed identity)
and it is ignored if ConnectionString is set.
</p>

</td>

</tr>

<tr>

<td>

<code>connectionString</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ConnectionString is the connection string to access Azure Queue Storage.
If this fields is not provided it will try to access via Azure AD with
StorageAccountName.
</p>

</td>

</tr>

<tr>

<td>

<code>queueName</code></br> <em> string </em>
</td>

<td>

<p>

QueueName is the name of the queue
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>dlq</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DLQ specifies if a dead-letter queue is configured for messages that
can’t be processed successfully. If set to true, messages with invalid
payload won’t be acknowledged to allow to forward them farther to the
dead-letter queue. The default value is false.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>decodeMessage</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DecodeMessage specifies if all the messages should be base64 decoded. If
set to true the decoding is done before the evaluation of JSONBody
</p>

</td>

</tr>

<tr>

<td>

<code>waitTimeInSeconds</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

WaitTimeInSeconds is the duration (in seconds) for which the event
source waits between empty results from the queue. The default value is
3 seconds.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AzureServiceBusEventSource">

AzureServiceBusEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

AzureServiceBusEventSource describes the event source for azure service
bus More info at
<a href="https://docs.microsoft.com/en-us/azure/service-bus-messaging/">https://docs.microsoft.com/en-us/azure/service-bus-messaging/</a>
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

<code>connectionString</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ConnectionString is the connection string for the Azure Service Bus. If
this fields is not provided it will try to access via Azure AD with
DefaultAzureCredential and FullyQualifiedNamespace.
</p>

</td>

</tr>

<tr>

<td>

<code>queueName</code></br> <em> string </em>
</td>

<td>

<p>

QueueName is the name of the Azure Service Bus Queue
</p>

</td>

</tr>

<tr>

<td>

<code>topicName</code></br> <em> string </em>
</td>

<td>

<p>

TopicName is the name of the Azure Service Bus Topic
</p>

</td>

</tr>

<tr>

<td>

<code>subscriptionName</code></br> <em> string </em>
</td>

<td>

<p>

SubscriptionName is the name of the Azure Service Bus Topic Subscription
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the service bus client
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>fullyQualifiedNamespace</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

FullyQualifiedNamespace is the Service Bus namespace name (ex:
myservicebus.servicebus.windows.net). This field is necessary to access
via Azure AD (managed identity) and it is ignored if ConnectionString is
set.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.AzureServiceBusTrigger">

AzureServiceBusTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
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

<code>connectionString</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

ConnectionString is the connection string for the Azure Service Bus
</p>

</td>

</tr>

<tr>

<td>

<code>queueName</code></br> <em> string </em>
</td>

<td>

<p>

QueueName is the name of the Azure Service Bus Queue
</p>

</td>

</tr>

<tr>

<td>

<code>topicName</code></br> <em> string </em>
</td>

<td>

<p>

TopicName is the name of the Azure Service Bus Topic
</p>

</td>

</tr>

<tr>

<td>

<code>subscriptionName</code></br> <em> string </em>
</td>

<td>

<p>

SubscriptionName is the name of the Azure Service Bus Topic Subscription
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the service bus client
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Backoff">

Backoff
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>,
<a href="#argoproj.io/v1alpha1.EmitterEventSource">EmitterEventSource</a>,
<a href="#argoproj.io/v1alpha1.K8SResourcePolicy">K8SResourcePolicy</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>,
<a href="#argoproj.io/v1alpha1.MQTTEventSource">MQTTEventSource</a>,
<a href="#argoproj.io/v1alpha1.NATSEventsSource">NATSEventsSource</a>,
<a href="#argoproj.io/v1alpha1.NSQEventSource">NSQEventSource</a>,
<a href="#argoproj.io/v1alpha1.PulsarEventSource">PulsarEventSource</a>,
<a href="#argoproj.io/v1alpha1.PulsarTrigger">PulsarTrigger</a>,
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
</p>

<p>

<p>

Backoff for an operation
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

<code>duration</code></br> <em>
<a href="#argoproj.io/v1alpha1.Int64OrString"> Int64OrString </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

The initial duration in nanoseconds or strings like “1s”, “3m”
</p>

</td>

</tr>

<tr>

<td>

<code>factor</code></br> <em> <a href="#argoproj.io/v1alpha1.Amount">
Amount </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Duration is multiplied by factor each iteration
</p>

</td>

</tr>

<tr>

<td>

<code>jitter</code></br> <em> <a href="#argoproj.io/v1alpha1.Amount">
Amount </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

The amount of jitter applied each iteration
</p>

</td>

</tr>

<tr>

<td>

<code>steps</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Exit with error after this many steps
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BasicAuth">

BasicAuth
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>,
<a href="#argoproj.io/v1alpha1.GerritEventSource">GerritEventSource</a>,
<a href="#argoproj.io/v1alpha1.HTTPTrigger">HTTPTrigger</a>,
<a href="#argoproj.io/v1alpha1.MQTTEventSource">MQTTEventSource</a>,
<a href="#argoproj.io/v1alpha1.NATSAuth">NATSAuth</a>,
<a href="#argoproj.io/v1alpha1.SchemaRegistryConfig">SchemaRegistryConfig</a>)
</p>

<p>

<p>

BasicAuth contains the reference to K8s secrets that holds the username
and password
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

<code>username</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Username refers to the Kubernetes secret that holds the username
required for basic auth.
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Password refers to the Kubernetes secret that holds the password
required for basic auth.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketAuth">

BitbucketAuth
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">BitbucketEventSource</a>)
</p>

<p>

<p>

BitbucketAuth holds the different auth strategies for connecting to
Bitbucket
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

<code>basic</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketBasicAuth"> BitbucketBasicAuth
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Basic is BasicAuth auth strategy.
</p>

</td>

</tr>

<tr>

<td>

<code>oauthToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

OAuthToken refers to the K8s secret that holds the OAuth Bearer token.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketBasicAuth">

BitbucketBasicAuth
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BitbucketAuth">BitbucketAuth</a>)
</p>

<p>

<p>

BitbucketBasicAuth holds the information required to authenticate user
via basic auth mechanism
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

<code>username</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Username refers to the K8s secret that holds the username.
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Password refers to the K8s secret that holds the password.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketEventSource">

BitbucketEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

BitbucketEventSource describes the event source for Bitbucket
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

<code>deleteHookOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteHookOnFinish determines whether to delete the defined Bitbucket
hook once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will be passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook refers to the configuration required to run an http server
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketAuth"> BitbucketAuth </a> </em>
</td>

<td>

<p>

Auth information required to connect to Bitbucket.
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Events this webhook is subscribed to.
</p>

</td>

</tr>

<tr>

<td>

<code>owner</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedOwner is the owner of the repository. Deprecated: use
Repositories instead. Will be unsupported in v1.9
</p>

</td>

</tr>

<tr>

<td>

<code>projectKey</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedProjectKey is the key of the project to which the repository
relates Deprecated: use Repositories instead. Will be unsupported in
v1.9
</p>

</td>

</tr>

<tr>

<td>

<code>repositorySlug</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedRepositorySlug is a URL-friendly version of a repository name,
automatically generated by Bitbucket for use in the URL Deprecated: use
Repositories instead. Will be unsupported in v1.9
</p>

</td>

</tr>

<tr>

<td>

<code>repositories</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketRepository">
\[\]BitbucketRepository </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Repositories holds a list of repositories for which integration needs to
set up
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketRepository">

BitbucketRepository
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">BitbucketEventSource</a>)
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

<code>owner</code></br> <em> string </em>
</td>

<td>

<p>

Owner is the owner of the repository
</p>

</td>

</tr>

<tr>

<td>

<code>repositorySlug</code></br> <em> string </em>
</td>

<td>

<p>

RepositorySlug is a URL-friendly version of a repository name,
automatically generated by Bitbucket for use in the URL
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketServerEventSource">

BitbucketServerEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

BitbucketServerEventSource refers to event-source related to Bitbucket
Server events
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration to run a http server.
</p>

</td>

</tr>

<tr>

<td>

<code>projectKey</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedProjectKey is the key of project for which integration needs
to set up. Deprecated: use Repositories instead. Will be unsupported in
v1.8.
</p>

</td>

</tr>

<tr>

<td>

<code>repositorySlug</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedRepositorySlug is the slug of the repository for which
integration needs to set up. Deprecated: use Repositories instead. Will
be unsupported in v1.8.
</p>

</td>

</tr>

<tr>

<td>

<code>projects</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Projects holds a list of projects for which integration needs to set up,
this will add the webhook to all repositories in the project.
</p>

</td>

</tr>

<tr>

<td>

<code>repositories</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketServerRepository">
\[\]BitbucketServerRepository </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Repositories holds a list of repositories for which integration needs to
set up.
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Events are bitbucket event to listen to. Refer
<a href="https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html">https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html</a>
</p>

</td>

</tr>

<tr>

<td>

<code>skipBranchRefsChangedOnOpenPR</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

SkipBranchRefsChangedOnOpenPR bypasses the event repo:refs_changed for
branches whenever there’s an associated open pull request. This helps in
optimizing the event handling process by avoiding unnecessary triggers
for branch reference changes that are already part of a pull request
under review.
</p>

</td>

</tr>

<tr>

<td>

<code>oneEventPerChange</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

OneEventPerChange controls whether to process each change in a
repo:refs_changed webhook event as a separate event. This setting is
useful when multiple tags are pushed simultaneously for the same commit,
and each tag needs to independently trigger an action, such as a
distinct workflow in Argo Workflows. When enabled, the
BitbucketServerEventSource publishes an individual
BitbucketServerEventData for each change, ensuring independent
processing of each tag or reference update in a single webhook event.
</p>

</td>

</tr>

<tr>

<td>

<code>accessToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AccessToken is reference to K8s secret which holds the bitbucket api
access information.
</p>

</td>

</tr>

<tr>

<td>

<code>webhookSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

WebhookSecret is reference to K8s secret which holds the bitbucket
webhook secret (for HMAC validation).
</p>

</td>

</tr>

<tr>

<td>

<code>bitbucketserverBaseURL</code></br> <em> string </em>
</td>

<td>

<p>

BitbucketServerBaseURL is the base URL for API requests to a custom
endpoint.
</p>

</td>

</tr>

<tr>

<td>

<code>deleteHookOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteHookOnFinish determines whether to delete the Bitbucket Server
hook for the project once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the bitbucketserver client.
</p>

</td>

</tr>

<tr>

<td>

<code>checkInterval</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

CheckInterval is a duration in which to wait before checking that the
webhooks exist, e.g. 1s, 30m, 2h… (defaults to 1m)
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BitbucketServerRepository">

BitbucketServerRepository
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">BitbucketServerEventSource</a>)
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

<code>projectKey</code></br> <em> string </em>
</td>

<td>

<p>

ProjectKey is the key of project for which integration needs to set up.
</p>

</td>

</tr>

<tr>

<td>

<code>repositorySlug</code></br> <em> string </em>
</td>

<td>

<p>

RepositorySlug is the slug of the repository for which integration needs
to set up.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.BusConfig">

BusConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBusStatus">EventBusStatus</a>)
</p>

<p>

<p>

BusConfig has the finalized configuration for EventBus
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

<code>nats</code></br> <em> <a href="#argoproj.io/v1alpha1.NATSConfig">
NATSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>jetstream</code></br> <em>
<a href="#argoproj.io/v1alpha1.JetStreamConfig"> JetStreamConfig </a>
</em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em> <a href="#argoproj.io/v1alpha1.KafkaBus">
KafkaBus </a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.CalendarEventSource">

CalendarEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

CalendarEventSource describes a time based dependency. One of the fields
(schedule, interval, or recurrence) must be passed. Schedule takes
precedence over interval; interval takes precedence over recurrence
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

<code>schedule</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Schedule is a cron-like expression. For reference, see:
<a href="https://en.wikipedia.org/wiki/Cron">https://en.wikipedia.org/wiki/Cron</a>
</p>

</td>

</tr>

<tr>

<td>

<code>interval</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Interval is a string that describes an interval duration, e.g. 1s, 30m,
2h…
</p>

</td>

</tr>

<tr>

<td>

<code>exclusionDates</code></br> <em> \[\]string </em>
</td>

<td>

<p>

ExclusionDates defines the list of DATE-TIME exceptions for recurring
events.
</p>

</td>

</tr>

<tr>

<td>

<code>timezone</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Timezone in which to run the schedule
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>persistence</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventPersistence"> EventPersistence </a>
</em>
</td>

<td>

<p>

Persistence hold the configuration for event persistence
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.CatchupConfiguration">

CatchupConfiguration
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventPersistence">EventPersistence</a>)
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

<code>enabled</code></br> <em> bool </em>
</td>

<td>

<p>

Enabled enables to triggered the missed schedule when eventsource
restarts
</p>

</td>

</tr>

<tr>

<td>

<code>maxDuration</code></br> <em> string </em>
</td>

<td>

<p>

MaxDuration holds max catchup duration
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Comparator">

Comparator (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.DataFilter">DataFilter</a>)
</p>

<p>

<p>

Comparator refers to the comparator operator for a data filter
</p>

</p>

<h3 id="argoproj.io/v1alpha1.Condition">

Condition
</h3>

<p>

(<em>Appears on:</em> <a href="#argoproj.io/v1alpha1.Status">Status</a>)
</p>

<p>

<p>

Condition contains details about resource state
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

<code>type</code></br> <em>
<a href="#argoproj.io/v1alpha1.ConditionType"> ConditionType </a> </em>
</td>

<td>

<p>

Condition type.
</p>

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus </a> </em>
</td>

<td>

<p>

Condition status, True, False or Unknown.
</p>

</td>

</tr>

<tr>

<td>

<code>lastTransitionTime</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#time-v1-meta">
Kubernetes meta/v1.Time </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Last time the condition transitioned from one status to another.
</p>

</td>

</tr>

<tr>

<td>

<code>reason</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Unique, this should be a short, machine understandable string that gives
the reason for condition’s last transition. For example, “ImageNotFound”
</p>

</td>

</tr>

<tr>

<td>

<code>message</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Human-readable message indicating details about last transition.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ConditionType">

ConditionType (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Condition">Condition</a>)
</p>

<p>

<p>

ConditionType is a valid value of Condition.Type
</p>

</p>

<h3 id="argoproj.io/v1alpha1.ConditionsResetByTime">

ConditionsResetByTime
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ConditionsResetCriteria">ConditionsResetCriteria</a>)
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

<code>cron</code></br> <em> string </em>
</td>

<td>

<p>

Cron is a cron-like expression. For reference, see:
<a href="https://en.wikipedia.org/wiki/Cron">https://en.wikipedia.org/wiki/Cron</a>
</p>

</td>

</tr>

<tr>

<td>

<code>timezone</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ConditionsResetCriteria">

ConditionsResetCriteria
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
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

<code>byTime</code></br> <em>
<a href="#argoproj.io/v1alpha1.ConditionsResetByTime">
ConditionsResetByTime </a> </em>
</td>

<td>

<p>

Schedule is a cron-like expression. For reference, see:
<a href="https://en.wikipedia.org/wiki/Cron">https://en.wikipedia.org/wiki/Cron</a>
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ConfigMapPersistence">

ConfigMapPersistence
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventPersistence">EventPersistence</a>)
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

<code>name</code></br> <em> string </em>
</td>

<td>

<p>

Name of the configmap
</p>

</td>

</tr>

<tr>

<td>

<code>createIfNotExist</code></br> <em> bool </em>
</td>

<td>

<p>

CreateIfNotExist will create configmap if it doesn’t exists
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ContainerTemplate">

ContainerTemplate
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.JetStreamBus">JetStreamBus</a>,
<a href="#argoproj.io/v1alpha1.NativeStrategy">NativeStrategy</a>)
</p>

<p>

<p>

ContainerTemplate defines customized spec for a container
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

<code>resources</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>imagePullPolicy</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>securityContext</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#securitycontext-v1-core">
Kubernetes core/v1.SecurityContext </a> </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.CustomTrigger">

CustomTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

CustomTrigger refers to the specification of the custom trigger.
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

ServerURL is the url of the gRPC server that executes custom trigger
</p>

</td>

</tr>

<tr>

<td>

<code>secure</code></br> <em> bool </em>
</td>

<td>

<p>

Secure refers to type of the connection between sensor to custom trigger
gRPC
</p>

</td>

</tr>

<tr>

<td>

<code>certSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

CertSecret refers to the secret that contains cert for secure connection
between sensor and custom trigger gRPC server.
</p>

</td>

</tr>

<tr>

<td>

<code>serverNameOverride</code></br> <em> string </em>
</td>

<td>

<p>

ServerNameOverride for the secure connection between sensor and custom
trigger gRPC server.
</p>

</td>

</tr>

<tr>

<td>

<code>spec</code></br> <em> map\[string\]string </em>
</td>

<td>

<p>

Spec is the custom trigger resource specification that custom trigger
gRPC server knows how to interpret.
</p>

<br/> <br/>
<table>

</table>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters that is applied to resolved custom
trigger trigger object.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.DataFilter">

DataFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">EventDependencyFilter</a>)
</p>

<p>

<p>

DataFilter describes constraints and filters for event data Regular
Expressions are purposefully not a feature as they are overkill for our
uses here See Rob Pike’s Post:
<a href="https://commandcenter.blogspot.com/2011/08/regular-expressions-in-lexing-and.html">https://commandcenter.blogspot.com/2011/08/regular-expressions-in-lexing-and.html</a>
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

<code>path</code></br> <em> string </em>
</td>

<td>

<p>

Path is the JSONPath of the event’s (JSON decoded) data key Path is a
series of keys separated by a dot. A key may contain wildcard characters
‘\*’ and ‘?’. To access an array value use the index as the key. The dot
and wildcard characters can be escaped with ‘&rsquo;. See
<a href="https://github.com/tidwall/gjson#path-syntax">https://github.com/tidwall/gjson#path-syntax</a>
for more information on how to use this.
</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em> <a href="#argoproj.io/v1alpha1.JSONType">
JSONType </a> </em>
</td>

<td>

<p>

Type contains the JSON type of the data
</p>

</td>

</tr>

<tr>

<td>

<code>value</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Value is the allowed string values for this key Booleans are passed
using strconv.ParseBool() Numbers are parsed using as float64 using
strconv.ParseFloat() Strings are taken as is Nils this value is ignored
</p>

</td>

</tr>

<tr>

<td>

<code>comparator</code></br> <em>
<a href="#argoproj.io/v1alpha1.Comparator"> Comparator </a> </em>
</td>

<td>

<p>

Comparator compares the event data with a user given value. Can be
“\>=”, “\>”, “=”, “!=”, “\<”, or “\<=”. Is optional, and if left blank
treated as equality “=”.
</p>

</td>

</tr>

<tr>

<td>

<code>template</code></br> <em> string </em>
</td>

<td>

<p>

Template is a go-template for extracting a string from the event’s data.
A Template is evaluated with provided path, type and value. The
templating follows the standard go-template syntax as well as sprig’s
extra functions. See
<a href="https://pkg.go.dev/text/template">https://pkg.go.dev/text/template</a>
and
<a href="https://masterminds.github.io/sprig/">https://masterminds.github.io/sprig/</a>
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EmailTrigger">

EmailTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

EmailTrigger refers to the specification of the email notification
trigger.
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

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Username refers to the username used to connect to the smtp server.
</p>

</td>

</tr>

<tr>

<td>

<code>smtpPassword</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SMTPPassword refers to the Kubernetes secret that holds the smtp
password used to connect to smtp server.
</p>

</td>

</tr>

<tr>

<td>

<code>host</code></br> <em> string </em>
</td>

<td>

<p>

Host refers to the smtp host url to which email is send.
</p>

</td>

</tr>

<tr>

<td>

<code>port</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Port refers to the smtp server port to which email is send. Defaults to
0.
</p>

</td>

</tr>

<tr>

<td>

<code>to</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

To refers to the email addresses to which the emails are send.
</p>

</td>

</tr>

<tr>

<td>

<code>from</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

From refers to the address from which the email is send from.
</p>

</td>

</tr>

<tr>

<td>

<code>subject</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Subject refers to the subject line for the email send.
</p>

</td>

</tr>

<tr>

<td>

<code>body</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Body refers to the body/content of the email send.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EmitterEventSource">

EmitterEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

EmitterEventSource describes the event source for emitter More info at
<a href="https://emitter.io/develop/getting-started/">https://emitter.io/develop/getting-started/</a>
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

<code>broker</code></br> <em> string </em>
</td>

<td>

<p>

Broker URI to connect to.
</p>

</td>

</tr>

<tr>

<td>

<code>channelKey</code></br> <em> string </em>
</td>

<td>

<p>

ChannelKey refers to the channel key
</p>

</td>

</tr>

<tr>

<td>

<code>channelName</code></br> <em> string </em>
</td>

<td>

<p>

ChannelName refers to the channel name
</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Username to use to connect to broker
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Password to use to connect to broker
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the emitter client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Event">

Event
</h3>

<p>

<p>

Event represents the cloudevent received from an event source.
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

<code>context</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventContext"> EventContext </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>data</code></br> <em> \[\]byte </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventBus">

EventBus
</h3>

<p>

<p>

EventBus is the definition of a eventbus resource
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
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>
</td>

<td>

Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>

</tr>

<tr>

<td>

<code>spec</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventBusSpec"> EventBusSpec </a> </em>
</td>

<td>

<br/> <br/>
<table>

<tr>

<td>

<code>nats</code></br> <em> <a href="#argoproj.io/v1alpha1.NATSBus">
NATSBus </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

NATS eventbus
</p>

</td>

</tr>

<tr>

<td>

<code>jetstream</code></br> <em>
<a href="#argoproj.io/v1alpha1.JetStreamBus"> JetStreamBus </a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em> <a href="#argoproj.io/v1alpha1.KafkaBus">
KafkaBus </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Kafka eventbus
</p>

</td>

</tr>

<tr>

<td>

<code>jetstreamExotic</code></br> <em>
<a href="#argoproj.io/v1alpha1.JetStreamConfig"> JetStreamConfig </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

Exotic JetStream
</p>

</td>

</tr>

</table>

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventBusStatus"> EventBusStatus </a>
</em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventBusSpec">

EventBusSpec
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBus">EventBus</a>)
</p>

<p>

<p>

EventBusSpec refers to specification of eventbus resource
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

<code>nats</code></br> <em> <a href="#argoproj.io/v1alpha1.NATSBus">
NATSBus </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

NATS eventbus
</p>

</td>

</tr>

<tr>

<td>

<code>jetstream</code></br> <em>
<a href="#argoproj.io/v1alpha1.JetStreamBus"> JetStreamBus </a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em> <a href="#argoproj.io/v1alpha1.KafkaBus">
KafkaBus </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Kafka eventbus
</p>

</td>

</tr>

<tr>

<td>

<code>jetstreamExotic</code></br> <em>
<a href="#argoproj.io/v1alpha1.JetStreamConfig"> JetStreamConfig </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

Exotic JetStream
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventBusStatus">

EventBusStatus
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBus">EventBus</a>)
</p>

<p>

<p>

EventBusStatus holds the status of the eventbus resource
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

<code>Status</code></br> <em> <a href="#argoproj.io/v1alpha1.Status">
Status </a> </em>
</td>

<td>

<p>

(Members of <code>Status</code> are embedded into this type.)
</p>

</td>

</tr>

<tr>

<td>

<code>config</code></br> <em> <a href="#argoproj.io/v1alpha1.BusConfig">
BusConfig </a> </em>
</td>

<td>

<p>

Config holds the fininalized configuration of EventBus
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventBusType">

EventBusType (<code>string</code> alias)
</p>

</h3>

<p>

<p>

EventBusType is the type of event bus
</p>

</p>

<h3 id="argoproj.io/v1alpha1.EventContext">

EventContext
</h3>

<p>

(<em>Appears on:</em> <a href="#argoproj.io/v1alpha1.Event">Event</a>,
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">EventDependencyFilter</a>)
</p>

<p>

<p>

EventContext holds the context of the cloudevent received from an event
source.
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

ID of the event; must be non-empty and unique within the scope of the
producer.
</p>

</td>

</tr>

<tr>

<td>

<code>source</code></br> <em> string </em>
</td>

<td>

<p>

Source - A URI describing the event producer.
</p>

</td>

</tr>

<tr>

<td>

<code>specversion</code></br> <em> string </em>
</td>

<td>

<p>

SpecVersion - The version of the CloudEvents specification used by the
event.
</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em> string </em>
</td>

<td>

<p>

Type - The type of the occurrence which has happened.
</p>

</td>

</tr>

<tr>

<td>

<code>datacontenttype</code></br> <em> string </em>
</td>

<td>

<p>

DataContentType - A MIME (RFC2046) string describing the media type of
<code>data</code>.
</p>

</td>

</tr>

<tr>

<td>

<code>subject</code></br> <em> string </em>
</td>

<td>

<p>

Subject - The subject of the event in the context of the event producer
</p>

</td>

</tr>

<tr>

<td>

<code>time</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#time-v1-meta">
Kubernetes meta/v1.Time </a> </em>
</td>

<td>

<p>

Time - A Timestamp when the event happened.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventDependency">

EventDependency
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.SensorSpec">SensorSpec</a>)
</p>

<p>

<p>

EventDependency describes a dependency
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

Name is a unique name of this dependency
</p>

</td>

</tr>

<tr>

<td>

<code>eventSourceName</code></br> <em> string </em>
</td>

<td>

<p>

EventSourceName is the name of EventSource that Sensor depends on
</p>

</td>

</tr>

<tr>

<td>

<code>eventName</code></br> <em> string </em>
</td>

<td>

<p>

EventName is the name of the event
</p>

</td>

</tr>

<tr>

<td>

<code>filters</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">
EventDependencyFilter </a> </em>
</td>

<td>

<p>

Filters and rules governing toleration of success and constraints on the
context and data of an event
</p>

</td>

</tr>

<tr>

<td>

<code>transform</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventDependencyTransformer">
EventDependencyTransformer </a> </em>
</td>

<td>

<p>

Transform transforms the event data
</p>

</td>

</tr>

<tr>

<td>

<code>filtersLogicalOperator</code></br> <em>
<a href="#argoproj.io/v1alpha1.LogicalOperator"> LogicalOperator </a>
</em>
</td>

<td>

<p>

FiltersLogicalOperator defines how different filters are evaluated
together. Available values: and (&&), or (\|\|) Is optional and if left
blank treated as and (&&).
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventDependencyFilter">

EventDependencyFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependency">EventDependency</a>)
</p>

<p>

<p>

EventDependencyFilter defines filters and constraints for a event.
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

<code>time</code></br> <em> <a href="#argoproj.io/v1alpha1.TimeFilter">
TimeFilter </a> </em>
</td>

<td>

<p>

Time filter on the event with escalation
</p>

</td>

</tr>

<tr>

<td>

<code>context</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventContext"> EventContext </a> </em>
</td>

<td>

<p>

Context filter constraints
</p>

</td>

</tr>

<tr>

<td>

<code>data</code></br> <em> <a href="#argoproj.io/v1alpha1.DataFilter">
\[\]DataFilter </a> </em>
</td>

<td>

<p>

Data filter constraints with escalation
</p>

</td>

</tr>

<tr>

<td>

<code>exprs</code></br> <em> <a href="#argoproj.io/v1alpha1.ExprFilter">
\[\]ExprFilter </a> </em>
</td>

<td>

<p>

Exprs contains the list of expressions evaluated against the event
payload.
</p>

</td>

</tr>

<tr>

<td>

<code>dataLogicalOperator</code></br> <em>
<a href="#argoproj.io/v1alpha1.LogicalOperator"> LogicalOperator </a>
</em>
</td>

<td>

<p>

DataLogicalOperator defines how multiple Data filters (if defined) are
evaluated together. Available values: and (&&), or (\|\|) Is optional
and if left blank treated as and (&&).
</p>

</td>

</tr>

<tr>

<td>

<code>exprLogicalOperator</code></br> <em>
<a href="#argoproj.io/v1alpha1.LogicalOperator"> LogicalOperator </a>
</em>
</td>

<td>

<p>

ExprLogicalOperator defines how multiple Exprs filters (if defined) are
evaluated together. Available values: and (&&), or (\|\|) Is optional
and if left blank treated as and (&&).
</p>

</td>

</tr>

<tr>

<td>

<code>script</code></br> <em> string </em>
</td>

<td>

<p>

Script refers to a Lua script evaluated to determine the validity of an
event.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventDependencyTransformer">

EventDependencyTransformer
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependency">EventDependency</a>)
</p>

<p>

<p>

EventDependencyTransformer transforms the event
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

<code>jq</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

JQ holds the jq command applied for transformation
</p>

</td>

</tr>

<tr>

<td>

<code>script</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Script refers to a Lua script used to transform the event
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventPersistence">

EventPersistence
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.CalendarEventSource">CalendarEventSource</a>)
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

<code>catchup</code></br> <em>
<a href="#argoproj.io/v1alpha1.CatchupConfiguration">
CatchupConfiguration </a> </em>
</td>

<td>

<p>

Catchup enables to triggered the missed schedule when eventsource
restarts
</p>

</td>

</tr>

<tr>

<td>

<code>configMap</code></br> <em>
<a href="#argoproj.io/v1alpha1.ConfigMapPersistence">
ConfigMapPersistence </a> </em>
</td>

<td>

<p>

ConfigMap holds configmap details for persistence
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventSource">

EventSource
</h3>

<p>

<p>

EventSource is the definition of a eventsource resource
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
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>
</td>

<td>

Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>

</tr>

<tr>

<td>

<code>spec</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec"> EventSourceSpec </a>
</em>
</td>

<td>

<br/> <br/>
<table>

<tr>

<td>

<code>eventBusName</code></br> <em> string </em>
</td>

<td>

<p>

EventBusName references to a EventBus name. By default the value is
“default”
</p>

</td>

</tr>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Template is the pod specification for the event source
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

Service is the specifications of the service to expose the event source
</p>

</td>

</tr>

<tr>

<td>

<code>minio</code></br> <em> <a href="#argoproj.io/v1alpha1.S3Artifact">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.S3Artifact
</a> </em>
</td>

<td>

<p>

Minio event sources
</p>

</td>

</tr>

<tr>

<td>

<code>calendar</code></br> <em>
<a href="#argoproj.io/v1alpha1.CalendarEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.CalendarEventSource
</a> </em>
</td>

<td>

<p>

Calendar event sources
</p>

</td>

</tr>

<tr>

<td>

<code>file</code></br> <em>
<a href="#argoproj.io/v1alpha1.FileEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.FileEventSource
</a> </em>
</td>

<td>

<p>

File event sources
</p>

</td>

</tr>

<tr>

<td>

<code>resource</code></br> <em>
<a href="#argoproj.io/v1alpha1.ResourceEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.ResourceEventSource
</a> </em>
</td>

<td>

<p>

Resource event sources
</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.WebhookEventSource
</a> </em>
</td>

<td>

<p>

Webhook event sources
</p>

</td>

</tr>

<tr>

<td>

<code>amqp</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AMQPEventSource
</a> </em>
</td>

<td>

<p>

AMQP event sources
</p>

</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em>
<a href="#argoproj.io/v1alpha1.KafkaEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.KafkaEventSource
</a> </em>
</td>

<td>

<p>

Kafka event sources
</p>

</td>

</tr>

<tr>

<td>

<code>mqtt</code></br> <em>
<a href="#argoproj.io/v1alpha1.MQTTEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.MQTTEventSource
</a> </em>
</td>

<td>

<p>

MQTT event sources
</p>

</td>

</tr>

<tr>

<td>

<code>nats</code></br> <em>
<a href="#argoproj.io/v1alpha1.NATSEventsSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.NATSEventsSource
</a> </em>
</td>

<td>

<p>

NATS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>sns</code></br> <em>
<a href="#argoproj.io/v1alpha1.SNSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SNSEventSource
</a> </em>
</td>

<td>

<p>

SNS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>sqs</code></br> <em>
<a href="#argoproj.io/v1alpha1.SQSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SQSEventSource
</a> </em>
</td>

<td>

<p>

SQS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>pubSub</code></br> <em>
<a href="#argoproj.io/v1alpha1.PubSubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.PubSubEventSource
</a> </em>
</td>

<td>

<p>

PubSub event sources
</p>

</td>

</tr>

<tr>

<td>

<code>github</code></br> <em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GithubEventSource
</a> </em>
</td>

<td>

<p>

Github event sources
</p>

</td>

</tr>

<tr>

<td>

<code>gitlab</code></br> <em>
<a href="#argoproj.io/v1alpha1.GitlabEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GitlabEventSource
</a> </em>
</td>

<td>

<p>

Gitlab event sources
</p>

</td>

</tr>

<tr>

<td>

<code>hdfs</code></br> <em>
<a href="#argoproj.io/v1alpha1.HDFSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.HDFSEventSource
</a> </em>
</td>

<td>

<p>

HDFS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>slack</code></br> <em>
<a href="#argoproj.io/v1alpha1.SlackEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SlackEventSource
</a> </em>
</td>

<td>

<p>

Slack event sources
</p>

</td>

</tr>

<tr>

<td>

<code>storageGrid</code></br> <em>
<a href="#argoproj.io/v1alpha1.StorageGridEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.StorageGridEventSource
</a> </em>
</td>

<td>

<p>

StorageGrid event sources
</p>

</td>

</tr>

<tr>

<td>

<code>azureEventsHub</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureEventsHubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureEventsHubEventSource
</a> </em>
</td>

<td>

<p>

AzureEventsHub event sources
</p>

</td>

</tr>

<tr>

<td>

<code>stripe</code></br> <em>
<a href="#argoproj.io/v1alpha1.StripeEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.StripeEventSource
</a> </em>
</td>

<td>

<p>

Stripe event sources
</p>

</td>

</tr>

<tr>

<td>

<code>emitter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EmitterEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.EmitterEventSource
</a> </em>
</td>

<td>

<p>

Emitter event source
</p>

</td>

</tr>

<tr>

<td>

<code>redis</code></br> <em>
<a href="#argoproj.io/v1alpha1.RedisEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.RedisEventSource
</a> </em>
</td>

<td>

<p>

Redis event source
</p>

</td>

</tr>

<tr>

<td>

<code>nsq</code></br> <em>
<a href="#argoproj.io/v1alpha1.NSQEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.NSQEventSource
</a> </em>
</td>

<td>

<p>

NSQ event source
</p>

</td>

</tr>

<tr>

<td>

<code>pulsar</code></br> <em>
<a href="#argoproj.io/v1alpha1.PulsarEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.PulsarEventSource
</a> </em>
</td>

<td>

<p>

Pulsar event source
</p>

</td>

</tr>

<tr>

<td>

<code>generic</code></br> <em>
<a href="#argoproj.io/v1alpha1.GenericEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GenericEventSource
</a> </em>
</td>

<td>

<p>

Generic event source
</p>

</td>

</tr>

<tr>

<td>

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

Replicas is the event source deployment replicas
</p>

</td>

</tr>

<tr>

<td>

<code>bitbucketserver</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.BitbucketServerEventSource
</a> </em>
</td>

<td>

<p>

Bitbucket Server event sources
</p>

</td>

</tr>

<tr>

<td>

<code>bitbucket</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.BitbucketEventSource
</a> </em>
</td>

<td>

<p>

Bitbucket event sources
</p>

</td>

</tr>

<tr>

<td>

<code>redisStream</code></br> <em>
<a href="#argoproj.io/v1alpha1.RedisStreamEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.RedisStreamEventSource
</a> </em>
</td>

<td>

<p>

Redis stream source
</p>

</td>

</tr>

<tr>

<td>

<code>azureServiceBus</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureServiceBusEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureServiceBusEventSource
</a> </em>
</td>

<td>

<p>

Azure Service Bus event source
</p>

</td>

</tr>

<tr>

<td>

<code>azureQueueStorage</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureQueueStorageEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureQueueStorageEventSource
</a> </em>
</td>

<td>

<p>

AzureQueueStorage event source
</p>

</td>

</tr>

<tr>

<td>

<code>sftp</code></br> <em>
<a href="#argoproj.io/v1alpha1.SFTPEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SFTPEventSource
</a> </em>
</td>

<td>

<p>

SFTP event sources
</p>

</td>

</tr>

<tr>

<td>

<code>gerrit</code></br> <em>
<a href="#argoproj.io/v1alpha1.GerritEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GerritEventSource
</a> </em>
</td>

<td>

<p>

Gerrit event source
</p>

</td>

</tr>

</table>

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceStatus"> EventSourceStatus
</a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventSourceFilter">

EventSourceFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>,
<a href="#argoproj.io/v1alpha1.AzureEventsHubEventSource">AzureEventsHubEventSource</a>,
<a href="#argoproj.io/v1alpha1.AzureQueueStorageEventSource">AzureQueueStorageEventSource</a>,
<a href="#argoproj.io/v1alpha1.AzureServiceBusEventSource">AzureServiceBusEventSource</a>,
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">BitbucketEventSource</a>,
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">BitbucketServerEventSource</a>,
<a href="#argoproj.io/v1alpha1.CalendarEventSource">CalendarEventSource</a>,
<a href="#argoproj.io/v1alpha1.EmitterEventSource">EmitterEventSource</a>,
<a href="#argoproj.io/v1alpha1.FileEventSource">FileEventSource</a>,
<a href="#argoproj.io/v1alpha1.GenericEventSource">GenericEventSource</a>,
<a href="#argoproj.io/v1alpha1.GerritEventSource">GerritEventSource</a>,
<a href="#argoproj.io/v1alpha1.GithubEventSource">GithubEventSource</a>,
<a href="#argoproj.io/v1alpha1.GitlabEventSource">GitlabEventSource</a>,
<a href="#argoproj.io/v1alpha1.HDFSEventSource">HDFSEventSource</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>,
<a href="#argoproj.io/v1alpha1.MQTTEventSource">MQTTEventSource</a>,
<a href="#argoproj.io/v1alpha1.NATSEventsSource">NATSEventsSource</a>,
<a href="#argoproj.io/v1alpha1.NSQEventSource">NSQEventSource</a>,
<a href="#argoproj.io/v1alpha1.PubSubEventSource">PubSubEventSource</a>,
<a href="#argoproj.io/v1alpha1.PulsarEventSource">PulsarEventSource</a>,
<a href="#argoproj.io/v1alpha1.RedisEventSource">RedisEventSource</a>,
<a href="#argoproj.io/v1alpha1.RedisStreamEventSource">RedisStreamEventSource</a>,
<a href="#argoproj.io/v1alpha1.SFTPEventSource">SFTPEventSource</a>,
<a href="#argoproj.io/v1alpha1.SNSEventSource">SNSEventSource</a>,
<a href="#argoproj.io/v1alpha1.SQSEventSource">SQSEventSource</a>,
<a href="#argoproj.io/v1alpha1.SlackEventSource">SlackEventSource</a>,
<a href="#argoproj.io/v1alpha1.WebhookEventSource">WebhookEventSource</a>)
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

<code>expression</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventSourceSpec">

EventSourceSpec
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSource">EventSource</a>)
</p>

<p>

<p>

EventSourceSpec refers to specification of event-source resource
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

<code>eventBusName</code></br> <em> string </em>
</td>

<td>

<p>

EventBusName references to a EventBus name. By default the value is
“default”
</p>

</td>

</tr>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Template is the pod specification for the event source
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

Service is the specifications of the service to expose the event source
</p>

</td>

</tr>

<tr>

<td>

<code>minio</code></br> <em> <a href="#argoproj.io/v1alpha1.S3Artifact">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.S3Artifact
</a> </em>
</td>

<td>

<p>

Minio event sources
</p>

</td>

</tr>

<tr>

<td>

<code>calendar</code></br> <em>
<a href="#argoproj.io/v1alpha1.CalendarEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.CalendarEventSource
</a> </em>
</td>

<td>

<p>

Calendar event sources
</p>

</td>

</tr>

<tr>

<td>

<code>file</code></br> <em>
<a href="#argoproj.io/v1alpha1.FileEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.FileEventSource
</a> </em>
</td>

<td>

<p>

File event sources
</p>

</td>

</tr>

<tr>

<td>

<code>resource</code></br> <em>
<a href="#argoproj.io/v1alpha1.ResourceEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.ResourceEventSource
</a> </em>
</td>

<td>

<p>

Resource event sources
</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.WebhookEventSource
</a> </em>
</td>

<td>

<p>

Webhook event sources
</p>

</td>

</tr>

<tr>

<td>

<code>amqp</code></br> <em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AMQPEventSource
</a> </em>
</td>

<td>

<p>

AMQP event sources
</p>

</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em>
<a href="#argoproj.io/v1alpha1.KafkaEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.KafkaEventSource
</a> </em>
</td>

<td>

<p>

Kafka event sources
</p>

</td>

</tr>

<tr>

<td>

<code>mqtt</code></br> <em>
<a href="#argoproj.io/v1alpha1.MQTTEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.MQTTEventSource
</a> </em>
</td>

<td>

<p>

MQTT event sources
</p>

</td>

</tr>

<tr>

<td>

<code>nats</code></br> <em>
<a href="#argoproj.io/v1alpha1.NATSEventsSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.NATSEventsSource
</a> </em>
</td>

<td>

<p>

NATS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>sns</code></br> <em>
<a href="#argoproj.io/v1alpha1.SNSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SNSEventSource
</a> </em>
</td>

<td>

<p>

SNS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>sqs</code></br> <em>
<a href="#argoproj.io/v1alpha1.SQSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SQSEventSource
</a> </em>
</td>

<td>

<p>

SQS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>pubSub</code></br> <em>
<a href="#argoproj.io/v1alpha1.PubSubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.PubSubEventSource
</a> </em>
</td>

<td>

<p>

PubSub event sources
</p>

</td>

</tr>

<tr>

<td>

<code>github</code></br> <em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GithubEventSource
</a> </em>
</td>

<td>

<p>

Github event sources
</p>

</td>

</tr>

<tr>

<td>

<code>gitlab</code></br> <em>
<a href="#argoproj.io/v1alpha1.GitlabEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GitlabEventSource
</a> </em>
</td>

<td>

<p>

Gitlab event sources
</p>

</td>

</tr>

<tr>

<td>

<code>hdfs</code></br> <em>
<a href="#argoproj.io/v1alpha1.HDFSEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.HDFSEventSource
</a> </em>
</td>

<td>

<p>

HDFS event sources
</p>

</td>

</tr>

<tr>

<td>

<code>slack</code></br> <em>
<a href="#argoproj.io/v1alpha1.SlackEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SlackEventSource
</a> </em>
</td>

<td>

<p>

Slack event sources
</p>

</td>

</tr>

<tr>

<td>

<code>storageGrid</code></br> <em>
<a href="#argoproj.io/v1alpha1.StorageGridEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.StorageGridEventSource
</a> </em>
</td>

<td>

<p>

StorageGrid event sources
</p>

</td>

</tr>

<tr>

<td>

<code>azureEventsHub</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureEventsHubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureEventsHubEventSource
</a> </em>
</td>

<td>

<p>

AzureEventsHub event sources
</p>

</td>

</tr>

<tr>

<td>

<code>stripe</code></br> <em>
<a href="#argoproj.io/v1alpha1.StripeEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.StripeEventSource
</a> </em>
</td>

<td>

<p>

Stripe event sources
</p>

</td>

</tr>

<tr>

<td>

<code>emitter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EmitterEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.EmitterEventSource
</a> </em>
</td>

<td>

<p>

Emitter event source
</p>

</td>

</tr>

<tr>

<td>

<code>redis</code></br> <em>
<a href="#argoproj.io/v1alpha1.RedisEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.RedisEventSource
</a> </em>
</td>

<td>

<p>

Redis event source
</p>

</td>

</tr>

<tr>

<td>

<code>nsq</code></br> <em>
<a href="#argoproj.io/v1alpha1.NSQEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.NSQEventSource
</a> </em>
</td>

<td>

<p>

NSQ event source
</p>

</td>

</tr>

<tr>

<td>

<code>pulsar</code></br> <em>
<a href="#argoproj.io/v1alpha1.PulsarEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.PulsarEventSource
</a> </em>
</td>

<td>

<p>

Pulsar event source
</p>

</td>

</tr>

<tr>

<td>

<code>generic</code></br> <em>
<a href="#argoproj.io/v1alpha1.GenericEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GenericEventSource
</a> </em>
</td>

<td>

<p>

Generic event source
</p>

</td>

</tr>

<tr>

<td>

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

Replicas is the event source deployment replicas
</p>

</td>

</tr>

<tr>

<td>

<code>bitbucketserver</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.BitbucketServerEventSource
</a> </em>
</td>

<td>

<p>

Bitbucket Server event sources
</p>

</td>

</tr>

<tr>

<td>

<code>bitbucket</code></br> <em>
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.BitbucketEventSource
</a> </em>
</td>

<td>

<p>

Bitbucket event sources
</p>

</td>

</tr>

<tr>

<td>

<code>redisStream</code></br> <em>
<a href="#argoproj.io/v1alpha1.RedisStreamEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.RedisStreamEventSource
</a> </em>
</td>

<td>

<p>

Redis stream source
</p>

</td>

</tr>

<tr>

<td>

<code>azureServiceBus</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureServiceBusEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureServiceBusEventSource
</a> </em>
</td>

<td>

<p>

Azure Service Bus event source
</p>

</td>

</tr>

<tr>

<td>

<code>azureQueueStorage</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureQueueStorageEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.AzureQueueStorageEventSource
</a> </em>
</td>

<td>

<p>

AzureQueueStorage event source
</p>

</td>

</tr>

<tr>

<td>

<code>sftp</code></br> <em>
<a href="#argoproj.io/v1alpha1.SFTPEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SFTPEventSource
</a> </em>
</td>

<td>

<p>

SFTP event sources
</p>

</td>

</tr>

<tr>

<td>

<code>gerrit</code></br> <em>
<a href="#argoproj.io/v1alpha1.GerritEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.GerritEventSource
</a> </em>
</td>

<td>

<p>

Gerrit event source
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventSourceStatus">

EventSourceStatus
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSource">EventSource</a>)
</p>

<p>

<p>

EventSourceStatus holds the status of the event-source resource
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

<code>Status</code></br> <em> <a href="#argoproj.io/v1alpha1.Status">
Status </a> </em>
</td>

<td>

<p>

(Members of <code>Status</code> are embedded into this type.)
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.EventSourceType">

EventSourceType (<code>string</code> alias)
</p>

</h3>

<p>

<p>

EventSourceType is the type of event source
</p>

</p>

<h3 id="argoproj.io/v1alpha1.ExprFilter">

ExprFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">EventDependencyFilter</a>)
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

<code>expr</code></br> <em> string </em>
</td>

<td>

<p>

Expr refers to the expression that determines the outcome of the filter.
</p>

</td>

</tr>

<tr>

<td>

<code>fields</code></br> <em>
<a href="#argoproj.io/v1alpha1.PayloadField"> \[\]PayloadField </a>
</em>
</td>

<td>

<p>

Fields refers to set of keys that refer to the paths within event
payload.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.FileArtifact">

FileArtifact
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation">ArtifactLocation</a>)
</p>

<p>

<p>

FileArtifact contains information about an artifact in a filesystem
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

<code>path</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.FileEventSource">

FileEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

FileEventSource describes an event-source for file related events.
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

<code>eventType</code></br> <em> string </em>
</td>

<td>

<p>

Type of file operations to watch Refer
<a href="https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go">https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go</a>
for more information
</p>

</td>

</tr>

<tr>

<td>

<code>watchPathConfig</code></br> <em>
<a href="#argoproj.io/v1alpha1.WatchPathConfig"> WatchPathConfig </a>
</em>
</td>

<td>

<p>

WatchPathConfig contains configuration about the file path to watch
</p>

</td>

</tr>

<tr>

<td>

<code>polling</code></br> <em> bool </em>
</td>

<td>

<p>

Use polling instead of inotify
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GenericEventSource">

GenericEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

GenericEventSource refers to a generic event source. It can be used to
implement a custom event source.
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL of the gRPC server that implements the event source.
</p>

</td>

</tr>

<tr>

<td>

<code>config</code></br> <em> string </em>
</td>

<td>

<p>

Config is the event source configuration
</p>

</td>

</tr>

<tr>

<td>

<code>insecure</code></br> <em> bool </em>
</td>

<td>

<p>

Insecure determines the type of connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>authSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AuthSecret holds a secret selector that contains a bearer token for
authentication
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GerritEventSource">

GerritEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

GerritEventSource refers to event-source related to gerrit events
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration to run a http server
</p>

</td>

</tr>

<tr>

<td>

<code>hookName</code></br> <em> string </em>
</td>

<td>

<p>

HookName is the name of the webhook
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Events are gerrit event to listen to. Refer
<a href="https://gerrit-review.googlesource.com/Documentation/cmd-stream-events.html#events">https://gerrit-review.googlesource.com/Documentation/cmd-stream-events.html#events</a>
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em> <a href="#argoproj.io/v1alpha1.BasicAuth">
BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Auth hosts secret selectors for username and password
</p>

</td>

</tr>

<tr>

<td>

<code>gerritBaseURL</code></br> <em> string </em>
</td>

<td>

<p>

GerritBaseURL is the base URL for API requests to a custom endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>deleteHookOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteHookOnFinish determines whether to delete the Gerrit hook for the
project once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>projects</code></br> <em> \[\]string </em>
</td>

<td>

<p>

List of project namespace paths like “whynowy/test”.
</p>

</td>

</tr>

<tr>

<td>

<code>sslVerify</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

SslVerify to enable ssl verification
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GitArtifact">

GitArtifact
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation">ArtifactLocation</a>)
</p>

<p>

<p>

GitArtifact contains information about an artifact stored in git
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

Git URL
</p>

</td>

</tr>

<tr>

<td>

<code>cloneDirectory</code></br> <em> string </em>
</td>

<td>

<p>

Directory to clone the repository. We clone complete directory because
GitArtifact is not limited to any specific Git service providers. Hence
we don’t use any specific git provider client.
</p>

</td>

</tr>

<tr>

<td>

<code>creds</code></br> <em> <a href="#argoproj.io/v1alpha1.GitCreds">
GitCreds </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Creds contain reference to git username and password
</p>

</td>

</tr>

<tr>

<td>

<code>sshKeySecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SSHKeySecret refers to the secret that contains SSH key
</p>

</td>

</tr>

<tr>

<td>

<code>filePath</code></br> <em> string </em>
</td>

<td>

<p>

Path to file that contains trigger resource definition
</p>

</td>

</tr>

<tr>

<td>

<code>branch</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Branch to use to pull trigger resource
</p>

</td>

</tr>

<tr>

<td>

<code>tag</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Tag to use to pull trigger resource
</p>

</td>

</tr>

<tr>

<td>

<code>ref</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Ref to use to pull trigger resource. Will result in a shallow clone and
fetch.
</p>

</td>

</tr>

<tr>

<td>

<code>remote</code></br> <em>
<a href="#argoproj.io/v1alpha1.GitRemoteConfig"> GitRemoteConfig </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

Remote to manage set of tracked repositories. Defaults to “origin”.
Refer
<a href="https://git-scm.com/docs/git-remote">https://git-scm.com/docs/git-remote</a>
</p>

</td>

</tr>

<tr>

<td>

<code>insecureIgnoreHostKey</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Whether to ignore host key
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GitCreds">

GitCreds
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GitArtifact">GitArtifact</a>)
</p>

<p>

<p>

GitCreds contain reference to git username and password
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

<code>username</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GitRemoteConfig">

GitRemoteConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GitArtifact">GitArtifact</a>)
</p>

<p>

<p>

GitRemoteConfig contains the configuration of a Git remote
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

Name of the remote to fetch from.
</p>

</td>

</tr>

<tr>

<td>

<code>urls</code></br> <em> \[\]string </em>
</td>

<td>

<p>

URLs the URLs of a remote repository. It must be non-empty. Fetch will
always use the first URL, while push will use all of them.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GithubAppCreds">

GithubAppCreds
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">GithubEventSource</a>)
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

<code>privateKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

PrivateKey refers to a K8s secret containing the GitHub app private key
</p>

</td>

</tr>

<tr>

<td>

<code>appID</code></br> <em> int64 </em>
</td>

<td>

<p>

AppID refers to the GitHub App ID for the application you created
</p>

</td>

</tr>

<tr>

<td>

<code>installationID</code></br> <em> int64 </em>
</td>

<td>

<p>

InstallationID refers to the Installation ID of the GitHub app you
created and installed
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GithubEventSource">

GithubEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

GithubEventSource refers to event-source for github related events
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

<code>id</code></br> <em> int64 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Id is the webhook’s id Deprecated: This is not used at all, will be
removed in v1.6
</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook refers to the configuration required to run a http server
</p>

</td>

</tr>

<tr>

<td>

<code>owner</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedOwner refers to GitHub owner name i.e. argoproj Deprecated:
use Repositories instead. Will be unsupported in v 1.6
</p>

</td>

</tr>

<tr>

<td>

<code>repository</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedRepository refers to GitHub repo name i.e. argo-events
Deprecated: use Repositories instead. Will be unsupported in v 1.6
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Events refer to Github events to which the event source will subscribe
</p>

</td>

</tr>

<tr>

<td>

<code>apiToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

APIToken refers to a K8s secret containing github api token
</p>

</td>

</tr>

<tr>

<td>

<code>webhookSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

WebhookSecret refers to K8s secret containing GitHub webhook secret
<a href="https://developer.github.com/webhooks/securing/">https://developer.github.com/webhooks/securing/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>insecure</code></br> <em> bool </em>
</td>

<td>

<p>

Insecure tls verification
</p>

</td>

</tr>

<tr>

<td>

<code>active</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Active refers to status of the webhook for event deliveries.
<a href="https://developer.github.com/webhooks/creating/#active">https://developer.github.com/webhooks/creating/#active</a>
</p>

</td>

</tr>

<tr>

<td>

<code>contentType</code></br> <em> string </em>
</td>

<td>

<p>

ContentType of the event delivery
</p>

</td>

</tr>

<tr>

<td>

<code>githubBaseURL</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

GitHub base URL (for GitHub Enterprise)
</p>

</td>

</tr>

<tr>

<td>

<code>githubUploadURL</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

GitHub upload URL (for GitHub Enterprise)
</p>

</td>

</tr>

<tr>

<td>

<code>deleteHookOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteHookOnFinish determines whether to delete the GitHub hook for the
repository once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>repositories</code></br> <em>
<a href="#argoproj.io/v1alpha1.OwnedRepositories"> \[\]OwnedRepositories
</a> </em>
</td>

<td>

<p>

Repositories holds the information of repositories, which uses repo
owner as the key, and list of repo names as the value. Not required if
Organizations is set.
</p>

</td>

</tr>

<tr>

<td>

<code>organizations</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Organizations holds the names of organizations (used for organization
level webhooks). Not required if Repositories is set.
</p>

</td>

</tr>

<tr>

<td>

<code>githubApp</code></br> <em>
<a href="#argoproj.io/v1alpha1.GithubAppCreds"> GithubAppCreds </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

GitHubApp holds the GitHub app credentials
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.GitlabEventSource">

GitlabEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

GitlabEventSource refers to event-source related to Gitlab events
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration to run a http server
</p>

</td>

</tr>

<tr>

<td>

<code>projectID</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeprecatedProjectID is the id of project for which integration needs to
setup Deprecated: use Projects instead. Will be unsupported in v 1.7
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Events are gitlab event to listen to. Refer
<a href="https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794">https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794</a>.
</p>

</td>

</tr>

<tr>

<td>

<code>accessToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

AccessToken references to k8 secret which holds the gitlab api access
information
</p>

</td>

</tr>

<tr>

<td>

<code>enableSSLVerification</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

EnableSSLVerification to enable ssl verification
</p>

</td>

</tr>

<tr>

<td>

<code>gitlabBaseURL</code></br> <em> string </em>
</td>

<td>

<p>

GitlabBaseURL is the base URL for API requests to a custom endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>deleteHookOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteHookOnFinish determines whether to delete the GitLab hook for the
project once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>projects</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

List of project IDs or project namespace paths like “whynowy/test”.
Projects and groups cannot be empty at the same time.
</p>

</td>

</tr>

<tr>

<td>

<code>secretToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SecretToken references to k8 secret which holds the Secret Token used by
webhook config
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>groups</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

List of group IDs or group name like “test”. Group level hook available
in Premium and Ultimate Gitlab.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.HDFSEventSource">

HDFSEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

HDFSEventSource refers to event-source for HDFS related events
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

<code>WatchPathConfig</code></br> <em>
<a href="#argoproj.io/v1alpha1.WatchPathConfig"> WatchPathConfig </a>
</em>
</td>

<td>

<p>

(Members of <code>WatchPathConfig</code> are embedded into this type.)
</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em> string </em>
</td>

<td>

<p>

Type of file operations to watch
</p>

</td>

</tr>

<tr>

<td>

<code>checkInterval</code></br> <em> string </em>
</td>

<td>

<p>

CheckInterval is a string that describes an interval duration to check
the directory state, e.g. 1s, 30m, 2h… (defaults to 1m)
</p>

</td>

</tr>

<tr>

<td>

<code>addresses</code></br> <em> \[\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>hdfsUser</code></br> <em> string </em>
</td>

<td>

<p>

HDFSUser is the user to access HDFS file system. It is ignored if either
ccache or keytab is used.
</p>

</td>

</tr>

<tr>

<td>

<code>krbCCacheSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

KrbCCacheSecret is the secret selector for Kerberos ccache Either ccache
or keytab can be set to use Kerberos.
</p>

</td>

</tr>

<tr>

<td>

<code>krbKeytabSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

KrbKeytabSecret is the secret selector for Kerberos keytab Either ccache
or keytab can be set to use Kerberos.
</p>

</td>

</tr>

<tr>

<td>

<code>krbUsername</code></br> <em> string </em>
</td>

<td>

<p>

KrbUsername is the Kerberos username used with Kerberos keytab It must
be set if keytab is used.
</p>

</td>

</tr>

<tr>

<td>

<code>krbRealm</code></br> <em> string </em>
</td>

<td>

<p>

KrbRealm is the Kerberos realm used with Kerberos keytab It must be set
if keytab is used.
</p>

</td>

</tr>

<tr>

<td>

<code>krbConfigConfigMap</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#configmapkeyselector-v1-core">
Kubernetes core/v1.ConfigMapKeySelector </a> </em>
</td>

<td>

<p>

KrbConfig is the configmap selector for Kerberos config as string It
must be set if either ccache or keytab is used.
</p>

</td>

</tr>

<tr>

<td>

<code>krbServicePrincipalName</code></br> <em> string </em>
</td>

<td>

<p>

KrbServicePrincipalName is the principal name of Kerberos service It
must be set if either ccache or keytab is used.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.HTTPTrigger">

HTTPTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

HTTPTrigger is the trigger for the HTTP request
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL refers to the URL to send HTTP request to.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the HTTP client.
</p>

</td>

</tr>

<tr>

<td>

<code>method</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Method refers to the type of the HTTP request. Refer
<a href="https://golang.org/src/net/http/method.go">https://golang.org/src/net/http/method.go</a>
for more info. Default value is POST.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the HTTP trigger resource.
</p>

</td>

</tr>

<tr>

<td>

<code>timeout</code></br> <em> int64 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Timeout refers to the HTTP request timeout in seconds. Default value is
60 seconds.
</p>

</td>

</tr>

<tr>

<td>

<code>basicAuth</code></br> <em>
<a href="#argoproj.io/v1alpha1.BasicAuth"> BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

BasicAuth configuration for the http request.
</p>

</td>

</tr>

<tr>

<td>

<code>headers</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Headers for the HTTP request.
</p>

</td>

</tr>

<tr>

<td>

<code>secureHeaders</code></br> <em>
<a href="#argoproj.io/v1alpha1.*github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SecureHeader">
\[\]\*github.com/argoproj/argo-events/pkg/apis/events/v1alpha1.SecureHeader
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Secure Headers stored in Kubernetes Secrets for the HTTP requests.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Int64OrString">

Int64OrString
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Backoff">Backoff</a>)
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

<code>type</code></br> <em> <a href="#argoproj.io/v1alpha1.Type"> Type
</a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>int64Val</code></br> <em> int64 </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>strVal</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.JSONType">

JSONType (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.DataFilter">DataFilter</a>)
</p>

<p>

<p>

JSONType contains the supported JSON types for data filtering
</p>

</p>

<h3 id="argoproj.io/v1alpha1.JetStreamBus">

JetStreamBus
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBusSpec">EventBusSpec</a>)
</p>

<p>

<p>

JetStreamBus holds the JetStream EventBus information
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

<code>version</code></br> <em> string </em>
</td>

<td>

<p>

JetStream version, such as “2.7.3”
</p>

</td>

</tr>

<tr>

<td>

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

JetStream StatefulSet size
</p>

</td>

</tr>

<tr>

<td>

<code>containerTemplate</code></br> <em>
<a href="#argoproj.io/v1alpha1.ContainerTemplate"> ContainerTemplate
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ContainerTemplate contains customized spec for Nats JetStream container
</p>

</td>

</tr>

<tr>

<td>

<code>reloaderContainerTemplate</code></br> <em>
<a href="#argoproj.io/v1alpha1.ContainerTemplate"> ContainerTemplate
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ReloaderContainerTemplate contains customized spec for config reloader
container
</p>

</td>

</tr>

<tr>

<td>

<code>metricsContainerTemplate</code></br> <em>
<a href="#argoproj.io/v1alpha1.ContainerTemplate"> ContainerTemplate
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

MetricsContainerTemplate contains customized spec for metrics container
</p>

</td>

</tr>

<tr>

<td>

<code>persistence</code></br> <em>
<a href="#argoproj.io/v1alpha1.PersistenceStrategy"> PersistenceStrategy
</a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em>
<a href="#argoproj.io/v1alpha1.Metadata"> Metadata </a> </em>
</td>

<td>

<p>

Metadata sets the pods’s metadata, i.e. annotations and labels
</p>

</td>

</tr>

<tr>

<td>

<code>nodeSelector</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

NodeSelector is a selector which must be true for the pod to fit on a
node. Selector which must match a node’s labels for the pod to be
scheduled on that node. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/assign-pod-node/">https://kubernetes.io/docs/concepts/configuration/assign-pod-node/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>tolerations</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#toleration-v1-core">
\[\]Kubernetes core/v1.Toleration </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, the pod’s tolerations.
</p>

</td>

</tr>

<tr>

<td>

<code>securityContext</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#podsecuritycontext-v1-core">
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

<tr>

<td>

<code>imagePullSecrets</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#localobjectreference-v1-core">
\[\]Kubernetes core/v1.LocalObjectReference </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ImagePullSecrets is an optional list of references to secrets in the
same namespace to use for pulling any of the images used by this
PodSpec. If specified, these secrets will be passed to individual puller
implementations for them to use. For example, in the case of docker,
only DockerConfig type secrets are honored. More info:
<a href="https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod">https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod</a>
</p>

</td>

</tr>

<tr>

<td>

<code>priorityClassName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, indicates the Redis pod’s priority. “system-node-critical”
and “system-cluster-critical” are two special keywords which indicate
the highest priorities with the former being the highest priority. Any
other name must be defined by creating a PriorityClass object with that
name. If not specified, the pod priority will be default or zero if
there is no default. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>priority</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

The priority value. Various system components use this field to find the
priority of the Redis pod. When Priority Admission Controller is
enabled, it prevents users from setting this field. The admission
controller populates this field from PriorityClassName. The higher the
value, the higher the priority. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>affinity</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#affinity-v1-core">
Kubernetes core/v1.Affinity </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

The pod’s scheduling constraints More info:
<a href="https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/">https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>serviceAccountName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ServiceAccountName to apply to the StatefulSet
</p>

</td>

</tr>

<tr>

<td>

<code>settings</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

JetStream configuration, if not specified, global settings in
controller-config will be used. See
<a href="https://docs.nats.io/running-a-nats-service/configuration#jetstream">https://docs.nats.io/running-a-nats-service/configuration#jetstream</a>.
Only configure “max_memory_store” or “max_file_store”, do not set
“store_dir” as it has been hardcoded.
</p>

</td>

</tr>

<tr>

<td>

<code>startArgs</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Optional arguments to start nats-server. For example, “-D” to enable
debugging output, “-DV” to enable debugging and tracing. Check
<a href="https://docs.nats.io/">https://docs.nats.io/</a> for all the
available arguments.
</p>

</td>

</tr>

<tr>

<td>

<code>streamConfig</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Optional configuration for the streams to be created in this JetStream
service, if specified, it will be merged with the default configuration
in controller-config. It accepts a YAML format configuration, available
fields include, “maxBytes”, “maxMsgs”, “maxAge” (e.g. 72h), “replicas”
(1, 3, 5), “duplicates” (e.g. 5m), “retention” (e.g. 0: Limits
(default), 1: Interest, 2: WorkQueue), “Discard” (e.g. 0: DiscardOld
(default), 1: DiscardNew).
</p>

</td>

</tr>

<tr>

<td>

<code>maxPayload</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Maximum number of bytes in a message payload, 0 means unlimited.
Defaults to 1MB
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.JetStreamConfig">

JetStreamConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BusConfig">BusConfig</a>,
<a href="#argoproj.io/v1alpha1.EventBusSpec">EventBusSpec</a>)
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

JetStream (Nats) URL
</p>

</td>

</tr>

<tr>

<td>

<code>accessSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Secret for auth
</p>

</td>

</tr>

<tr>

<td>

<code>streamConfig</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.K8SResource">

K8SResource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation">ArtifactLocation</a>)
</p>

<p>

<p>

K8SResource represent arbitrary structured data.
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

<code>value</code></br> <em> \[\]byte </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.K8SResourcePolicy">

K8SResourcePolicy
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerPolicy">TriggerPolicy</a>)
</p>

<p>

<p>

K8SResourcePolicy refers to the policy used to check the state of K8s
based triggers using labels
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

<code>labels</code></br> <em> map\[string\]string </em>
</td>

<td>

<p>

Labels required to identify whether a resource is in success state
</p>

</td>

</tr>

<tr>

<td>

<code>backoff</code></br> <em> <a href="#argoproj.io/v1alpha1.Backoff">
Backoff </a> </em>
</td>

<td>

<p>

Backoff before checking resource state
</p>

</td>

</tr>

<tr>

<td>

<code>errorOnBackoffTimeout</code></br> <em> bool </em>
</td>

<td>

<p>

ErrorOnBackoffTimeout determines whether sensor should transition to
error state if the trigger policy is unable to determine the state of
the resource
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.KafkaBus">

KafkaBus
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BusConfig">BusConfig</a>,
<a href="#argoproj.io/v1alpha1.EventBusSpec">EventBusSpec</a>)
</p>

<p>

<p>

KafkaBus holds the KafkaBus EventBus information
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL to kafka cluster, multiple URLs separated by comma
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Topic name, defaults to {namespace_name}-{eventbus_name}
</p>

</td>

</tr>

<tr>

<td>

<code>version</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Kafka version, sarama defaults to the oldest supported stable version
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the kafka client.
</p>

</td>

</tr>

<tr>

<td>

<code>sasl</code></br> <em> <a href="#argoproj.io/v1alpha1.SASLConfig">
SASLConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SASL configuration for the kafka client
</p>

</td>

</tr>

<tr>

<td>

<code>consumerGroup</code></br> <em>
<a href="#argoproj.io/v1alpha1.KafkaConsumerGroup"> KafkaConsumerGroup
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Consumer group for kafka client
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.KafkaConsumerGroup">

KafkaConsumerGroup
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.KafkaBus">KafkaBus</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>)
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

<code>groupName</code></br> <em> string </em>
</td>

<td>

<p>

The name for the consumer group to use
</p>

</td>

</tr>

<tr>

<td>

<code>oldest</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

When starting up a new group do we want to start from the oldest event
(true) or the newest event (false), defaults to false
</p>

</td>

</tr>

<tr>

<td>

<code>rebalanceStrategy</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Rebalance strategy can be one of: sticky, roundrobin, range. Range is
the default.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.KafkaEventSource">

KafkaEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

KafkaEventSource refers to event-source for Kafka related events
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL to kafka cluster, multiple URLs separated by comma
</p>

</td>

</tr>

<tr>

<td>

<code>partition</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Partition name
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<p>

Topic name
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the kafka client.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>consumerGroup</code></br> <em>
<a href="#argoproj.io/v1alpha1.KafkaConsumerGroup"> KafkaConsumerGroup
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Consumer group for kafka client
</p>

</td>

</tr>

<tr>

<td>

<code>limitEventsPerSecond</code></br> <em> int64 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Sets a limit on how many events get read from kafka per second.
</p>

</td>

</tr>

<tr>

<td>

<code>version</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Specify what kafka version is being connected to enables certain
features in sarama, defaults to 1.0.0
</p>

</td>

</tr>

<tr>

<td>

<code>sasl</code></br> <em> <a href="#argoproj.io/v1alpha1.SASLConfig">
SASLConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SASL configuration for the kafka client
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>config</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Yaml format Sarama config for Kafka connection. It follows the struct of
sarama.Config. See
<a href="https://github.com/IBM/sarama/blob/main/config.go">https://github.com/IBM/sarama/blob/main/config.go</a>
e.g.
</p>

<p>

consumer: fetch: min: 1 net: MaxOpenRequests: 5
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.KafkaTrigger">

KafkaTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

KafkaTrigger refers to the specification of the Kafka trigger.
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL of the Kafka broker, multiple URLs separated by comma.
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<p>

Name of the topic. More info at
<a href="https://kafka.apache.org/documentation/#intro_topics">https://kafka.apache.org/documentation/#intro_topics</a>
</p>

</td>

</tr>

<tr>

<td>

<code>partition</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

DEPRECATED
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters that is applied to resolved Kafka
trigger object.
</p>

</td>

</tr>

<tr>

<td>

<code>requiredAcks</code></br> <em> int32 </em>
</td>

<td>

<p>

RequiredAcks used in producer to tell the broker how many replica
acknowledgements Defaults to 1 (Only wait for the leader to ack).
</p>

</td>

</tr>

<tr>

<td>

<code>compress</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Compress determines whether to compress message or not. Defaults to
false. If set to true, compresses message using snappy compression.
</p>

</td>

</tr>

<tr>

<td>

<code>flushFrequency</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

FlushFrequency refers to the frequency in milliseconds to flush batches.
Defaults to 500 milliseconds.
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the Kafka producer.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>partitioningKey</code></br> <em> string </em>
</td>

<td>

<p>

The partitioning key for the messages put on the Kafka topic.
</p>

</td>

</tr>

<tr>

<td>

<code>version</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Specify what kafka version is being connected to enables certain
features in sarama, defaults to 1.0.0
</p>

</td>

</tr>

<tr>

<td>

<code>sasl</code></br> <em> <a href="#argoproj.io/v1alpha1.SASLConfig">
SASLConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SASL configuration for the kafka client
</p>

</td>

</tr>

<tr>

<td>

<code>schemaRegistry</code></br> <em>
<a href="#argoproj.io/v1alpha1.SchemaRegistryConfig">
SchemaRegistryConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Schema Registry configuration to producer message with avro format
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.KubernetesResourceOperation">

KubernetesResourceOperation (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.StandardK8STrigger">StandardK8STrigger</a>)
</p>

<p>

<p>

KubernetesResourceOperation refers to the type of operation performed on
the K8s resource
</p>

</p>

<h3 id="argoproj.io/v1alpha1.LogTrigger">

LogTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
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

<code>intervalSeconds</code></br> <em> uint64 </em>
</td>

<td>

<em>(Optional)</em>
<p>

Only print messages every interval. Useful to prevent logging too much
data for busy events.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.LogicalOperator">

LogicalOperator (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependency">EventDependency</a>,
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">EventDependencyFilter</a>)
</p>

<p>

</p>

<h3 id="argoproj.io/v1alpha1.MQTTEventSource">

MQTTEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

MQTTEventSource refers to event-source for MQTT related events
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL to connect to broker
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<p>

Topic name
</p>

</td>

</tr>

<tr>

<td>

<code>clientId</code></br> <em> string </em>
</td>

<td>

<p>

ClientID is the id of the client
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<p>

ConnectionBackoff holds backoff applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the mqtt client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em> <a href="#argoproj.io/v1alpha1.BasicAuth">
BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Auth hosts secret selectors for username and password
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Metadata">

Metadata
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.JetStreamBus">JetStreamBus</a>,
<a href="#argoproj.io/v1alpha1.NativeStrategy">NativeStrategy</a>,
<a href="#argoproj.io/v1alpha1.Template">Template</a>)
</p>

<p>

<p>

Metadata holds the annotations and labels of an event source pod
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

<code>annotations</code></br> <em> map\[string\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>labels</code></br> <em> map\[string\]string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSAuth">

NATSAuth
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.NATSEventsSource">NATSEventsSource</a>)
</p>

<p>

<p>

NATSAuth refers to the auth info for NATS EventSource
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

<code>basic</code></br> <em> <a href="#argoproj.io/v1alpha1.BasicAuth">
BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Baisc auth with username and password
</p>

</td>

</tr>

<tr>

<td>

<code>token</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Token used to connect
</p>

</td>

</tr>

<tr>

<td>

<code>nkey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

NKey used to connect
</p>

</td>

</tr>

<tr>

<td>

<code>credential</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

credential used to connect
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSBus">

NATSBus
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBusSpec">EventBusSpec</a>)
</p>

<p>

<p>

NATSBus holds the NATS eventbus information
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

<code>native</code></br> <em>
<a href="#argoproj.io/v1alpha1.NativeStrategy"> NativeStrategy </a>
</em>
</td>

<td>

<p>

Native means to bring up a native NATS service
</p>

</td>

</tr>

<tr>

<td>

<code>exotic</code></br> <em>
<a href="#argoproj.io/v1alpha1.NATSConfig"> NATSConfig </a> </em>
</td>

<td>

<p>

Exotic holds an exotic NATS config
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSConfig">

NATSConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BusConfig">BusConfig</a>,
<a href="#argoproj.io/v1alpha1.NATSBus">NATSBus</a>)
</p>

<p>

<p>

NATSConfig holds the config of NATS
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

NATS streaming url
</p>

</td>

</tr>

<tr>

<td>

<code>clusterID</code></br> <em> string </em>
</td>

<td>

<p>

Cluster ID for nats streaming
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em>
<a href="#argoproj.io/v1alpha1.AuthStrategy"> AuthStrategy </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Auth strategy, default to AuthStrategyNone
</p>

</td>

</tr>

<tr>

<td>

<code>accessSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Secret for auth
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSEventsSource">

NATSEventsSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

NATSEventsSource refers to event-source for NATS related events
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL to connect to NATS cluster
</p>

</td>

</tr>

<tr>

<td>

<code>subject</code></br> <em> string </em>
</td>

<td>

<p>

Subject holds the name of the subject onto which messages are published
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<p>

ConnectionBackoff holds backoff applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the nats client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em> <a href="#argoproj.io/v1alpha1.NATSAuth">
NATSAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Auth information
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>queue</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Queue is the name of the queue group to subscribe as if specified. Uses
QueueSubscribe logic to subscribe as queue group. If the queue is empty,
uses default Subscribe logic.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NATSTrigger">

NATSTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

NATSTrigger refers to the specification of the NATS trigger.
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL of the NATS cluster.
</p>

</td>

</tr>

<tr>

<td>

<code>subject</code></br> <em> string </em>
</td>

<td>

<p>

Name of the subject to put message on.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the NATS producer.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NSQEventSource">

NSQEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

NSQEventSource describes the event source for NSQ PubSub More info at
<a href="https://godoc.org/github.com/nsqio/go-nsq">https://godoc.org/github.com/nsqio/go-nsq</a>
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

<code>hostAddress</code></br> <em> string </em>
</td>

<td>

<p>

HostAddress is the address of the host for NSQ lookup
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<p>

Topic to subscribe to.
</p>

</td>

</tr>

<tr>

<td>

<code>channel</code></br> <em> string </em>
</td>

<td>

<p>

Channel used for subscription
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the nsq client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.NativeStrategy">

NativeStrategy
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.NATSBus">NATSBus</a>)
</p>

<p>

<p>

NativeStrategy indicates to install a native NATS service
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

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

Size is the NATS StatefulSet size
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em>
<a href="#argoproj.io/v1alpha1.AuthStrategy"> AuthStrategy </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>persistence</code></br> <em>
<a href="#argoproj.io/v1alpha1.PersistenceStrategy"> PersistenceStrategy
</a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

<tr>

<td>

<code>containerTemplate</code></br> <em>
<a href="#argoproj.io/v1alpha1.ContainerTemplate"> ContainerTemplate
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ContainerTemplate contains customized spec for NATS container
</p>

</td>

</tr>

<tr>

<td>

<code>metricsContainerTemplate</code></br> <em>
<a href="#argoproj.io/v1alpha1.ContainerTemplate"> ContainerTemplate
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

MetricsContainerTemplate contains customized spec for metrics container
</p>

</td>

</tr>

<tr>

<td>

<code>nodeSelector</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

NodeSelector is a selector which must be true for the pod to fit on a
node. Selector which must match a node’s labels for the pod to be
scheduled on that node. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/assign-pod-node/">https://kubernetes.io/docs/concepts/configuration/assign-pod-node/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>tolerations</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#toleration-v1-core">
\[\]Kubernetes core/v1.Toleration </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, the pod’s tolerations.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em>
<a href="#argoproj.io/v1alpha1.Metadata"> Metadata </a> </em>
</td>

<td>

<p>

Metadata sets the pods’s metadata, i.e. annotations and labels
</p>

</td>

</tr>

<tr>

<td>

<code>securityContext</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#podsecuritycontext-v1-core">
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

<tr>

<td>

<code>maxAge</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Max Age of existing messages, i.e. “72h”, “4h35m”
</p>

</td>

</tr>

<tr>

<td>

<code>imagePullSecrets</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#localobjectreference-v1-core">
\[\]Kubernetes core/v1.LocalObjectReference </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ImagePullSecrets is an optional list of references to secrets in the
same namespace to use for pulling any of the images used by this
PodSpec. If specified, these secrets will be passed to individual puller
implementations for them to use. For example, in the case of docker,
only DockerConfig type secrets are honored. More info:
<a href="https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod">https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod</a>
</p>

</td>

</tr>

<tr>

<td>

<code>serviceAccountName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ServiceAccountName to apply to NATS StatefulSet
</p>

</td>

</tr>

<tr>

<td>

<code>priorityClassName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, indicates the EventSource pod’s priority.
“system-node-critical” and “system-cluster-critical” are two special
keywords which indicate the highest priorities with the former being the
highest priority. Any other name must be defined by creating a
PriorityClass object with that name. If not specified, the pod priority
will be default or zero if there is no default. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>priority</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

The priority value. Various system components use this field to find the
priority of the EventSource pod. When Priority Admission Controller is
enabled, it prevents users from setting this field. The admission
controller populates this field from PriorityClassName. The higher the
value, the higher the priority. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>affinity</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#affinity-v1-core">
Kubernetes core/v1.Affinity </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

The pod’s scheduling constraints More info:
<a href="https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/">https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>maxMsgs</code></br> <em> uint64 </em>
</td>

<td>

<p>

Maximum number of messages per channel, 0 means unlimited. Defaults to
1000000
</p>

</td>

</tr>

<tr>

<td>

<code>maxBytes</code></br> <em> string </em>
</td>

<td>

<p>

Total size of messages per channel, 0 means unlimited. Defaults to 1GB
</p>

</td>

</tr>

<tr>

<td>

<code>maxSubs</code></br> <em> uint64 </em>
</td>

<td>

<p>

Maximum number of subscriptions per channel, 0 means unlimited. Defaults
to 1000
</p>

</td>

</tr>

<tr>

<td>

<code>maxPayload</code></br> <em> string </em>
</td>

<td>

<p>

Maximum number of bytes in a message payload, 0 means unlimited.
Defaults to 1MB
</p>

</td>

</tr>

<tr>

<td>

<code>raftHeartbeatTimeout</code></br> <em> string </em>
</td>

<td>

<p>

Specifies the time in follower state without a leader before attempting
an election, i.e. “72h”, “4h35m”. Defaults to 2s
</p>

</td>

</tr>

<tr>

<td>

<code>raftElectionTimeout</code></br> <em> string </em>
</td>

<td>

<p>

Specifies the time in candidate state without a leader before attempting
an election, i.e. “72h”, “4h35m”. Defaults to 2s
</p>

</td>

</tr>

<tr>

<td>

<code>raftLeaseTimeout</code></br> <em> string </em>
</td>

<td>

<p>

Specifies how long a leader waits without being able to contact a quorum
of nodes before stepping down as leader, i.e. “72h”, “4h35m”. Defaults
to 1s
</p>

</td>

</tr>

<tr>

<td>

<code>raftCommitTimeout</code></br> <em> string </em>
</td>

<td>

<p>

Specifies the time without an Apply() operation before sending an
heartbeat to ensure timely commit, i.e. “72h”, “4h35m”. Defaults to
100ms
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.OpenWhiskTrigger">

OpenWhiskTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

OpenWhiskTrigger refers to the specification of the OpenWhisk trigger.
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

<code>host</code></br> <em> string </em>
</td>

<td>

<p>

Host URL of the OpenWhisk.
</p>

</td>

</tr>

<tr>

<td>

<code>version</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Version for the API. Defaults to v1.
</p>

</td>

</tr>

<tr>

<td>

<code>namespace</code></br> <em> string </em>
</td>

<td>

<p>

Namespace for the action. Defaults to “\_”.
</p>

</td>

</tr>

<tr>

<td>

<code>authToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AuthToken for authentication.
</p>

</td>

</tr>

<tr>

<td>

<code>actionName</code></br> <em> string </em>
</td>

<td>

<p>

Name of the action/function.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.OwnedRepositories">

OwnedRepositories
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">GithubEventSource</a>)
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

<code>owner</code></br> <em> string </em>
</td>

<td>

<p>

Organization or user name
</p>

</td>

</tr>

<tr>

<td>

<code>names</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Repository names
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.PayloadField">

PayloadField
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ExprFilter">ExprFilter</a>)
</p>

<p>

<p>

PayloadField binds a value at path within the event payload against a
name.
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

<code>path</code></br> <em> string </em>
</td>

<td>

<p>

Path is the JSONPath of the event’s (JSON decoded) data key Path is a
series of keys separated by a dot. A key may contain wildcard characters
‘\*’ and ‘?’. To access an array value use the index as the key. The dot
and wildcard characters can be escaped with ‘&rsquo;. See
<a href="https://github.com/tidwall/gjson#path-syntax">https://github.com/tidwall/gjson#path-syntax</a>
for more information on how to use this.
</p>

</td>

</tr>

<tr>

<td>

<code>name</code></br> <em> string </em>
</td>

<td>

<p>

Name acts as key that holds the value at the path.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.PersistenceStrategy">

PersistenceStrategy
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.JetStreamBus">JetStreamBus</a>,
<a href="#argoproj.io/v1alpha1.NativeStrategy">NativeStrategy</a>)
</p>

<p>

<p>

PersistenceStrategy defines the strategy of persistence
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

<code>storageClassName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Name of the StorageClass required by the claim. More info:
<a href="https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1">https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</a>
</p>

</td>

</tr>

<tr>

<td>

<code>accessMode</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#persistentvolumeaccessmode-v1-core">
Kubernetes core/v1.PersistentVolumeAccessMode </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Available access modes such as ReadWriteOnce, ReadWriteMany
<a href="https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes">https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes</a>
</p>

</td>

</tr>

<tr>

<td>

<code>volumeSize</code></br> <em>
k8s.io/apimachinery/pkg/api/resource.Quantity </em>
</td>

<td>

<p>

Volume size, e.g. 10Gi
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.PubSubEventSource">

PubSubEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

PubSubEventSource refers to event-source for GCP PubSub related events.
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

<code>projectID</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ProjectID is GCP project ID for the subscription. Required if you run
Argo Events outside of GKE/GCE. (otherwise, the default value is its
project)
</p>

</td>

</tr>

<tr>

<td>

<code>topicProjectID</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

TopicProjectID is GCP project ID for the topic. By default, it is same
as ProjectID.
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Topic to which the subscription should belongs. Required if you want the
eventsource to create a new subscription. If you specify this field
along with an existing subscription, it will be verified whether it
actually belongs to the specified topic.
</p>

</td>

</tr>

<tr>

<td>

<code>subscriptionID</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

SubscriptionID is ID of subscription. Required if you use existing
subscription. The default value will be auto generated hash based on
this eventsource setting, so the subscription might be recreated every
time you update the setting, which has a possibility of event loss.
</p>

</td>

</tr>

<tr>

<td>

<code>credentialSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

CredentialSecret references to the secret that contains JSON credentials
to access GCP. If it is missing, it implicitly uses Workload Identity to
access.
<a href="https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity">https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity</a>
</p>

</td>

</tr>

<tr>

<td>

<code>deleteSubscriptionOnFinish</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DeleteSubscriptionOnFinish determines whether to delete the GCP PubSub
subscription once the event source is stopped.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.PulsarEventSource">

PulsarEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

PulsarEventSource describes the event source for Apache Pulsar
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

<code>topics</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Name of the topics to subscribe to.
</p>

</td>

</tr>

<tr>

<td>

<code>type</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Type of the subscription. Only “exclusive” and “shared” is supported.
Defaults to exclusive.
</p>

</td>

</tr>

<tr>

<td>

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

Configure the service URL for the Pulsar service.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsTrustCertsSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Trusted TLS certificate secret.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsAllowInsecureConnection</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Whether the Pulsar client accept untrusted TLS certificate from broker.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsValidateHostname</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Whether the Pulsar client verify the validity of the host name from
broker.
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the pulsar client.
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>authTokenSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication token for the pulsar client. Either token or athenz can
be set to use auth.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>authAthenzParams</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication athenz parameters for the pulsar client. Refer
<a href="https://github.com/apache/pulsar-client-go/blob/master/pulsar/auth/athenz.go">https://github.com/apache/pulsar-client-go/blob/master/pulsar/auth/athenz.go</a>
Either token or athenz can be set to use auth.
</p>

</td>

</tr>

<tr>

<td>

<code>authAthenzSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication athenz privateKey secret for the pulsar client.
AuthAthenzSecret must be set if AuthAthenzParams is used.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.PulsarTrigger">

PulsarTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

PulsarTrigger refers to the specification of the Pulsar trigger.
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

Configure the service URL for the Pulsar service.
</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>
</td>

<td>

<p>

Name of the topic. See
<a href="https://pulsar.apache.org/docs/en/concepts-messaging/">https://pulsar.apache.org/docs/en/concepts-messaging/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters that is applied to resolved Kafka
trigger object.
</p>

</td>

</tr>

<tr>

<td>

<code>payload</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Payload is the list of key-value extracted from an event payload to
construct the request payload.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsTrustCertsSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Trusted TLS certificate secret.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsAllowInsecureConnection</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Whether the Pulsar client accept untrusted TLS certificate from broker.
</p>

</td>

</tr>

<tr>

<td>

<code>tlsValidateHostname</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

Whether the Pulsar client verify the validity of the host name from
broker.
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the pulsar client.
</p>

</td>

</tr>

<tr>

<td>

<code>authTokenSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication token for the pulsar client. Either token or athenz can
be set to use auth.
</p>

</td>

</tr>

<tr>

<td>

<code>connectionBackoff</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Backoff holds parameters applied to connection.
</p>

</td>

</tr>

<tr>

<td>

<code>authAthenzParams</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication athenz parameters for the pulsar client. Refer
<a href="https://github.com/apache/pulsar-client-go/blob/master/pulsar/auth/athenz.go">https://github.com/apache/pulsar-client-go/blob/master/pulsar/auth/athenz.go</a>
Either token or athenz can be set to use auth.
</p>

</td>

</tr>

<tr>

<td>

<code>authAthenzSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Authentication athenz privateKey secret for the pulsar client.
AuthAthenzSecret must be set if AuthAthenzParams is used.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.RateLimit">

RateLimit
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
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

<code>unit</code></br> <em>
<a href="#argoproj.io/v1alpha1.RateLimiteUnit"> RateLimiteUnit </a>
</em>
</td>

<td>

<p>

Defaults to Second
</p>

</td>

</tr>

<tr>

<td>

<code>requestsPerUnit</code></br> <em> int32 </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.RateLimiteUnit">

RateLimiteUnit (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.RateLimit">RateLimit</a>)
</p>

<p>

</p>

<h3 id="argoproj.io/v1alpha1.RedisEventSource">

RedisEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

RedisEventSource describes an event source for the Redis PubSub. More
info at
<a href="https://godoc.org/github.com/go-redis/redis#example-PubSub">https://godoc.org/github.com/go-redis/redis#example-PubSub</a>
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

<code>hostAddress</code></br> <em> string </em>
</td>

<td>

<p>

HostAddress refers to the address of the Redis host/server
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Password required for authentication if any.
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

Namespace to use to retrieve the password from. It should only be
specified if password is declared
</p>

</td>

</tr>

<tr>

<td>

<code>db</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

DB to use. If not specified, default DB 0 will be used.
</p>

</td>

</tr>

<tr>

<td>

<code>channels</code></br> <em> \[\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the redis client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Username required for ACL style authentication if any.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.RedisStreamEventSource">

RedisStreamEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

RedisStreamEventSource describes an event source for Redis streams
(<a href="https://redis.io/topics/streams-intro">https://redis.io/topics/streams-intro</a>)
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

<code>hostAddress</code></br> <em> string </em>
</td>

<td>

<p>

HostAddress refers to the address of the Redis host/server (master
instance)
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Password required for authentication if any.
</p>

</td>

</tr>

<tr>

<td>

<code>db</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

DB to use. If not specified, default DB 0 will be used.
</p>

</td>

</tr>

<tr>

<td>

<code>streams</code></br> <em> \[\]string </em>
</td>

<td>

<p>

Streams to look for entries. XREADGROUP is used on all streams using a
single consumer group.
</p>

</td>

</tr>

<tr>

<td>

<code>maxMsgCountPerRead</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

MaxMsgCountPerRead holds the maximum number of messages per stream that
will be read in each XREADGROUP of all streams Example: if there are 2
streams and MaxMsgCountPerRead=10, then each XREADGROUP may read upto a
total of 20 messages. Same as COUNT option in
XREADGROUP(<a href="https://redis.io/topics/streams-intro">https://redis.io/topics/streams-intro</a>).
Defaults to 10
</p>

</td>

</tr>

<tr>

<td>

<code>consumerGroup</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ConsumerGroup refers to the Redis stream consumer group that will be
created on all redis streams. Messages are read through this group.
Defaults to ‘argo-events-cg’
</p>

</td>

</tr>

<tr>

<td>

<code>tls</code></br> <em> <a href="#argoproj.io/v1alpha1.TLSConfig">
TLSConfig </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

TLS configuration for the redis client.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Username required for ACL style authentication if any.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ResourceEventSource">

ResourceEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

ResourceEventSource refers to a event-source for K8s resource related
events.
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

<code>namespace</code></br> <em> string </em>
</td>

<td>

<p>

Namespace where resource is deployed
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.ResourceFilter"> ResourceFilter </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter is applied on the metadata of the resource If you apply filter,
then the internal event informer will only monitor objects that pass the
filter.
</p>

</td>

</tr>

<tr>

<td>

<code>GroupVersionResource</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#groupversionresource-v1-meta">
Kubernetes meta/v1.GroupVersionResource </a> </em>
</td>

<td>

<p>

(Members of <code>GroupVersionResource</code> are embedded into this
type.)
</p>

<p>

Group of the resource
</p>

</td>

</tr>

<tr>

<td>

<code>eventTypes</code></br> <em>
<a href="#argoproj.io/v1alpha1.ResourceEventType"> \[\]ResourceEventType
</a> </em>
</td>

<td>

<p>

EventTypes is the list of event type to watch. Possible values are -
ADD, UPDATE and DELETE.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ResourceEventType">

ResourceEventType (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ResourceEventSource">ResourceEventSource</a>)
</p>

<p>

<p>

ResourceEventType is the type of event for the K8s resource mutation
</p>

</p>

<h3 id="argoproj.io/v1alpha1.ResourceFilter">

ResourceFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ResourceEventSource">ResourceEventSource</a>)
</p>

<p>

<p>

ResourceFilter contains K8s ObjectMeta information to further filter
resource event objects
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

<code>prefix</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Prefix filter is applied on the resource name.
</p>

</td>

</tr>

<tr>

<td>

<code>labels</code></br> <em> <a href="#argoproj.io/v1alpha1.Selector">
\[\]Selector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Labels provide listing options to K8s API to watch resource/s. Refer
<a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/label-selectors/">https://kubernetes.io/docs/concepts/overview/working-with-objects/label-selectors/</a>
for more info. Unlike K8s field selector, multiple values are passed as
comma separated values instead of list of values. Eg: value:
value1,value2. Same as K8s label selector, operator “=”, “==”, “!=”,
“exists”, “!”, “notin”, “in”, “gt” and “lt” are supported
</p>

</td>

</tr>

<tr>

<td>

<code>fields</code></br> <em> <a href="#argoproj.io/v1alpha1.Selector">
\[\]Selector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Fields provide field filters similar to K8s field selector (see
<a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/">https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/</a>).
Unlike K8s field selector, it supports arbitrary fileds like
“spec.serviceAccountName”, and the value could be a string or a regex.
Same as K8s field selector, operator “=”, “==” and “!=” are supported.
</p>

</td>

</tr>

<tr>

<td>

<code>createdBy</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#time-v1-meta">
Kubernetes meta/v1.Time </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If resource is created before the specified time then the event is
treated as valid.
</p>

</td>

</tr>

<tr>

<td>

<code>afterStart</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

If the resource is created after the start time then the event is
treated as valid.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.S3Artifact">

S3Artifact
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation">ArtifactLocation</a>,
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

S3Artifact contains information about an S3 connection and bucket
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

<code>endpoint</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>bucket</code></br> <em> <a href="#argoproj.io/v1alpha1.S3Bucket">
S3Bucket </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>region</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>insecure</code></br> <em> bool </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>accessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em> <a href="#argoproj.io/v1alpha1.S3Filter">
S3Filter </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>caCertificate</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.S3Bucket">

S3Bucket
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.S3Artifact">S3Artifact</a>)
</p>

<p>

<p>

S3Bucket contains information to describe an S3 Bucket
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

<code>key</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>name</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.S3Filter">

S3Filter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.S3Artifact">S3Artifact</a>)
</p>

<p>

<p>

S3Filter represents filters to apply to bucket notifications for
specifying constraints on objects
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

<code>prefix</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>suffix</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SASLConfig">

SASLConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.KafkaBus">KafkaBus</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>,
<a href="#argoproj.io/v1alpha1.KafkaTrigger">KafkaTrigger</a>)
</p>

<p>

<p>

SASLConfig refers to SASL configuration for a client
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

<code>mechanism</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

SASLMechanism is the name of the enabled SASL mechanism. Possible
values: OAUTHBEARER, PLAIN (defaults to PLAIN).
</p>

</td>

</tr>

<tr>

<td>

<code>userSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

User is the authentication identity (authcid) to present for SASL/PLAIN
or SASL/SCRAM authentication
</p>

</td>

</tr>

<tr>

<td>

<code>passwordSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Password for SASL/PLAIN authentication
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SFTPEventSource">

SFTPEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

SFTPEventSource describes an event-source for sftp related events.
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

<code>eventType</code></br> <em> string </em>
</td>

<td>

<p>

Type of file operations to watch Refer
<a href="https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go">https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go</a>
for more information
</p>

</td>

</tr>

<tr>

<td>

<code>watchPathConfig</code></br> <em>
<a href="#argoproj.io/v1alpha1.WatchPathConfig"> WatchPathConfig </a>
</em>
</td>

<td>

<p>

WatchPathConfig contains configuration about the file path to watch
</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Username required for authentication if any.
</p>

</td>

</tr>

<tr>

<td>

<code>password</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Password required for authentication if any.
</p>

</td>

</tr>

<tr>

<td>

<code>sshKeySecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SSHKeySecret refers to the secret that contains SSH key. Key needs to
contain private key and public key.
</p>

</td>

</tr>

<tr>

<td>

<code>address</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Address sftp address.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>pollIntervalDuration</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

PollIntervalDuration the interval at which to poll the SFTP server
defaults to 10 seconds
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SNSEventSource">

SNSEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

SNSEventSource refers to event-source for AWS SNS related events
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook configuration for http server
</p>

</td>

</tr>

<tr>

<td>

<code>topicArn</code></br> <em> string </em>
</td>

<td>

<p>

TopicArn
</p>

</td>

</tr>

<tr>

<td>

<code>accessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

AccessKey refers K8s secret containing aws access key
</p>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SecretKey refers K8s secret containing aws secret key
</p>

</td>

</tr>

<tr>

<td>

<code>region</code></br> <em> string </em>
</td>

<td>

<p>

Region is AWS region
</p>

</td>

</tr>

<tr>

<td>

<code>roleARN</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

RoleARN is the Amazon Resource Name (ARN) of the role to assume.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>validateSignature</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

ValidateSignature is boolean that can be set to true for SNS signature
verification
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>endpoint</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Endpoint configures connection to a specific SNS endpoint instead of
Amazons servers
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SQSEventSource">

SQSEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

SQSEventSource refers to event-source for AWS SQS related events
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

<code>accessKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

AccessKey refers K8s secret containing aws access key
</p>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SecretKey refers K8s secret containing aws secret key
</p>

</td>

</tr>

<tr>

<td>

<code>region</code></br> <em> string </em>
</td>

<td>

<p>

Region is AWS region
</p>

</td>

</tr>

<tr>

<td>

<code>queue</code></br> <em> string </em>
</td>

<td>

<p>

Queue is AWS SQS queue to listen to for messages
</p>

</td>

</tr>

<tr>

<td>

<code>waitTimeSeconds</code></br> <em> int64 </em>
</td>

<td>

<p>

WaitTimeSeconds is The duration (in seconds) for which the call waits
for a message to arrive in the queue before returning.
</p>

</td>

</tr>

<tr>

<td>

<code>roleARN</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

RoleARN is the Amazon Resource Name (ARN) of the role to assume.
</p>

</td>

</tr>

<tr>

<td>

<code>jsonBody</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

JSONBody specifies that all event body payload coming from this source
will be JSON
</p>

</td>

</tr>

<tr>

<td>

<code>queueAccountId</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

QueueAccountID is the ID of the account that created the queue to
monitor
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>dlq</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

DLQ specifies if a dead-letter queue is configured for messages that
can’t be processed successfully. If set to true, messages with invalid
payload won’t be acknowledged to allow to forward them farther to the
dead-letter queue. The default value is false.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

<tr>

<td>

<code>endpoint</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Endpoint configures connection to a specific SQS endpoint instead of
Amazons servers
</p>

</td>

</tr>

<tr>

<td>

<code>sessionToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SessionToken refers to K8s secret containing AWS temporary
credentials(STS) session token
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SchemaRegistryConfig">

SchemaRegistryConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.KafkaTrigger">KafkaTrigger</a>)
</p>

<p>

<p>

SchemaRegistryConfig refers to configuration for a client
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

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

Schema Registry URL.
</p>

</td>

</tr>

<tr>

<td>

<code>schemaId</code></br> <em> int32 </em>
</td>

<td>

<p>

Schema ID
</p>

</td>

</tr>

<tr>

<td>

<code>auth</code></br> <em> <a href="#argoproj.io/v1alpha1.BasicAuth">
BasicAuth </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

SchemaRegistry - basic authentication
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SecureHeader">

SecureHeader
</h3>

<p>

<p>

SecureHeader refers to HTTP Headers with auth tokens as values
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

</td>

</tr>

<tr>

<td>

<code>valueFrom</code></br> <em>
<a href="#argoproj.io/v1alpha1.ValueFromSource"> ValueFromSource </a>
</em>
</td>

<td>

<p>

Values can be read from either secrets or configmaps
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Selector">

Selector
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ResourceFilter">ResourceFilter</a>)
</p>

<p>

<p>

Selector represents conditional operation to select K8s objects.
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

<code>key</code></br> <em> string </em>
</td>

<td>

<p>

Key name
</p>

</td>

</tr>

<tr>

<td>

<code>operation</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Supported operations like ==, != etc. Defaults to ==. Refer
<a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors">https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors</a>
for more info.
</p>

</td>

</tr>

<tr>

<td>

<code>value</code></br> <em> string </em>
</td>

<td>

<p>

Value
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Sensor">

Sensor
</h3>

<p>

<p>

Sensor is the definition of a sensor resource
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
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta </a> </em>
</td>

<td>

Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>

</tr>

<tr>

<td>

<code>spec</code></br> <em> <a href="#argoproj.io/v1alpha1.SensorSpec">
SensorSpec </a> </em>
</td>

<td>

<br/> <br/>
<table>

<tr>

<td>

<code>dependencies</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventDependency"> \[\]EventDependency
</a> </em>
</td>

<td>

<p>

Dependencies is a list of the events that this sensor is dependent on.
</p>

</td>

</tr>

<tr>

<td>

<code>triggers</code></br> <em> <a href="#argoproj.io/v1alpha1.Trigger">
\[\]Trigger </a> </em>
</td>

<td>

<p>

Triggers is a list of the things that this sensor evokes. These are the
outputs from this sensor.
</p>

</td>

</tr>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Template is the pod specification for the sensor
</p>

</td>

</tr>

<tr>

<td>

<code>errorOnFailedRound</code></br> <em> bool </em>
</td>

<td>

<p>

ErrorOnFailedRound if set to true, marks sensor state as
<code>error</code> if the previous trigger round fails. Once sensor
state is set to <code>error</code>, no further triggers will be
processed.
</p>

</td>

</tr>

<tr>

<td>

<code>eventBusName</code></br> <em> string </em>
</td>

<td>

<p>

EventBusName references to a EventBus name. By default the value is
“default”
</p>

</td>

</tr>

<tr>

<td>

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

Replicas is the sensor deployment replicas
</p>

</td>

</tr>

<tr>

<td>

<code>revisionHistoryLimit</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

RevisionHistoryLimit specifies how many old deployment revisions to
retain
</p>

</td>

</tr>

<tr>

<td>

<code>loggingFields</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

LoggingFields add additional key-value pairs when logging happens
</p>

</td>

</tr>

</table>

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="#argoproj.io/v1alpha1.SensorStatus"> SensorStatus </a> </em>
</td>

<td>

<em>(Optional)</em>
</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SensorSpec">

SensorSpec
</h3>

<p>

(<em>Appears on:</em> <a href="#argoproj.io/v1alpha1.Sensor">Sensor</a>)
</p>

<p>

<p>

SensorSpec represents desired sensor state
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

<code>dependencies</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventDependency"> \[\]EventDependency
</a> </em>
</td>

<td>

<p>

Dependencies is a list of the events that this sensor is dependent on.
</p>

</td>

</tr>

<tr>

<td>

<code>triggers</code></br> <em> <a href="#argoproj.io/v1alpha1.Trigger">
\[\]Trigger </a> </em>
</td>

<td>

<p>

Triggers is a list of the things that this sensor evokes. These are the
outputs from this sensor.
</p>

</td>

</tr>

<tr>

<td>

<code>template</code></br> <em>
<a href="#argoproj.io/v1alpha1.Template"> Template </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Template is the pod specification for the sensor
</p>

</td>

</tr>

<tr>

<td>

<code>errorOnFailedRound</code></br> <em> bool </em>
</td>

<td>

<p>

ErrorOnFailedRound if set to true, marks sensor state as
<code>error</code> if the previous trigger round fails. Once sensor
state is set to <code>error</code>, no further triggers will be
processed.
</p>

</td>

</tr>

<tr>

<td>

<code>eventBusName</code></br> <em> string </em>
</td>

<td>

<p>

EventBusName references to a EventBus name. By default the value is
“default”
</p>

</td>

</tr>

<tr>

<td>

<code>replicas</code></br> <em> int32 </em>
</td>

<td>

<p>

Replicas is the sensor deployment replicas
</p>

</td>

</tr>

<tr>

<td>

<code>revisionHistoryLimit</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

RevisionHistoryLimit specifies how many old deployment revisions to
retain
</p>

</td>

</tr>

<tr>

<td>

<code>loggingFields</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

LoggingFields add additional key-value pairs when logging happens
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SensorStatus">

SensorStatus
</h3>

<p>

(<em>Appears on:</em> <a href="#argoproj.io/v1alpha1.Sensor">Sensor</a>)
</p>

<p>

<p>

SensorStatus contains information about the status of a sensor.
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

<code>Status</code></br> <em> <a href="#argoproj.io/v1alpha1.Status">
Status </a> </em>
</td>

<td>

<p>

(Members of <code>Status</code> are embedded into this type.)
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
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

Service holds the service information eventsource exposes
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
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#serviceport-v1-core">
\[\]Kubernetes core/v1.ServicePort </a> </em>
</td>

<td>

<p>

The list of ports that are exposed by this ClusterIP service.
</p>

</td>

</tr>

<tr>

<td>

<code>clusterIP</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

clusterIP is the IP address of the service and is usually assigned
randomly by the master. If an address is specified manually and is not
in use by others, it will be allocated to the service; otherwise,
creation of the service will fail. This field can not be changed through
updates. Valid values are “None”, empty string (“”), or a valid IP
address. “None” can be specified for headless services when proxying is
not required. More info:
<a href="https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies">https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies</a>
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SlackEventSource">

SlackEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

SlackEventSource refers to event-source for Slack related events
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

<code>signingSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Slack App signing secret
</p>

</td>

</tr>

<tr>

<td>

<code>token</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Token for URL verification handshake
</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration for a REST endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SlackSender">

SlackSender
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.SlackTrigger">SlackTrigger</a>)
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

<code>username</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Username is the Slack application’s username
</p>

</td>

</tr>

<tr>

<td>

<code>icon</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Icon is the Slack application’s icon, e.g. :robot_face: or
<a href="https://example.com/image.png">https://example.com/image.png</a>
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SlackThread">

SlackThread
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.SlackTrigger">SlackTrigger</a>)
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

<code>messageAggregationKey</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

MessageAggregationKey allows to aggregate the messages to a thread by
some key.
</p>

</td>

</tr>

<tr>

<td>

<code>broadcastMessageToChannel</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

BroadcastMessageToChannel allows to also broadcast the message from the
thread to the channel
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.SlackTrigger">

SlackTrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

SlackTrigger refers to the specification of the slack notification
trigger.
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

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Parameters is the list of key-value extracted from event’s payload that
are applied to the trigger resource.
</p>

</td>

</tr>

<tr>

<td>

<code>slackToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

SlackToken refers to the Kubernetes secret that holds the slack token
required to send messages.
</p>

</td>

</tr>

<tr>

<td>

<code>channel</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Channel refers to which Slack channel to send Slack message.
</p>

</td>

</tr>

<tr>

<td>

<code>message</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Message refers to the message to send to the Slack channel.
</p>

</td>

</tr>

<tr>

<td>

<code>attachments</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Attachments is a JSON format string that represents an array of Slack
attachments according to the attachments API:
<a href="https://api.slack.com/reference/messaging/attachments">https://api.slack.com/reference/messaging/attachments</a>
.
</p>

</td>

</tr>

<tr>

<td>

<code>blocks</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Blocks is a JSON format string that represents an array of Slack blocks
according to the blocks API:
<a href="https://api.slack.com/reference/block-kit/blocks">https://api.slack.com/reference/block-kit/blocks</a>
.
</p>

</td>

</tr>

<tr>

<td>

<code>thread</code></br> <em>
<a href="#argoproj.io/v1alpha1.SlackThread"> SlackThread </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Thread refers to additional options for sending messages to a Slack
thread.
</p>

</td>

</tr>

<tr>

<td>

<code>sender</code></br> <em>
<a href="#argoproj.io/v1alpha1.SlackSender"> SlackSender </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Sender refers to additional configuration of the Slack application that
sends the message.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.StandardK8STrigger">

StandardK8STrigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>

<p>

<p>

StandardK8STrigger is the standard Kubernetes resource trigger
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

<code>source</code></br> <em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation"> ArtifactLocation </a>
</em>
</td>

<td>

<p>

Source of the K8s resource file(s)
</p>

</td>

</tr>

<tr>

<td>

<code>operation</code></br> <em>
<a href="#argoproj.io/v1alpha1.KubernetesResourceOperation">
KubernetesResourceOperation </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Operation refers to the type of operation performed on the k8s resource.
Default value is Create.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters that is applied to resolved K8s
trigger object.
</p>

</td>

</tr>

<tr>

<td>

<code>patchStrategy</code></br> <em>
k8s.io/apimachinery/pkg/types.PatchType </em>
</td>

<td>

<em>(Optional)</em>
<p>

PatchStrategy controls the K8s object patching strategy when the trigger
operation is specified as patch. possible values:
“application/json-patch+json” “application/merge-patch+json”
“application/strategic-merge-patch+json” “application/apply-patch+yaml”.
Defaults to “application/merge-patch+json”
</p>

</td>

</tr>

<tr>

<td>

<code>liveObject</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

LiveObject specifies whether the resource should be directly fetched
from K8s instead of being marshaled from the resource artifact. If set
to true, the resource artifact must contain the information required to
uniquely identify the resource in the cluster, that is, you must specify
“apiVersion”, “kind” as well as “name” and “namespace” meta data. Only
valid for operation type <code>update</code>
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Status">

Status
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventBusStatus">EventBusStatus</a>,
<a href="#argoproj.io/v1alpha1.EventSourceStatus">EventSourceStatus</a>,
<a href="#argoproj.io/v1alpha1.SensorStatus">SensorStatus</a>)
</p>

<p>

<p>

Status is a common structure which can be used for Status field.
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

<code>conditions</code></br> <em>
<a href="#argoproj.io/v1alpha1.Condition"> \[\]Condition </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Conditions are the latest available observations of a resource’s current
state.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.StatusPolicy">

StatusPolicy
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerPolicy">TriggerPolicy</a>)
</p>

<p>

<p>

StatusPolicy refers to the policy used to check the state of the trigger
using response status
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

<code>allow</code></br> <em> \[\]int32 </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.StorageGridEventSource">

StorageGridEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

StorageGridEventSource refers to event-source for StorageGrid related
events
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration for a REST endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.StorageGridFilter"> StorageGridFilter
</a> </em>
</td>

<td>

<p>

Filter on object key which caused the notification.
</p>

</td>

</tr>

<tr>

<td>

<code>topicArn</code></br> <em> string </em>
</td>

<td>

<p>

TopicArn
</p>

</td>

</tr>

<tr>

<td>

<code>bucket</code></br> <em> string </em>
</td>

<td>

<p>

Name of the bucket to register notifications for.
</p>

</td>

</tr>

<tr>

<td>

<code>region</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

S3 region. Defaults to us-east-1
</p>

</td>

</tr>

<tr>

<td>

<code>authToken</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

Auth token for storagegrid api
</p>

</td>

</tr>

<tr>

<td>

<code>apiURL</code></br> <em> string </em>
</td>

<td>

<p>

APIURL is the url of the storagegrid api.
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.StorageGridFilter">

StorageGridFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.StorageGridEventSource">StorageGridEventSource</a>)
</p>

<p>

<p>

StorageGridFilter represents filters to apply to bucket notifications
for specifying constraints on objects
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

<code>prefix</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>suffix</code></br> <em> string </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.StripeEventSource">

StripeEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

StripeEventSource describes the event source for stripe webhook
notifications More info at
<a href="https://stripe.com/docs/webhooks">https://stripe.com/docs/webhooks</a>
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

<code>webhook</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

Webhook holds configuration for a REST endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>createWebhook</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

CreateWebhook if specified creates a new webhook programmatically.
</p>

</td>

</tr>

<tr>

<td>

<code>apiKey</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

APIKey refers to K8s secret that holds Stripe API key. Used only if
CreateWebhook is enabled.
</p>

</td>

</tr>

<tr>

<td>

<code>eventFilter</code></br> <em> \[\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

EventFilter describes the type of events to listen to. If not specified,
all types of events will be processed. More info at
<a href="https://stripe.com/docs/api/events/list">https://stripe.com/docs/api/events/list</a>
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TLSConfig">

TLSConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>,
<a href="#argoproj.io/v1alpha1.AzureServiceBusEventSource">AzureServiceBusEventSource</a>,
<a href="#argoproj.io/v1alpha1.AzureServiceBusTrigger">AzureServiceBusTrigger</a>,
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">BitbucketServerEventSource</a>,
<a href="#argoproj.io/v1alpha1.EmitterEventSource">EmitterEventSource</a>,
<a href="#argoproj.io/v1alpha1.HTTPTrigger">HTTPTrigger</a>,
<a href="#argoproj.io/v1alpha1.KafkaBus">KafkaBus</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>,
<a href="#argoproj.io/v1alpha1.KafkaTrigger">KafkaTrigger</a>,
<a href="#argoproj.io/v1alpha1.MQTTEventSource">MQTTEventSource</a>,
<a href="#argoproj.io/v1alpha1.NATSEventsSource">NATSEventsSource</a>,
<a href="#argoproj.io/v1alpha1.NATSTrigger">NATSTrigger</a>,
<a href="#argoproj.io/v1alpha1.NSQEventSource">NSQEventSource</a>,
<a href="#argoproj.io/v1alpha1.PulsarEventSource">PulsarEventSource</a>,
<a href="#argoproj.io/v1alpha1.PulsarTrigger">PulsarTrigger</a>,
<a href="#argoproj.io/v1alpha1.RedisEventSource">RedisEventSource</a>,
<a href="#argoproj.io/v1alpha1.RedisStreamEventSource">RedisStreamEventSource</a>)
</p>

<p>

<p>

TLSConfig refers to TLS configuration for a client.
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

<code>caCertSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

CACertSecret refers to the secret that contains the CA cert
</p>

</td>

</tr>

<tr>

<td>

<code>clientCertSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

ClientCertSecret refers to the secret that contains the client cert
</p>

</td>

</tr>

<tr>

<td>

<code>clientKeySecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

ClientKeySecret refers to the secret that contains the client key
</p>

</td>

</tr>

<tr>

<td>

<code>insecureSkipVerify</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

If true, skips creation of TLSConfig with certs and creates an empty
TLSConfig. (Defaults to false)
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
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>,
<a href="#argoproj.io/v1alpha1.SensorSpec">SensorSpec</a>)
</p>

<p>

<p>

Template holds the information of a deployment template
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
<a href="#argoproj.io/v1alpha1.Metadata"> Metadata </a> </em>
</td>

<td>

<p>

Metadata sets the pods’s metadata, i.e. annotations and labels
</p>

</td>

</tr>

<tr>

<td>

<code>serviceAccountName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

ServiceAccountName is the name of the ServiceAccount to use to run
sensor pod. More info:
<a href="https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/">https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>container</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#container-v1-core">
Kubernetes core/v1.Container </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Container is the main container image to run in the sensor pod
</p>

</td>

</tr>

<tr>

<td>

<code>volumes</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#volume-v1-core">
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
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#podsecuritycontext-v1-core">
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

<tr>

<td>

<code>nodeSelector</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

NodeSelector is a selector which must be true for the pod to fit on a
node. Selector which must match a node’s labels for the pod to be
scheduled on that node. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/assign-pod-node/">https://kubernetes.io/docs/concepts/configuration/assign-pod-node/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>tolerations</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#toleration-v1-core">
\[\]Kubernetes core/v1.Toleration </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, the pod’s tolerations.
</p>

</td>

</tr>

<tr>

<td>

<code>imagePullSecrets</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#localobjectreference-v1-core">
\[\]Kubernetes core/v1.LocalObjectReference </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ImagePullSecrets is an optional list of references to secrets in the
same namespace to use for pulling any of the images used by this
PodSpec. If specified, these secrets will be passed to individual puller
implementations for them to use. For example, in the case of docker,
only DockerConfig type secrets are honored. More info:
<a href="https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod">https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod</a>
</p>

</td>

</tr>

<tr>

<td>

<code>priorityClassName</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, indicates the EventSource pod’s priority.
“system-node-critical” and “system-cluster-critical” are two special
keywords which indicate the highest priorities with the former being the
highest priority. Any other name must be defined by creating a
PriorityClass object with that name. If not specified, the pod priority
will be default or zero if there is no default. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>priority</code></br> <em> int32 </em>
</td>

<td>

<em>(Optional)</em>
<p>

The priority value. Various system components use this field to find the
priority of the EventSource pod. When Priority Admission Controller is
enabled, it prevents users from setting this field. The admission
controller populates this field from PriorityClassName. The higher the
value, the higher the priority. More info:
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>affinity</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#affinity-v1-core">
Kubernetes core/v1.Affinity </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If specified, the pod’s scheduling constraints
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TimeFilter">

TimeFilter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventDependencyFilter">EventDependencyFilter</a>)
</p>

<p>

<p>

TimeFilter describes a window in time. It filters out events that occur
outside the time limits. In other words, only events that occur after
Start and before Stop will pass this filter.
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

<code>start</code></br> <em> string </em>
</td>

<td>

<p>

Start is the beginning of a time window in UTC. Before this time, events
for this dependency are ignored. Format is hh:mm:ss.
</p>

</td>

</tr>

<tr>

<td>

<code>stop</code></br> <em> string </em>
</td>

<td>

<p>

Stop is the end of a time window in UTC. After or equal to this time,
events for this dependency are ignored and Format is hh:mm:ss. If it is
smaller than Start, it is treated as next day of Start (e.g.:
22:00:00-01:00:00 means 22:00:00-25:00:00).
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Trigger">

Trigger
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.SensorSpec">SensorSpec</a>,
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
</p>

<p>

<p>

Trigger is an action taken, output produced, an event created, a message
sent
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
<a href="#argoproj.io/v1alpha1.TriggerTemplate"> TriggerTemplate </a>
</em>
</td>

<td>

<p>

Template describes the trigger specification.
</p>

</td>

</tr>

<tr>

<td>

<code>parameters</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameter"> \[\]TriggerParameter
</a> </em>
</td>

<td>

<p>

Parameters is the list of parameters applied to the trigger template
definition
</p>

</td>

</tr>

<tr>

<td>

<code>policy</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerPolicy"> TriggerPolicy </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Policy to configure backoff and execution criteria for the trigger
</p>

</td>

</tr>

<tr>

<td>

<code>retryStrategy</code></br> <em>
<a href="#argoproj.io/v1alpha1.Backoff"> Backoff </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Retry strategy, defaults to no retry
</p>

</td>

</tr>

<tr>

<td>

<code>rateLimit</code></br> <em>
<a href="#argoproj.io/v1alpha1.RateLimit"> RateLimit </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Rate limit, default unit is Second
</p>

</td>

</tr>

<tr>

<td>

<code>atLeastOnce</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

AtLeastOnce determines the trigger execution semantics. Defaults to
false. Trigger execution will use at-most-once semantics. If set to
true, Trigger execution will switch to at-least-once semantics.
</p>

</td>

</tr>

<tr>

<td>

<code>dlqTrigger</code></br> <em>
<a href="#argoproj.io/v1alpha1.Trigger"> Trigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

If the trigger fails, it will retry up to the configured number of
retries. If the maximum retries are reached and the trigger is set to
execute atLeastOnce, the dead letter queue (DLQ) trigger will be invoked
if specified. Invoking the dead letter queue trigger helps prevent data
loss.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TriggerParameter">

TriggerParameter
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AWSLambdaTrigger">AWSLambdaTrigger</a>,
<a href="#argoproj.io/v1alpha1.ArgoWorkflowTrigger">ArgoWorkflowTrigger</a>,
<a href="#argoproj.io/v1alpha1.AzureEventHubsTrigger">AzureEventHubsTrigger</a>,
<a href="#argoproj.io/v1alpha1.AzureServiceBusTrigger">AzureServiceBusTrigger</a>,
<a href="#argoproj.io/v1alpha1.CustomTrigger">CustomTrigger</a>,
<a href="#argoproj.io/v1alpha1.EmailTrigger">EmailTrigger</a>,
<a href="#argoproj.io/v1alpha1.HTTPTrigger">HTTPTrigger</a>,
<a href="#argoproj.io/v1alpha1.KafkaTrigger">KafkaTrigger</a>,
<a href="#argoproj.io/v1alpha1.NATSTrigger">NATSTrigger</a>,
<a href="#argoproj.io/v1alpha1.OpenWhiskTrigger">OpenWhiskTrigger</a>,
<a href="#argoproj.io/v1alpha1.PulsarTrigger">PulsarTrigger</a>,
<a href="#argoproj.io/v1alpha1.SlackTrigger">SlackTrigger</a>,
<a href="#argoproj.io/v1alpha1.StandardK8STrigger">StandardK8STrigger</a>,
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
</p>

<p>

<p>

TriggerParameter indicates a passed parameter to a service template
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

<code>src</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameterSource">
TriggerParameterSource </a> </em>
</td>

<td>

<p>

Src contains a source reference to the value of the parameter from a
dependency
</p>

</td>

</tr>

<tr>

<td>

<code>dest</code></br> <em> string </em>
</td>

<td>

<p>

Dest is the JSONPath of a resource key. A path is a series of keys
separated by a dot. The colon character can be escaped with ‘.’ The -1
key can be used to append a value to an existing array. See
<a href="https://github.com/tidwall/sjson#path-syntax">https://github.com/tidwall/sjson#path-syntax</a>
for more information about how this is used.
</p>

</td>

</tr>

<tr>

<td>

<code>operation</code></br> <em>
<a href="#argoproj.io/v1alpha1.TriggerParameterOperation">
TriggerParameterOperation </a> </em>
</td>

<td>

<p>

Operation is what to do with the existing value at Dest, whether to
‘prepend’, ‘overwrite’, or ‘append’ it.
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TriggerParameterOperation">

TriggerParameterOperation (<code>string</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerParameter">TriggerParameter</a>)
</p>

<p>

<p>

TriggerParameterOperation represents how to set a trigger destination
resource key
</p>

</p>

<h3 id="argoproj.io/v1alpha1.TriggerParameterSource">

TriggerParameterSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.TriggerParameter">TriggerParameter</a>)
</p>

<p>

<p>

TriggerParameterSource defines the source for a parameter from a event
event
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

<code>dependencyName</code></br> <em> string </em>
</td>

<td>

<p>

DependencyName refers to the name of the dependency. The event which is
stored for this dependency is used as payload for the parameterization.
Make sure to refer to one of the dependencies you have defined under
Dependencies list.
</p>

</td>

</tr>

<tr>

<td>

<code>contextKey</code></br> <em> string </em>
</td>

<td>

<p>

ContextKey is the JSONPath of the event’s (JSON decoded) context key
ContextKey is a series of keys separated by a dot. A key may contain
wildcard characters ‘\*’ and ‘?’. To access an array value use the index
as the key. The dot and wildcard characters can be escaped with
‘&rsquo;. See
<a href="https://github.com/tidwall/gjson#path-syntax">https://github.com/tidwall/gjson#path-syntax</a>
for more information on how to use this.
</p>

</td>

</tr>

<tr>

<td>

<code>contextTemplate</code></br> <em> string </em>
</td>

<td>

<p>

ContextTemplate is a go-template for extracting a string from the
event’s context. If a ContextTemplate is provided with a ContextKey, the
template will be evaluated first and fallback to the ContextKey. The
templating follows the standard go-template syntax as well as sprig’s
extra functions. See
<a href="https://pkg.go.dev/text/template">https://pkg.go.dev/text/template</a>
and
<a href="https://masterminds.github.io/sprig/">https://masterminds.github.io/sprig/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>dataKey</code></br> <em> string </em>
</td>

<td>

<p>

DataKey is the JSONPath of the event’s (JSON decoded) data key DataKey
is a series of keys separated by a dot. A key may contain wildcard
characters ‘\*’ and ‘?’. To access an array value use the index as the
key. The dot and wildcard characters can be escaped with ‘&rsquo;. See
<a href="https://github.com/tidwall/gjson#path-syntax">https://github.com/tidwall/gjson#path-syntax</a>
for more information on how to use this.
</p>

</td>

</tr>

<tr>

<td>

<code>dataTemplate</code></br> <em> string </em>
</td>

<td>

<p>

DataTemplate is a go-template for extracting a string from the event’s
data. If a DataTemplate is provided with a DataKey, the template will be
evaluated first and fallback to the DataKey. The templating follows the
standard go-template syntax as well as sprig’s extra functions. See
<a href="https://pkg.go.dev/text/template">https://pkg.go.dev/text/template</a>
and
<a href="https://masterminds.github.io/sprig/">https://masterminds.github.io/sprig/</a>
</p>

</td>

</tr>

<tr>

<td>

<code>value</code></br> <em> string </em>
</td>

<td>

<p>

Value is the default literal value to use for this parameter source This
is only used if the DataKey is invalid. If the DataKey is invalid and
this is not defined, this param source will produce an error.
</p>

</td>

</tr>

<tr>

<td>

<code>useRawData</code></br> <em> bool </em>
</td>

<td>

<em>(Optional)</em>
<p>

UseRawData indicates if the value in an event at data key should be used
without converting to string. When true, a number, boolean, json or
string parameter may be extracted. When the field is unspecified, or
explicitly false, the behavior is to turn the extracted field into a
string. (e.g. when set to true, the parameter 123 will resolve to the
numerical type, but when false, or not provided, the string “123” will
be resolved)
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TriggerPolicy">

TriggerPolicy
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
</p>

<p>

<p>

TriggerPolicy dictates the policy for the trigger retries
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

<code>k8s</code></br> <em>
<a href="#argoproj.io/v1alpha1.K8SResourcePolicy"> K8SResourcePolicy
</a> </em>
</td>

<td>

<p>

K8SResourcePolicy refers to the policy used to check the state of K8s
based triggers using using labels
</p>

</td>

</tr>

<tr>

<td>

<code>status</code></br> <em>
<a href="#argoproj.io/v1alpha1.StatusPolicy"> StatusPolicy </a> </em>
</td>

<td>

<p>

Status refers to the policy used to check the state of the trigger using
response status
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TriggerTemplate">

TriggerTemplate
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Trigger">Trigger</a>)
</p>

<p>

<p>

TriggerTemplate is the template that describes trigger specification.
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

Name is a unique name of the action to take.
</p>

</td>

</tr>

<tr>

<td>

<code>conditions</code></br> <em> string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Conditions is the conditions to execute the trigger. For example:
“(dep01 \|\| dep02) && dep04”
</p>

</td>

</tr>

<tr>

<td>

<code>k8s</code></br> <em>
<a href="#argoproj.io/v1alpha1.StandardK8STrigger"> StandardK8STrigger
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

StandardK8STrigger refers to the trigger designed to create or update a
generic Kubernetes resource.
</p>

</td>

</tr>

<tr>

<td>

<code>argoWorkflow</code></br> <em>
<a href="#argoproj.io/v1alpha1.ArgoWorkflowTrigger"> ArgoWorkflowTrigger
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

ArgoWorkflow refers to the trigger that can perform various operations
on an Argo workflow.
</p>

</td>

</tr>

<tr>

<td>

<code>http</code></br> <em> <a href="#argoproj.io/v1alpha1.HTTPTrigger">
HTTPTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

HTTP refers to the trigger designed to dispatch a HTTP request with
on-the-fly constructable payload.
</p>

</td>

</tr>

<tr>

<td>

<code>awsLambda</code></br> <em>
<a href="#argoproj.io/v1alpha1.AWSLambdaTrigger"> AWSLambdaTrigger </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

AWSLambda refers to the trigger designed to invoke AWS Lambda function
with with on-the-fly constructable payload.
</p>

</td>

</tr>

<tr>

<td>

<code>custom</code></br> <em>
<a href="#argoproj.io/v1alpha1.CustomTrigger"> CustomTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

CustomTrigger refers to the trigger designed to connect to a gRPC
trigger server and execute a custom trigger.
</p>

</td>

</tr>

<tr>

<td>

<code>kafka</code></br> <em>
<a href="#argoproj.io/v1alpha1.KafkaTrigger"> KafkaTrigger </a> </em>
</td>

<td>

<p>

Kafka refers to the trigger designed to place messages on Kafka topic.
</p>

</td>

</tr>

<tr>

<td>

<code>nats</code></br> <em> <a href="#argoproj.io/v1alpha1.NATSTrigger">
NATSTrigger </a> </em>
</td>

<td>

<p>

NATS refers to the trigger designed to place message on NATS subject.
</p>

</td>

</tr>

<tr>

<td>

<code>slack</code></br> <em>
<a href="#argoproj.io/v1alpha1.SlackTrigger"> SlackTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Slack refers to the trigger designed to send slack notification message.
</p>

</td>

</tr>

<tr>

<td>

<code>openWhisk</code></br> <em>
<a href="#argoproj.io/v1alpha1.OpenWhiskTrigger"> OpenWhiskTrigger </a>
</em>
</td>

<td>

<em>(Optional)</em>
<p>

OpenWhisk refers to the trigger designed to invoke OpenWhisk action.
</p>

</td>

</tr>

<tr>

<td>

<code>log</code></br> <em> <a href="#argoproj.io/v1alpha1.LogTrigger">
LogTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Log refers to the trigger designed to invoke log the event.
</p>

</td>

</tr>

<tr>

<td>

<code>azureEventHubs</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureEventHubsTrigger">
AzureEventHubsTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AzureEventHubs refers to the trigger send an event to an Azure Event
Hub.
</p>

</td>

</tr>

<tr>

<td>

<code>pulsar</code></br> <em>
<a href="#argoproj.io/v1alpha1.PulsarTrigger"> PulsarTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Pulsar refers to the trigger designed to place messages on Pulsar topic.
</p>

</td>

</tr>

<tr>

<td>

<code>conditionsReset</code></br> <em>
<a href="#argoproj.io/v1alpha1.ConditionsResetCriteria">
\[\]ConditionsResetCriteria </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Criteria to reset the conditons
</p>

</td>

</tr>

<tr>

<td>

<code>azureServiceBus</code></br> <em>
<a href="#argoproj.io/v1alpha1.AzureServiceBusTrigger">
AzureServiceBusTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AzureServiceBus refers to the trigger designed to place messages on
Azure Service Bus
</p>

</td>

</tr>

<tr>

<td>

<code>email</code></br> <em>
<a href="#argoproj.io/v1alpha1.EmailTrigger"> EmailTrigger </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Email refers to the trigger designed to send an email notification
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TriggerType">

TriggerType (<code>string</code> alias)
</p>

</h3>

<p>

<p>

TriggerType is the type of trigger
</p>

</p>

<h3 id="argoproj.io/v1alpha1.Type">

Type (<code>int64</code> alias)
</p>

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.Int64OrString">Int64OrString</a>)
</p>

<p>

<p>

Type represents the stored type of Int64OrString.
</p>

</p>

<h3 id="argoproj.io/v1alpha1.URLArtifact">

URLArtifact
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.ArtifactLocation">ArtifactLocation</a>)
</p>

<p>

<p>

URLArtifact contains information about an artifact at an http endpoint.
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

<code>path</code></br> <em> string </em>
</td>

<td>

<p>

Path is the complete URL
</p>

</td>

</tr>

<tr>

<td>

<code>verifyCert</code></br> <em> bool </em>
</td>

<td>

<p>

VerifyCert decides whether the connection is secure or not
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.ValueFromSource">

ValueFromSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.SecureHeader">SecureHeader</a>)
</p>

<p>

<p>

ValueFromSource allows you to reference keys from either a Configmap or
Secret
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

<code>secretKeyRef</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

</td>

</tr>

<tr>

<td>

<code>configMapKeyRef</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#configmapkeyselector-v1-core">
Kubernetes core/v1.ConfigMapKeySelector </a> </em>
</td>

<td>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.WatchPathConfig">

WatchPathConfig
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.FileEventSource">FileEventSource</a>,
<a href="#argoproj.io/v1alpha1.HDFSEventSource">HDFSEventSource</a>,
<a href="#argoproj.io/v1alpha1.SFTPEventSource">SFTPEventSource</a>)
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

<code>directory</code></br> <em> string </em>
</td>

<td>

<p>

Directory to watch for events
</p>

</td>

</tr>

<tr>

<td>

<code>path</code></br> <em> string </em>
</td>

<td>

<p>

Path is relative path of object to watch with respect to the directory
</p>

</td>

</tr>

<tr>

<td>

<code>pathRegexp</code></br> <em> string </em>
</td>

<td>

<p>

PathRegexp is regexp of relative path of object to watch with respect to
the directory
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.WebhookContext">

WebhookContext
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.BitbucketEventSource">BitbucketEventSource</a>,
<a href="#argoproj.io/v1alpha1.BitbucketServerEventSource">BitbucketServerEventSource</a>,
<a href="#argoproj.io/v1alpha1.GerritEventSource">GerritEventSource</a>,
<a href="#argoproj.io/v1alpha1.GithubEventSource">GithubEventSource</a>,
<a href="#argoproj.io/v1alpha1.GitlabEventSource">GitlabEventSource</a>,
<a href="#argoproj.io/v1alpha1.SNSEventSource">SNSEventSource</a>,
<a href="#argoproj.io/v1alpha1.SlackEventSource">SlackEventSource</a>,
<a href="#argoproj.io/v1alpha1.StorageGridEventSource">StorageGridEventSource</a>,
<a href="#argoproj.io/v1alpha1.StripeEventSource">StripeEventSource</a>,
<a href="#argoproj.io/v1alpha1.WebhookEventSource">WebhookEventSource</a>)
</p>

<p>

<p>

WebhookContext holds a general purpose REST API context
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

<code>endpoint</code></br> <em> string </em>
</td>

<td>

<p>

REST API endpoint
</p>

</td>

</tr>

<tr>

<td>

<code>method</code></br> <em> string </em>
</td>

<td>

<p>

Method is HTTP request method that indicates the desired action to be
performed for a given resource. See RFC7231 Hypertext Transfer Protocol
(HTTP/1.1): Semantics and Content
</p>

</td>

</tr>

<tr>

<td>

<code>port</code></br> <em> string </em>
</td>

<td>

<p>

Port on which HTTP server is listening for incoming events.
</p>

</td>

</tr>

<tr>

<td>

<code>url</code></br> <em> string </em>
</td>

<td>

<p>

URL is the url of the server.
</p>

</td>

</tr>

<tr>

<td>

<code>serverCertSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

ServerCertPath refers the file that contains the cert.
</p>

</td>

</tr>

<tr>

<td>

<code>serverKeySecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<p>

ServerKeyPath refers the file that contains private key
</p>

</td>

</tr>

<tr>

<td>

<code>metadata</code></br> <em> map\[string\]string </em>
</td>

<td>

<em>(Optional)</em>
<p>

Metadata holds the user defined metadata which will passed along the
event payload.
</p>

</td>

</tr>

<tr>

<td>

<code>authSecret</code></br> <em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

AuthSecret holds a secret selector that contains a bearer token for
authentication
</p>

</td>

</tr>

<tr>

<td>

<code>maxPayloadSize</code></br> <em> int64 </em>
</td>

<td>

<em>(Optional)</em>
<p>

MaxPayloadSize is the maximum webhook payload size that the server will
accept. Requests exceeding that limit will be rejected with “request too
large” response. Default value: 1048576 (1MB).
</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.WebhookEventSource">

WebhookEventSource
</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>)
</p>

<p>

<p>

CalendarEventSource describes an HTTP based EventSource
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

<code>WebhookContext</code></br> <em>
<a href="#argoproj.io/v1alpha1.WebhookContext"> WebhookContext </a>
</em>
</td>

<td>

<p>

(Members of <code>WebhookContext</code> are embedded into this type.)
</p>

</td>

</tr>

<tr>

<td>

<code>filter</code></br> <em>
<a href="#argoproj.io/v1alpha1.EventSourceFilter"> EventSourceFilter
</a> </em>
</td>

<td>

<em>(Optional)</em>
<p>

Filter
</p>

</td>

</tr>

</tbody>

</table>

<hr/>

<p>

<em> Generated with <code>gen-crd-api-reference-docs</code>. </em>
</p>
