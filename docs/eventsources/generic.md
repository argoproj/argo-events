# Generic EventSource

Generic eventsource extends Argo-Events eventsources via a simple contract. This is specifically useful when you want
to onboard a custom eventsource implementation.

## Interface

In order to qualify as generic eventsource, the eventsource server needs to implement following gRPC contract,

        syntax = "proto3";
        
        package generic;
        
        service Eventing {
            rpc StartEventSource(EventSource) returns (stream Event);
        }
        
        message EventSource {
            // The event source name.
            string name = 1;
            // The event source configuration value.
            bytes config = 2;
        }
        
        /**
        * Represents an event
        */
        message Event {
            // The event source name.
            string name = 1;
            // The event payload.
            bytes payload = 2;
        }

The proto file is available [here](https://github.com/argoproj/argo-events/blob/master/eventsources/sources/generic/generic.proto).

## Architecture

<br/>
<br/>

![arch](../assets/generic-eventsource.png)

<br/>

## Example

