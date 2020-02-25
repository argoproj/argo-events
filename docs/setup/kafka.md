# Kafka Gateway & Sensor

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/docs-gateway-setup/docs/assets/kafka-setup.png?raw=true" alt="KAFKA Setup"/>
</p>

<br/>
<br/>

## Event Structure
The structure of an event dispatched by the gateway to the sensor looks like following,

        {
            "context": {
              "type": "type_of_gateway",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_gateway",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_event_within_event_source"
            },
            "data": {
              "subject": "name_of_the_nats_subject",
              "data": "message_payload"
            }
        }


