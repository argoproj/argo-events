# Standard K8s Resources
In the previous sections, you saw how to trigger the Argo workflows. In this tutorial, you 
will see how to trigger Pod and Deployment.

Similarly you can trigger any standard Kubernetes resources.

Having the ability to trigger standard Kubernetes resources is quite powerful as it gives ability to
set up pipelines for existing workloads.



## Prerequisites
1. Make sure that `argo-events-sa` service account has necessary permissions to 
create the Kubernetes resource of your choice.
2. The `Webhook` gateway is already set up.

## Pod

1. Create a sensor with K8s trigger. Pay close attention to the `group`, `version` and `kind`
   keys within the trigger resource. These keys determine the type of kubernetes object.
   
   You will notice that the `group` key is empty, that means we want to use `core` group.
   For any other groups, you need to specify the `group` key.
   
   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-standard-k8s-resources/sensor-pod.yaml
   ```

2. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`
   
   ```bash
   curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```
   
3. Now, you should see a pod being created.
   
   ```bash
   kubectl -n argo-events get po
   ```
  
  Output
  
  ```
   _________________________________________ 
  / {"context":{"type":"webhook","specVersi \
  | on":"0.3","source":"webhook-gateway","e |
  | ventID":"30306463666539362d346666642d34 |
  | 3336332d383861312d336538363333613564313 |
  | 932","time":"2020-01-11T21:23:07.682961 |
  | Z","dataContentType":"application/json" |
  | ,"subject":"example"},"data":"eyJoZWFkZ |
  | XIiOnsiQWNjZXB0IjpbIiovKiJdLCJDb250ZW50 |
  | LUxlbmd0aCI6WyIxOSJdLCJDb250ZW50LVR5cGU |
  | iOlsiYXBwbGljYXRpb24vanNvbiJdLCJVc2VyLU |
  | FnZW50IjpbImN1cmwvNy41NC4wIl19LCJib2R5I |
  \ jp7Im1lc3NhZ2UiOiJoZXkhISJ9fQ=="}       /
   ----------------------------------------- 
      \
       \
        \     
                      ##        .            
                ## ## ##       ==            
             ## ## ## ##      ===            
         /""""""""""""""""___/ ===        
    ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
         \______ o          __/            
          \    \        __/             
            \____\______/   
  ```

## Deployment
1. Lets create a sensor with a K8s deployment as trigger.
   
   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-standard-k8s-resources/sensor-deployment.yaml
   ```


2. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`
   
   ```bash
   curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```
   
3. Now, you should see a deployment being created. Get the corresponding pod.
   
   ```bash
   kubectl -n argo-events get deployments
   ```

  Output
  
  ```
   _________________________________________ 
  / {"context":{"type":"webhook","specVersi \
  | on":"0.3","source":"webhook-gateway","e |
  | ventID":"30306463666539362d346666642d34 |
  | 3336332d383861312d336538363333613564313 |
  | 932","time":"2020-01-11T21:23:07.682961 |
  | Z","dataContentType":"application/json" |
  | ,"subject":"example"},"data":"eyJoZWFkZ |
  | XIiOnsiQWNjZXB0IjpbIiovKiJdLCJDb250ZW50 |
  | LUxlbmd0aCI6WyIxOSJdLCJDb250ZW50LVR5cGU |
  | iOlsiYXBwbGljYXRpb24vanNvbiJdLCJVc2VyLU |
  | FnZW50IjpbImN1cmwvNy41NC4wIl19LCJib2R5I |
  \ jp7Im1lc3NhZ2UiOiJoZXkhISJ9fQ=="}       /
   ----------------------------------------- 
      \
       \
        \     
                      ##        .            
                ## ## ##       ==            
             ## ## ## ##      ===            
         /""""""""""""""""___/ ===        
    ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
         \______ o          __/            
          \    \        __/             
            \____\______/   
  ```
