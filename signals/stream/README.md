# Stream Signal Plugins
In today's ecosystem with the rise in popularity of real-time systems and microservices, there exists a plethora of message brokers and streaming platforms from which to choose from. This results in the problem of being able to support a wide variety of platforms (external and internal) depending on the technology stack. The current solution to this problem uses pluggable binaries that implement the `Signaler` interface, run as a separate process to the main `sensor-controller`, and communicate over `RPC` or `gRPC`. 

## Use Cases
1. User wishes to listen to events from outside the currently default builtin stream functionality without having to change the code or add to the existing `Stream` configuration.
2. User wishes to leverage an existing platform from their stack as a source to trigger workflows.

## Design
- `Signaler` interface is defined within the `shared` package. All signals must implement this interface.
```
// Signaler is the interface for signaling
type Signaler interface {
	Start(*v1alpha1.Signal) (<-chan *v1alpha1.Event, error)
	Stop() error
}
``` 
- Use [go-plugin](https://github.com/hashicorp/go-plugin) to register/create all signals. 
- Plugin must have a `main()` function and use `go-plugin` to serve up the plugin structure/functionality via RPC or gRPC. (One plugin per binary!)
- Binaries MUST be included in the sensor-controller's `Dockerfile` as:
```
COPY dist/nats-plugin /
ENV SIGNAL_PLUGIN=/nats-plugin
```

## Build your own
Building your own stream signal plugin is easy. All you need is a `struct` which implements the `Signaler` interface. You can write your plugin in a number of different languages, however using a language other than `Go` requires a bit more work and is not covered in this. 

If you're using Go, take a look at the `builtin` package for examples on writing your own custom plugin. You'll need to call the `github.com/hashicorp/go-plugin`.`Serve()` method in the main function of your program. You'll also need to append an entry to the `PluginMap` in the `shared` directory. Please, put the new plugin under the `custom` package. You'll then need to compile the program and output it as: `dist/stream-plugin`.

## Future work
It may be desirable to use 2 or more streaming services as signals. In the current implementation, only one stream plugin binary is included in the controller's `Dockerfile`. We can include the ability to add a directory for all the plugin binary files. These files can then be included in the Docker build or injected dynamically into the program through a `PVC`. 