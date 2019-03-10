# Community Gateways

## How to set up gateways that run web server internally?
For any gateway that run web server internally, you need to create the gateway resource before creating gateway configmap.
Why? Because the `webhook` configuration in gateway configmap contains the `url` which the URL of the service that exposes the gateway pod, you need to have
the gateway pod and service running before you can create the configmap. 

  ### Example
  
  Lets setup AWS SNS gateway,
  
  1. Create the gateway resource, 
     ```bash
     kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/aws-sns.yaml
     ```
     
     Once the gateway resource is created, you should see the corresponding gateway pod and service being created.
     Make sure the gateway service is being backed by the gateway pod.
     
     How to get the URL for that resolves to your service is up to you. You can use Ingress controller, Route (if your installation is on Openshift) or other means.
     
  2. Create the gateway configmap. The gateway configmap basically contains the event source configurations. You need to put the URL of your service. 
  
     ```bash
     kubectl -n argo-events create -f  ../../examples/gateway/aws-sns-gateway-configmap.yaml
     ```
     
  3. Create the sensor.
  
     ```bash
     kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/aws-sns.yaml
     ```
     
  Now your setup is complete.
