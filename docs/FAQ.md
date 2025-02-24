# FAQs

**Q. How to get started with Argo Events?**

**A**. The recommended way to get started with Argo Events is:

 1. Read the basic concepts about [EventBus](https://argoproj.github.io/argo-events/concepts/eventbus/), [Sensor](https://argoproj.github.io/argo-events/concepts/sensor/) and [Event Source](https://argoproj.github.io/argo-events/concepts/event_source/).
 2. Install Argo Events as outlined [here](https://argoproj.github.io/argo-events/installation/).
 3. Read the tutorials available [here](https://argoproj.github.io/argo-events/tutorials/01-introduction/).

**Q. Can I deploy event-source and sensor in a namespace different than `argo-events`?**

**A**. Yes. If you want to deploy the event-source in a different namespace than `argo-events`, please update the event-source definition
with the desired namespace and service account. Make sure to grant the service account the [necessary roles](https://github.com/argoproj/argo-events/blob/master/manifests/namespace-install.yaml).

**Q. How to debug Argo-Events.**

**A**.

1. Make sure you have installed everything as instructed [here](https://argoproj.github.io/argo-events/installation/).
1. Make sure you have the `EventBus` resource created within the namespace.
1. The event-bus, event-source and sensor pods must be running. If you see any issue with the pods, check the logs
   for sensor-controller, event-source-controller and event-bus-controller.
1. If event-source and sensor pods are running, but you are not receiving any events:
     * Make sure you have configured the event source correctly.
     * Check the event-source pod's containers logs.

Note: You can set the environment variable `LOG_LEVEL:info/debug/error` in any of the containers to output debug logs. See [here](https://github.com/argoproj/argo-events/blob/master/examples/sensors/log-debug.yaml) for a debug example.

**Q. The event-source pod is receiving events but nothing happens.**

**A**.

1. Check the sensor resource is deployed and a pod is running for the resource.
If the sensor pod is running, check for `Started to subscribe events for triggers` in the logs.
If the sensor has subscribed to the event-bus but is unable to create the trigger resource, please raise an issue on GitHub.

2. The sensor's dependencies have a specific eventSourceName and eventName that should match the values defined in the `EventSource` resource. See full details [here](https://github.com/argoproj/argo-events/blob/master/docs/eventsources/naming.md).

**Q. Helm chart installation does not work.**

**A.** The Helm chart for argo events is maintained by the community and can be out of sync with latest release version.
The official installation file is available [here](https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml).
If you notice the Helm chart is outdated, we encourage you to contribute to the [argo-helm](https://github.com/argoproj/argo-helm) repository on GitHub.

**Q. Kustomization file doesn't have a `X` resource.**

**A.** The kustomization.yaml file is maintained by the community. If you notice that it is out of sync with the official installation file, please
raise a PR.

**Q. Can I use the Minio event-source for AWS S3 notifications?**

**A.** No. The Minio event-source is exclusively for use only with Minio servers. If you want to trigger workloads on an AWS S3 bucket notification,
set up the AWS SNS event-source.

**Q. If I have multiple event dependencies and triggers in a single sensor, can I execute a specific trigger upon a specific event?**  

**A.** Yes, this functionality is offered by the sensor event resolution circuitry. Please take a look at the [Circuit and Switch](https://argoproj.github.io/argo-events/tutorials/06-circuit-and-switch/) tutorial.

**Q. The latest image tag does not point to latest release tag?**

**A.** When it comes to image tags, the golden rule is _do not trust the latest tag_. Always use the pinned version of the images.
   We will try to keep the `latest` tag in sync with the most recently released version.

**Q. Where can I find the event structure for a particular event-source?**

**A.** Please refer to [this file](https://github.com/argoproj/argo-events/blob/master/pkg/apis/events/v1alpha1/eventsource_types.go) to understand the structure
of different types of events dispatched by the event-source pod.
