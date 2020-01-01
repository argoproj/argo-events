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
github.com/argoproj/argo-events/common.Backoff </em>

</td>

<td>

<em>(Optional)</em>

<p>

Backoff holds parameters applied to connection.

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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.CalendarEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.FileEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.ResourceEventSource
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
map\[string\]github.com/argoproj/argo-events/gateways/server/common/webhook.Context
</em>

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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.AMQPEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.KafkaEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.MQTTEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.NATSEventsSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.SNSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.SQSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.PubSubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.GithubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.GitlabEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.HDFSEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.SlackEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.StorageGridEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.AzureEventsHubEventSource
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
map\[string\]github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1.StripeEventSource
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

<code>type</code></br> <em> Argo Events common.EventSourceType </em>

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
github.com/argoproj/argo-events/gateways/server/common/fsevent.WatchPathConfig
</em>

</td>

<td>

<p>

WatchPathConfig contains configuration about the file path to watch

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

<code>webhook</code></br> <em>
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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

<code>webhook</code></br> <em>
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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
github.com/argoproj/argo-events/gateways/server/common/fsevent.WatchPathConfig
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
github.com/argoproj/argo-events/common.Backoff </em>

</td>

<td>

<p>

Backoff holds parameters applied to connection.

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
github.com/argoproj/argo-events/common.Backoff </em>

</td>

<td>

<p>

ConnectionBackoff holds backoff applied to connection.

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
github.com/argoproj/argo-events/common.Backoff </em>

</td>

<td>

<p>

ConnectionBackoff holds backoff applied to connection.

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

Filter is applied on the metadata of the resource

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

<code>eventType</code></br> <em>
<a href="#argoproj.io/v1alpha1.ResourceEventType"> ResourceEventType
</a> </em>

</td>

<td>

<em>(Optional)</em>

<p>

Type is the event type. If not provided, the gateway will watch all
events for a resource.

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

</td>

</tr>

<tr>

<td>

<code>labels</code></br> <em> map\[string\]string </em>

</td>

<td>

<em>(Optional)</em>

</td>

</tr>

<tr>

<td>

<code>fields</code></br> <em> map\[string\]string </em>

</td>

<td>

<em>(Optional)</em>

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
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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

<code>webhook</code></br> <em>
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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

<code>namespace</code></br> <em> string </em>

</td>

<td>

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

<code>webhook</code></br> <em>
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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

<code>webhook</code></br> <em>
github.com/argoproj/argo-events/gateways/server/common/webhook.Context
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

<hr/>

<p>

<em> Generated with <code>gen-crd-api-reference-docs</code> on git
commit <code>6ce129b</code>. </em>

</p>
