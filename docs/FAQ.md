# FAQs

**Q. How to get started with Argo Events?**

**A**. Recommended way to get started with Argo Events is,

 1. Read the basic concepts about [Gateway](https://argoproj.github.io/argo-events/concepts/gateway/), [Sensor](https://argoproj.github.io/argo-events/concepts/sensor/) and [Event Source](https://argoproj.github.io/argo-events/concepts/event_source/).
 2. Install the setup as outlined [here](https://argoproj.github.io/argo-events/installation/).
 3. Read the tutorials available [here](https://argoproj.github.io/argo-events/tutorials/01-introduction/). 

**Q. Can I deploy gateway and sensor in a namespace different that `argo-events`?**

**A**.   Yes. If you want to deploy the gateway in a different namespace that `argo-events`, then please update the
gateway definition with desired namespace and service account. Make sure to grant the service account the necessary roles.
Also note that the gateway and sensor controllers are configured to process the gateway and sensor resources
in `argo-events` namespace with instance-id `argo-events`. You can change the configuration by updating the
appropriate controller configmap. 

**Q. How to debug Argo-Events.**

**A**.
 
1. Make sure you have installed everything as instructed [here](https://argoproj.github.io/argo-events/installation/).
1. The gateway and sensor pods must be running. If you see any issue with the pods, check the logs
   for sensor-controller and gateway-controller.
1. If gateway and sensor pods are running, but you are not receiving any events:
     * Make sure you have configured the event source correctly.
     * Check the logs for both of the gateway pod's containers.
1. If the gateway-client displays `dispatched event` but nothing happens then read following Q and A.   

Note: You can set environment variable `DEBUG_LOG:true` in any of the containers to output debug logs.

**Q. Gateway is receiving the events but nothing happens.**

**A**. 

1. Check the sensor resource is deployed and a pod is created for the resource.
If sensor pod is running, check the `subscribers` list in the gateway resource. The sensor service url must be
registered as a subscriber in order to receive events from gateway. The `gateway-client` container should also log an error related to this situation.

1. If the gateway was able to send an event to sensor, then check the sensor logs, either the sensor event resolution circuitry has rejected the event or
the sensor failed to execute the trigger due to an error.

**Q. Helm chart installation does not work.**

**A.** The helm chart for argo events is maintained by the community and can be out of sync with latest release version. The official installation file is available [here](https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/installation.yaml).
If you notice the helm chart is outdated, we encourage you to contribute to the [argo-helm](https://github.com/argoproj/argo-helm).

**Q. Kustomization file doesn't have a `X` resource.**

**A.** The kustomization.yaml file is maintained by the community. If you notice that it is out of sync with the official installation file, please
raise a PR.

**Q. Can I use Minio gateway for AWS S3 notifications?**

**A.** No. Minio gateway is exclusive for the Minio server. If you want to trigger workloads on AWS S3 bucket notification,
   then set up the AWS SNS gateway.

**Q. If I have multiple event dependencies and triggers in a single sensor, can I execute a specific trigger upon a specific event?**  

**A.** Yes, this is precisely the functionality the sensor event resolution circuitry offers. Please take a look at the [Circuit and Switch](https://argoproj.github.io/argo-events/tutorials/06-circuit-and-switch/).

**Q. The latest image tag does not point to latest release tag?**

**A.** When it comes to image tags, the golden rule is do not trust the latest tag. Always use the pinned version of the images.
   We will try to keep the `latest` in sync with the latest release version.

**Q. Where can I find the event structure for a particular gateway?**

**A.** Please refer [this file](https://github.com/argoproj/argo-events/blob/master/pkg/apis/events/event-data.go) to understand the structure of different types of events dispatched by gateways.
