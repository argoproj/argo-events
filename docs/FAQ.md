# FAQ

**Q**. **Can I deploy gateway and sensor in a namespace different that `argo-events`?**

**A**.   Yes. If you want to deploy the gateway in a different namespace that `argo-events`, then please update the
gateway definition with desired namespace and service account. Make sure to grant the service account the necessary roles.
Also note that the gateway and sensor controllers are configured to process the gateway and sensor resources
in `argo-events` namespace with instance-id `argo-events`. You can change the configuration by updating the
appropriate controller configmap. 

**Q**. **Gateway is receiving the events but nothing happens.**

**A**. First, check the sensor resource is deployed and a pod is created for the resource.
       If sensor pod is running, check the `subscribers` list in the gateway resource. The sensor service url must be
       registered as a subscriber in order to receive events from gateway. The `gateway-client` container should also log an error related to this situation.
       Second, if the gateway was able to send an event to sensor, then check the sensor logs, either the sensor event resolution circuitry has rejected the event or
       the sensor failed to execute the trigger due to an error.
  