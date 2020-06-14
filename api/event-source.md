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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

SharedAccessKey is the the generated value of the key

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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace refers to Kubernetes namespace which is used to retrieve the
shared access key and name from.

</p>

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

<code>userPayload</code></br> <em> encoding/json.RawMessage </em>

</td>

<td>

<em>(Optional)</em>

<p>

UserPayload will be sent to sensor as extra data once the event is
triggered

</p>

</td>

</tr>

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.Context">

Context

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.EventSourceSpec">EventSourceSpec</a>,
<a href="#argoproj.io/v1alpha1.GithubEventSource">GithubEventSource</a>,
<a href="#argoproj.io/v1alpha1.GitlabEventSource">GitlabEventSource</a>,
<a href="#argoproj.io/v1alpha1.SNSEventSource">SNSEventSource</a>,
<a href="#argoproj.io/v1alpha1.SlackEventSource">SlackEventSource</a>,
<a href="#argoproj.io/v1alpha1.StorageGridEventSource">StorageGridEventSource</a>,
<a href="#argoproj.io/v1alpha1.StripeEventSource">StripeEventSource</a>)

</p>

<p>

<p>

Context holds a general purpose REST API context

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

<code>serverCertPath</code></br> <em> string </em>

</td>

<td>

<p>

ServerCertPath refers the file that contains the cert.

</p>

</td>

</tr>

<tr>

<td>

<code>serverKeyPath</code></br> <em> string </em>

</td>

<td>

<p>

ServerKeyPath refers the file that contains private key

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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace to use to retrieve the channel key and optional
username/password

</p>

</td>

</tr>

<tr>

<td>

<code>username</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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
<a href="#argoproj.io/v1alpha1.EventSourceStatus"> EventSourceStatus
</a> </em>

</td>

<td>

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

<code>minio</code></br> <em>
map\[string\]github.com/argoproj/argo-events/pkg/apis/common.S3Artifact
</em>

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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.CalendarEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.FileEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.ResourceEventSource
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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.Context
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.AMQPEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.KafkaEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.MQTTEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.NATSEventsSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SNSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SQSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.PubSubEventSource
</a> </em>

</td>

<td>

<p>

PubSub eevnt sources

</p>

</td>

</tr>

<tr>

<td>

<code>github</code></br> <em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GithubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GitlabEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.HDFSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SlackEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.StorageGridEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.AzureEventsHubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.StripeEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.EmitterEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.RedisEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.NSQEventSource
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

<code>generic</code></br> <em>
<a href="#argoproj.io/v1alpha1.GenericEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GenericEventSource
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

<code>type</code></br> <em>
github.com/argoproj/argo-events/pkg/apis/common.EventSourceType </em>

</td>

<td>

<p>

Type of the event source

</p>

</td>

</tr>

</table>

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

<code>minio</code></br> <em>
map\[string\]github.com/argoproj/argo-events/pkg/apis/common.S3Artifact
</em>

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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.CalendarEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.FileEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.ResourceEventSource
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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.Context
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.AMQPEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.KafkaEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.MQTTEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.NATSEventsSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SNSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SQSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.PubSubEventSource
</a> </em>

</td>

<td>

<p>

PubSub eevnt sources

</p>

</td>

</tr>

<tr>

<td>

<code>github</code></br> <em>
<a href="#argoproj.io/v1alpha1.GithubEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GithubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GitlabEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.HDFSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.SlackEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.StorageGridEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.AzureEventsHubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.StripeEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.EmitterEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.RedisEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.NSQEventSource
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

<code>generic</code></br> <em>
<a href="#argoproj.io/v1alpha1.GenericEventSource">
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1.GenericEventSource
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

<code>type</code></br> <em>
github.com/argoproj/argo-events/pkg/apis/common.EventSourceType </em>

</td>

<td>

<p>

Type of the event source

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

<code>createdAt</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time </a> </em>

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

<code>value</code></br> <em> string </em>

</td>

<td>

<p>

Value of the event source

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

<p>

Id is the webhook’s id

</p>

</td>

</tr>

<tr>

<td>

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

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

<p>

Owner refers to GitHub owner name i.e. argoproj

</p>

</td>

</tr>

<tr>

<td>

<code>repository</code></br> <em> string </em>

</td>

<td>

<p>

Repository refers to GitHub repo name i.e. argo-events

</p>

</td>

</tr>

<tr>

<td>

<code>events</code></br> <em> \[\]string </em>

</td>

<td>

<p>

Events refer to Github events to subscribe to which the gateway will
subscribe

</p>

</td>

</tr>

<tr>

<td>

<code>apiToken</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

APIToken refers to a K8s secret containing github api token

</p>

</td>

</tr>

<tr>

<td>

<code>webhookSecret</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://developer.github.com/webhooks/creating/#active">https://developer.github.com/webhooks/creating/\#active</a>

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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace refers to Kubernetes namespace which is used to retrieve
webhook secret and api token from.

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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

</td>

<td>

<p>

Webhook holds configuration to run a http server

</p>

</td>

</tr>

<tr>

<td>

<code>projectId</code></br> <em> string </em>

</td>

<td>

<p>

ProjectId is the id of project for which integration needs to setup

</p>

</td>

</tr>

<tr>

<td>

<code>event</code></br> <em> string </em>

</td>

<td>

<p>

Event is a gitlab event to listen to. Refer
<a href="https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794">https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go\#L794</a>.

</p>

</td>

</tr>

<tr>

<td>

<code>accessToken</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

AccessToken is reference to k8 secret which holds the gitlab api access
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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace refers to Kubernetes namespace which is used to retrieve
access token from.

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

<code>allowDuplicate</code></br> <em> bool </em>

</td>

<td>

<p>

AllowDuplicate allows the gateway to register the same webhook
integrations for multiple event source configurations. Defaults to
false.

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

<p>

Addresses is accessible addresses of HDFS name nodes

</p>

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#configmapkeyselector-v1-core">
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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace refers to Kubernetes namespace which is used to retrieve cache
secret and ket tab secret from.

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

URL to kafka cluster

</p>

</td>

</tr>

<tr>

<td>

<code>partition</code></br> <em> string </em>

</td>

<td>

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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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

</tbody>

</table>

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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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

NATSEventSource refers to event-source for NATS related events

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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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
github.com/argoproj/argo-events/pkg/apis/common.Backoff </em>

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

<p>

ProjectID is the unique identifier for your project on GCP

</p>

</td>

</tr>

<tr>

<td>

<code>topicProjectID</code></br> <em> string </em>

</td>

<td>

<p>

TopicProjectID identifies the project where the topic should exist or be
created (assumed to be the same as ProjectID by default)

</p>

</td>

</tr>

<tr>

<td>

<code>topic</code></br> <em> string </em>

</td>

<td>

<p>

Topic on which a subscription will be created

</p>

</td>

</tr>

<tr>

<td>

<code>credentialsFile</code></br> <em> string </em>

</td>

<td>

<p>

CredentialsFile is the file that contains credentials to authenticate
for GCP

</p>

</td>

</tr>

<tr>

<td>

<code>enableWorkflowIdentity</code></br> <em> bool </em>

</td>

<td>

<em>(Optional)</em>

<p>

EnableWorkflowIdentity determines if your project authenticates to GCP
with WorkflowIdentity or CredentialsFile. If true, authentication is
done with WorkflowIdentity. If false or omitted, authentication is done
with CredentialsFile.

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

</tbody>

</table>

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
<a href="https://godoc.org/github.com/go-redis/redis#example-PubSub">https://godoc.org/github.com/go-redis/redis\#example-PubSub</a>

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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

<p>

Channels to subscribe to listen events.

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#groupversionresource-v1-meta">
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

ResourceFilter contains K8 ObjectMeta information to further filter
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
for more info.

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

Fields provide listing options to K8s API to watch resource/s. Refer
<a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/">https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/</a>
for more info.

</p>

</td>

</tr>

<tr>

<td>

<code>createdBy</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

AccessKey refers K8 secret containing aws access key

</p>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

SecretKey refers K8 secret containing aws secret key

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

Namespace refers to Kubernetes namespace to read access related secret
from.

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

AccessKey refers K8 secret containing aws access key

</p>

</td>

</tr>

<tr>

<td>

<code>secretKey</code></br> <em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector </a> </em>

</td>

<td>

<p>

SecretKey refers K8 secret containing aws secret key

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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace refers to Kubernetes namespace to read access related secret
from.

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

QueueAccountId is the ID of the account that created the queue to
monitor

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

Supported operations like ==, \!=, \<=, \>= etc. Defaults to ==. Refer
<a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors">https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/\#label-selectors</a>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

</td>

<td>

<p>

Webhook holds configuration for a REST endpoint

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

Namespace refers to Kubernetes namespace which is used to retrieve token
and signing secret from.

</p>

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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

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

<p>

Events are s3 bucket notification events. For more information on s3
notifications, follow
<a href="https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations">https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html\#notification-how-to-event-types-and-destinations</a>
Note that storage grid notifications do not contain <code>s3:</code>

</p>

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

Filter represents filters to apply to bucket notifications for
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

<code>webhook</code></br> <em> <a href="#argoproj.io/v1alpha1.Context">
Context </a> </em>

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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#secretkeyselector-v1-core">
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

<code>namespace</code></br> <em> string </em>

</td>

<td>

<em>(Optional)</em>

<p>

Namespace to retrieve the APIKey secret from. Must be specified in order
to read API key from APIKey K8s secret.

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

</tbody>

</table>

<h3 id="argoproj.io/v1alpha1.TLSConfig">

TLSConfig

</h3>

<p>

(<em>Appears on:</em>
<a href="#argoproj.io/v1alpha1.AMQPEventSource">AMQPEventSource</a>,
<a href="#argoproj.io/v1alpha1.EmitterEventSource">EmitterEventSource</a>,
<a href="#argoproj.io/v1alpha1.KafkaEventSource">KafkaEventSource</a>,
<a href="#argoproj.io/v1alpha1.MQTTEventSource">MQTTEventSource</a>,
<a href="#argoproj.io/v1alpha1.NATSEventsSource">NATSEventsSource</a>,
<a href="#argoproj.io/v1alpha1.NSQEventSource">NSQEventSource</a>,
<a href="#argoproj.io/v1alpha1.RedisEventSource">RedisEventSource</a>)

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

<code>caCertPath</code></br> <em> string </em>

</td>

<td>

<p>

CACertPath refers the file path that contains the CA cert.

</p>

</td>

</tr>

<tr>

<td>

<code>clientCertPath</code></br> <em> string </em>

</td>

<td>

<p>

ClientCertPath refers the file path that contains client cert.

</p>

</td>

</tr>

<tr>

<td>

<code>clientKeyPath</code></br> <em> string </em>

</td>

<td>

<p>

ClientKeyPath refers the file path that contains client key.

</p>

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
<a href="#argoproj.io/v1alpha1.HDFSEventSource">HDFSEventSource</a>)

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

<hr/>

<p>

<em> Generated with <code>gen-crd-api-reference-docs</code>. </em>

</p>
