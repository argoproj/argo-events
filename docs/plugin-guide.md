# Plugging in Signals
This is a walkthrough for how to plugin different signals. I am leveraging [hashicorp's go-plugin](https://github.com/hashicorp/go-plugin) in implementing the signal interface and RPC.

## 1. Follow the [quickstart](quickstart.md) to get Minikube up & running

## 2. Build a Go binary plugin. See the `job/nats` implementation for an example for how to do this.

## 3. Modify the `job/Dockerfile` to copy the plugin binary and set the `SIGNAL_PLUGIN` env var to reference the binary name.

## 4. Build the job Dockerfile
```
$ make executor-job-image
```

## 5. Create a sensor
```
$ k create -f examples/nats-sensor.yaml
```

## 6. Trigger the signal execution
This depends on your signal implementation. For `NATS`, you can use `github.com/shogsbro/natscat` you just have to expose the NATS service externally as a `LoadBalancer`. 
```
$ go get github.com/shogsbro/natscat
$ cd $GOPATH/src/github.shogsbro/natscat
$ ./natscat -S http://192.168.99.100:32472 -s bucketevents "test"
```
