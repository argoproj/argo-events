# Sensor API
The API specifications for Kubernetes Sensor CRDs. See the ./examples directory for applications of this API.

## Sensor v1alpha1 event

| Group     | Version     | Kind     |
|:---------:|:-----------:|:--------:|
| `event`   | `v1alpha1`  | `Sensor` |


| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `apiVersion` | *string*                                 | Defines the versioned schema of this object. For us, this is `event/v1alpha1`. More info [here](https://git.k8s.io/community/contributors/devel/api-conventions.md#resources) |
| `kind`       | *string*                                 | Represents the REST resource this object represents. For us, this is `Sensor`. More info [here](https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds) |
| `metadata`   | *[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#objectmeta-v1-meta)* | ObjectMeta is metadata that all persisted resources must have, which includes all objects users must create. |
| `spec`       | *[SensorSpec](#sensorspec-v1alpha1-event)* | The specification for the sensor and its desired state |
| `status`     | *[SensorStatus](#sensorstatus-v1alpha1-event)* | Information about the most recently observed status of the sensor |


### SensorSpec v1alpha1 event
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `signals`    | *[Signal](#signal-v1alpha1-event)* array                | List of signal dependencies   |
| `triggers`   | *[Trigger](#trigger-v1alpha1-event)* array              | List of trigger actions       |
| `escalation` | *[EscalationPolicy](#escalation-policy)* | Specify the policy for escalating signal failures |
| `repeat`     | *bool*                                   | Should repeat execution of signal resolution and action triggering |


### SensorStatus v1alpha1 event
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `phase`      | *string*                                 | Current condition of the sensor |
| `startedAt`  | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the sensor started listening for events |
| `resolvedAt` | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the sensor successfully resolved all dependent signals |
| `message`    | *string*                                 | Human readable string indicating details about a sensor in its phase |
| `nodes`      | map *string* -> *[NodeStatus](#node-status)* | Mapping between a node ID and the node's status |
| `escalated`  | *bool*                                   | Flag for whether the sensor was escalated |


### Signal v1alpha1 event
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `name`       | *string*                                 | Unique name for this dependencies |
| `deadline`   | *int64*                                  | Duration in seconds after the *startedAt* time in which the signal is terminated (currently not implemented) |
| `nats`       | *[NATS](#nats)*                          | [NATS](https://nats.io/documentation/) stream signal |
| `mqtt`       | *[MQTT](#mqtt)*                          | [MQTT](http://mqtt.org/) stream signal |
| `amqp`       | *[AMQP](#amqp)*                          | [AMQP](https://www.amqp.org/) stream signal |
| `kafka`      | *[KAFKA](#kafka)*                        | [KAFKA](https://kafka.apache.org/) stream signal |
| `artifact`   | *[ArtifactSignal](#artifact-signal)*     | Artifact based signal. currently S3 is the only type supported |
| `calendar`   | *[CalendarSignal](#calendar-signal)*     | Time-based signal |
| `resource`   | *[ResourceSignal](#resource-signal)*     | Kubernetes resource based signal     |
| `constraints`| *[SignalConstraints](#signal-constraints)*  | Rules governing tolerations and accepted event-meta information |


### Trigger v1alpha1 event
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `name`       | *string*                                 | Unique name for this action   |
| `message`    | *[Message](#message)*                    | The message to send on a queue    |
| `resource`     | *[ResourceObject](#resourceobject)*   | The K8s Object to create      |
| `retryStrategy` | *[RetryStrategy](#retry-strategy)*    | Defines a strategy to retry if the trigger fails (currently not yet implemented) |

### Message
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `body`       | *string*                                 | The content of the message    |
| `stream`     | *[Stream](#stream-v1alpha1-event)*       | The stream resource queue on which to send the message |

### Stream v1alpha1 event
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `nats`       | *[NATS](#nats)*                          | [NATS](https://nats.io/documentation/) stream |
| `mqtt`       | *[MQTT](#mqtt)*                          | [MQTT](http://mqtt.org/) stream |
| `amqp`       | *[AMQP](#amqp)*                          | [AMQP](https://www.amqp.org/) stream |
| `kafka`      | *[KAFKA](#kafka)*                        | [KAFKA](https://kafka.apache.org/) stream |

### ResourceObject
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `namespace`  | *string*                                 | The Kubernetes namespace to create this resource object |
| `group`      | *string*                                 | The API Group as specified in the REST path and in the `apiVersion` field of a Kubernetes object, e.g. `core`, `batch`, etc. |
| `version`    | *string*                                 | The version as part of the named group's REST path: `apis/$GROUP_NAME/$VERSION`, and `apiVersion: $GROUP_NAME/$VERSION` |
| `kind`       | *string*                                 | The kind of the Kubernetes resource, as specified in the `TypeMeta` of the object |
| `labels`     | map *string* -> *string*                 | The labels to apply to this resource |
| `s3`         | *[S3](#s3-artifact)*                     | The S3 Artifact location of the resource file |

### NATS
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `url`        | *string*                                 | The url for client connections to the NATS cluster |
| `subject`    | *string*                                 | The subject on which to receive messages |

### MQTT
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `url`        | *string*                                 | The url of the MQTT message broker |
| `topic  `    | *string*                                 | The topic on which to receive messages |

### AMQP
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `url`        | *string*                                 | The url of the AMQP message broker |
| `exchangeName` | *string*                               | The name of the exchange |
| `exchangeType` | *string*                               | The type of the exchange, e.g. direct, fanout |
| `routingKey` | *string*                                 | A list of words delimited by dots. If the exchange is a `topic` exchange, the routingKey must be provided. The routing key specifies some features connected to the messages being received. As many words as you like, up to the limit of 255 bytes. |

### Kafka
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `url`        | *string*                                 | The url of the Kafka message broker |
| `topic`      | *string*                                 | The topic on which to consume messages |
| `partition`  | *int32*                                  | The partition on which to consume messages. If you wish to gaurantee consistency and availability, partitions only work as long as you are producing to one partition and consuming from one partition. This means that a single consumer should be assigned to read from this specific partition. For more information, see this [blog post](https://sookocheff.com/post/kafka/kafka-in-a-nutshell/) |


### Artifact Signal
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `s3`         | *[S3Artifact](#s3-artifact)              | An S3 artifact                |
| `stream`     | *[Stream](#stream-v1alpha1-event)        | The stream to listen for artifact notifications |


### Calendar Signal
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `schedule`   | *string*                                 | A [cron](https://en.wikipedia.org/wiki/Cron) |
| `interval`   | *string*                                 | A interval duration defined in Go. This is parsed using golang time library's [ParseDuration](https://golang.org/pkg/time/#ParseDuration) function |
| `recurrence` | *string* array                           | List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545. This feature is not yet implemented. |


### Resource Signal
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `namespace`  | *string*                                 | The Kubernetes namespace to watch for these resources |
| `group`      | *string*                                 | The API Group as specified in the REST path and in the `apiVersion` field of a Kubernetes object, e.g. `core`, `batch`, etc. |
| `version`    | *string*                                 | The version as part of the named group's REST path: `apis/$GROUP_NAME/$VERSION`, and `apiVersion: $GROUP_NAME/$VERSION` |
| `kind`       | *string*                                 | The kind of the Kubernetes resource, as specified in the `TypeMeta` of the object |


### Signal Constraints
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `time`       | *[TimeConstraints](#time-constraints)*   | The time constraints for this dependency |    


### Time Constraints
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `start`      | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the sensor should start accepting events from the specified dependency |
| `stop`       | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the sensor should stop accepting events from the specified dependency |


### Escalation Policy
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `level`      | *string*                                 | Degree of importance for this sensor |
| `trigger`    | *[Trigger](#trigger-v1alpha1-event)*     | The trigger (action) to fire to escalate this issue |


### Node Status
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `id`         | *string*                                 | Unique identifier for this sensor node |
| `name`       | *string*                                 | Name of the node (signal\trigger) used to generate the `id` |
| `displayName` | *string*                                | Human readable representation of the node |
| `type`       | *string*                                 | The type of the Node, e.g. `signal` or `trigger` |
| `phase`      | *string*                                 | The most current status of the node |
| `startedAt`  | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the node started processing |
| `resolvedAt` | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time at which the node finished processing |
| `message`    | *string*                                 | Human readable message indicating information and or explanation about the status of the node |


### S3 Artifact
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `endpoint`   | *string*                                 | The URL of the S3 object storage server |
| `key`        | *string*                                 | The name of the key of the object |
| `bucket`     | *string*                                 | The name of the S3 bucket     |
| `event`      | *string*                                 | The type of the notification event, e.g. "s3:ObjectCreated:*", "s3:ObjectCreated:Put*, etc... |
| `filter`     | *[S3Filter](#s3-filter)                  | An additional filter applied on S3 object notifications |

### S3 Filter
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `prefix`     | *string*                                 | The string literal prefix to match for S3 objects |
| `suffix`     | *string*                                 | The string literal suffix to match for S3 objects |

### Resource Filter
| Field        | Type                                     | Description                   |
|--------------|------------------------------------------|-------------------------------|
| `prefix`     | *string*                                 | The string literal prefix to match for resource object meta name |
| `labels`     | map *string* -> *string*                 | Labels is an unstructured key value map that is used as selectors in finding this resource |
| `annotations`| map *string* -> *string*                 | Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. |
| `createdBy`  | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#time-v1-meta)* | RFC 3339 date and time after which this resource should have been created |