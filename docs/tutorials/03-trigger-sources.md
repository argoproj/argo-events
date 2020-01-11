# Trigger Sources
A trigger source is the source of trigger resource. It can be either external source such
as `Git`, `S3`, `K8s Configmap`, `File`, any valid `URL` that hosts the resource or an internal resource
which is defined in the sensor object itself like `Inline` or `Resource`. 

In the previous sections, you have been dealing with the `Resource` trigger source. In this tutorial, we
will explore other trigger sources.

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
