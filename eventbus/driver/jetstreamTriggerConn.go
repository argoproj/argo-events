package driver


import (

	nats "github.com/nats-io/nats.go"
)


type JetstreamTriggerConn struct {
	sensorName string
	triggerName string
	keyValueStore *nats.KeyValue
}

func (conn *JetstreamTriggerConn) Subscribe(ctx context.Context,
	conn Connection,
	group string,
	//sensorName string,
	//triggerName string,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	dependencyExpr string,
	dependencies []Dependency,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event)) error
{
		log := stream.logger //.With("clientID", stream.clientID)
	
		// Create a Consumer
		_, err := stream.jetstreamContext.AddConsumer("default", &nats.ConsumerConfig{
			Durable: group,
		})
		if err != nil {
			// tbd
		}
	
		// derive subjects that we'll subscribe with using the dependencies passed in
		subjects := make(map[string]struct{}) // essentially a set
		for _, dep := range dependencies {
			subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = struct{}{}
		}
	
		stream.SetupKeyValueStore()
	
		err = stream.CleanUpOnStart(group, sensorName, triggerName, dependencies)
	
	
	
		return nil
}
	
	
func (conn *JetstreamTriggerConn) CleanUpOnStart(group string, dependencies []Dependency) error {
	// first look in K/V store for old Triggers that no longer exist

	// for each 

}
	