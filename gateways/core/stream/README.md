# Signal Microservices
In today's ecosystem with the rise in popularity of real-time systems and microservices, there exists a plethora of message brokers and streaming platforms from which to choose from. This results in the problem of being able to support a wide variety of platforms (external and internal) depending on the technology stack. The current solution to this problem uses [micro](https://github.com/micro/go-micro) microservices that implement the `Signaler` interface, run as a separate deployment (multiple pods + service) to the `sensor-controller`, and communicate over `gRPC`.

## Use Cases
1. User wishes to listen to events from outside the currently default builtin stream functionality without having to change the code or add to the existing `Stream` configuration.
2. User wishes to leverage an existing platform from their stack as a source to trigger workflows.

## Design
- `Signaler` interface is defined within the `sdk` package. All signals must implement this interface.
```
// Signaler is the interface for signaling
type Signaler interface {
	Start(*v1alpha1.Signal) (<-chan *v1alpha1.Event, error)
	Stop() error
}
``` 
- Use [micro k8s](https://github.com/micro/kubernetes) to register deployed microservices. Each pod is considered a node of the microservice.
- `Sensor controller` registers itself using the `micro k8s` as a client to the signal microservices. It then is able to 

## Build your own
Building your own signal microservice is easy. All you need is a `struct` which implements the `Signaler` interface. You can write your service in a number of different languages, however using a language other than `Go` requires a bit more work and is not covered in this. 

If you're using Go, take a look at the `builtin` package for examples on writing your own custom services. You'll need to call the `sdk.RegisterSignalServiceHandler()` method in the main function of your program. Please, put the new plugin under the `custom` package.
