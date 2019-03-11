<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/file.png?raw=true" alt="File"/>
</p>

<br/>

# File

File gateway watches changes to the file within specified directory.

## Where the directory should be?
The directory can be in the pod's own filesystem or you can mount a persistent volume and refer to a directory.
Make sure that the directory exists before you create the gateway configmap.

## How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Directory to watch for events
Directory string `json:"directory"`

// Path is relative path of object to watch with respect to the directory
Path string `json:"path,omitempty"`

// PathRegexp is regexp of relative path of object to watch with respect to the directory
// +Optional
PathRegexp string `json:"pathRegexp,omitempty"`

// Type of file operations to watch
// Refer https://github.com/fsnotify/fsnotify/blob/master/fsnotify.go for more information
Type string `json:"type"`
```

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: file-gateway-configmap
data:
  bindir: |- # event source name can be any valid string
    directory: "/bin/"  # directory where file events are watched
    type: CREATE # type of file event
    path: x.txt # file to watch to
```

## Setup

**1. Install Gateway**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/file.yaml
```

Make sure the gateway pod is created.

**2. Install Gateway Configmap**

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/file-gateway-configmap.yaml
```

**3. Install Sensor**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/file.yaml
```

Make sure the sensor pod is created.

**4. Trigger Workflow**

Exec into the gateway pod and go to the directory specified in event source and create a file. That should generate an event causing sensor to trigger a workflow.


## How to listen to notifications from different directories
Simply edit the gateway configmap and add new entry that contains the configuration required to listen to file within different directory and save
the configmap. The gateway will start listening to file notifications from new directory as well.
