# Trigger Sources
A trigger source is the source of trigger resource. It can be either external source such
as `Git`, `S3`, `K8s Configmap`, `File`, any valid `URL` that hosts the resource or an internal resource
which is defined in the sensor object itself like `Inline` or `Resource`. 

In the previous sections, you have been dealing with the `Resource` trigger source. In this tutorial, we
will explore other trigger sources.

## Prerequisites
1. The `Webhook` gateway is already set up.

## Git
Git trigger source refers for K8s trigger refers to the K8s resource stored in Git. 

The specification for the Git source is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#argoproj.io/v1alpha1.GitArtifact).

1. In order to fetch data from git, you need to set up the SSH key in sensor.

2. If you don't have ssh keys available, create them following this [guide](https://help.github.com/en/github/authenticating-to-github/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent)

3. Create a K8s secret that holds the SSH keys

   ```bash
   kubectl -n argo-events create secret generic git-ssh --from-file=key=.ssh/<YOUR_SSH_KEY_FILE_NAME>
   ```

4. Create a K8s secret that holds known hosts.

   ```bash
   kubectl -n argo-events create secret generic git-known-hosts --from-file=ssh_known_hosts=.ssh/known_hosts
   ```

5. Create a sensor with the git trigger source and refer it to the `hello world` worklfow stored
   on the Argo Git project

    ```bash
    kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-trigger-sources/sensor-git.yaml 
    ```   

6. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`
   
   ```bash
   curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```
   
7. Now, you should see an Argo workflow being created.
   
   ```bash
   kubectl -n argo-events get wf
   ```

## S3
You can refer to the K8s resource stored on S3 complaint store as the trigger source.

For this tutorial, lets set up a minio server which is S3 compliant store.

1. Create a K8s secret called `artifacts-minio` that holds your minio access key and secret key.
   The access key must be stored under `accesskey` key and secret key must be stored under
   `secretkey`.

2. Follow steps described [here](https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/k8s-yaml.md#minio-standalone-server-deployment) to set up the minio server.

3. Make sure a service is available to expose the minio server.

4. Create a bucket called `workflows` and store a basic `hello world` Argo workflow with key name `hello-world.yaml`.

5. Create the sensor with trigger source as S3.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-trigger-sources/sensor-minio.yaml
   ```

6. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`
   
   ```bash
   curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```
   
7. Now, you should see an Argo workflow being created.
   
   ```bash
   kubectl -n argo-events get wf
   ```

## K8s Configmap
K8s configmap can be treated as trigger sources if needed.

1. Lets create a configmap called `trigger-store`.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-trigger-sources/trigger-store.yaml
   ```
   
2. Create a sensor with trigger source as configmap and refer it to the `trigger-store`.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/03-trigger-sources/sensor-cm.yaml
   ```
   
3. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`
   
   ```bash
   curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```
   
4. Now, you should see an Argo workflow being created.
   
   ```bash
   kubectl -n argo-events get wf
   ```
   
## File & URL
File and URL trigger sources are pretty self explanatory. The example sensors are available under `examples/sensors` folder.
