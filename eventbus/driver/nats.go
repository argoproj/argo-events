package driver

import (
	"encoding/json"
	"os"
	"os/signal"
	"time"

	"github.com/Knetic/govaluate"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

type natsStreaming struct {
	url       string
	auth      *Auth
	clusterID string
	subject   string
	clientID  string

	natsConn *nats.Conn
	stanConn stan.Conn
	logger   *logrus.Logger
}

// NewNATSStreaming returns a nats streaming driver
func NewNATSStreaming(url, clusterID, subject, clientID string, auth *Auth, logger *logrus.Logger) Driver {
	return &natsStreaming{
		url:       url,
		clusterID: clusterID,
		subject:   subject,
		clientID:  clientID,
		auth:      auth,
		logger:    logger.WithField("clusterID", clusterID).WithField("clientID", clientID).Logger}
}

func (n *natsStreaming) Connect() error {
	opts := []nats.Option{}
	switch n.auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		opts = append(opts, nats.Token(n.auth.Crendential.Token))
	case eventbusv1alpha1.AuthStrategyNone:
	default:
		return errors.New("unsupported auth strategy")
	}
	nc, err := nats.Connect(n.url, opts...)
	if err != nil {
		n.logger.Errorf("failed to connect to NATS server, +%v", err)
		return err
	}
	n.natsConn = nc
	sc, err := stan.Connect(n.clusterID, n.clientID, stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			n.logger.Fatalf("Connection lost, reason: %v", reason)
		}))
	if err != nil {
		n.logger.Errorf("failed to connect to NATS streaming server, +%v", err)
	}
	n.stanConn = sc
	return nil
}

func (n *natsStreaming) Disconnect() error {
	if n.stanConn != nil {
		err := n.stanConn.Close()
		if err != nil {
			return err
		}
	}
	if n.natsConn != nil && n.natsConn.IsConnected() {
		n.natsConn.Close()
	}
	return nil
}

func (n *natsStreaming) Reconnect() error {
	if err := n.Disconnect(); err != nil {
		return err
	}
	return n.Connect()
}

func (n *natsStreaming) Publish(message []byte) error {
	return n.stanConn.Publish(n.subject, message)
}

// SubscribeCloudEvents is used to subscribe multiple dependency expression
// Parameter - dependencyExpr, example: "(dep1 || dep2) && dep3"
// Parameter - depencencies, array of dependencies information
// Parameter - filter, a function used to filter the message
// Parameter - action, a function to be triggered after all conditions meet
func (n *natsStreaming) SubscribeEventSources(dependencyExpr string, depencencies []Dependency, filter func(string, cloudevents.Event) bool, action func(map[string]cloudevents.Event)) error {
	msgHolder, err := newEventSourceMessageHolder(dependencyExpr, depencencies)
	if err != nil {
		return err
	}
	// use clientID as durable name?
	durableName := n.clientID
	sub, err := n.stanConn.Subscribe(n.subject, func(m *stan.Msg) {
		n.processEventSourceMsg(m, msgHolder, filter, action)
	}, stan.DurableName(durableName),
		stan.SetManualAckMode(),
		stan.StartAt(pb.StartPosition_NewOnly),
		stan.AckWait(1*time.Second),
		stan.MaxInflight(len(msgHolder.dependencies)+2))
	if err != nil {
		n.logger.Errorf("failed to subscribe to subject %s", n.subject)
		return err
	}
	n.logger.Infof("Subscribed to subject %s ...", n.subject)

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			n.logger.Info("Received an interrupt, unsubscribing and closing connection...")
			_ = sub.Close()
			n.logger.Infof("subscription on subject %s closed", n.subject)
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	return nil
}

func (n *natsStreaming) processEventSourceMsg(m *stan.Msg, msgHolder *eventSourceMessageHolder, filter func(dependencyName string, event cloudevents.Event) bool, action func(map[string]cloudevents.Event)) {
	var event *cloudevents.Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		n.logger.Errorf("Failed to convert to a cloudevent, discarding it... err: %v", err)
		m.Ack()
		return
	}

	// Old redelivered messages should be able to be acked in 60 seconds.
	// Reset if the flag didn't get cleared in that period for some reasons.
	if msgHolder.lastMeetTime > 0 || msgHolder.latestGoodMsgTimestamp > 0 {
		if time.Now().Unix()-msgHolder.lastMeetTime > 60 {
			msgHolder.resetAll()
			n.logger.Info("ATTENTION: Reset the flags because they didn't get cleared in 60 seconds...")
		}
	}

	// ACK all the old messages after conditions meet
	depName := msgHolder.getDependencyName(event.Source(), event.Subject())
	if m.Timestamp <= msgHolder.latestGoodMsgTimestamp {
		if depName != "" {
			msgHolder.reset(depName)
		}
		m.Ack()
		return
	}

	// Start a new round
	if depName == "" || !filter(depName, *event) {
		// message not interested
		m.Ack()
		return
	}
	if existingMsg, ok := msgHolder.msgs[depName]; ok {
		if m.Sequence == existingMsg.seq {
			// Redelivered latest messge, return
			return
		} else if m.Sequence < existingMsg.seq {
			// Redelivered old message, ack and return
			m.Ack()
			return
		}
	}
	// New message, set and check
	msgHolder.msgs[depName] = &eventSourceMessage{seq: m.Sequence, timestamp: m.Timestamp, event: event}
	msgHolder.parameters[depName] = true

	result, err := msgHolder.expr.Evaluate(msgHolder.parameters)
	if err != nil {
		n.logger.Errorf("failed to evaluate dependency expression: %v", err)
		// TODO: how to handle this situation?
		return
	}
	if result != true {
		return
	}
	msgHolder.latestGoodMsgTimestamp = m.Timestamp
	msgHolder.lastMeetTime = time.Now().Unix()
	// Trigger actions
	messages := make(map[string]cloudevents.Event)
	for k, v := range msgHolder.msgs {
		messages[k] = *v.event
	}
	n.logger.Infof("Triggering actions for client %s", n.clientID)

	go action(messages)

	msgHolder.reset(depName)
	m.Ack()
	return
}

// eventSourceMessage is used by messageHolder to hold the latest message
type eventSourceMessage struct {
	seq       uint64
	timestamp int64
	event     *cloudevents.Event
}

// eventSourceMessageHolder is a struct used to hold the message information of subscribed dependencies
type eventSourceMessageHolder struct {
	// time that all conditions meet
	lastMeetTime int64
	// timestamp of last msg when all the conditions meet
	latestGoodMsgTimestamp int64
	expr                   *govaluate.EvaluableExpression
	dependencies           []string
	// Mapping of [eventSourceName + eventName]dependencyName
	sourceDepMap map[string]string
	parameters   map[string]interface{}
	msgs         map[string]*eventSourceMessage
}

func newEventSourceMessageHolder(dependencyExpr string, depencencies []Dependency) (*eventSourceMessageHolder, error) {
	expression, err := govaluate.NewEvaluableExpression(dependencyExpr)
	if err != nil {
		return nil, err
	}
	deps := unique(expression.Vars())
	if len(dependencyExpr) == 0 {
		return nil, errors.Errorf("no dependencies found: %s", dependencyExpr)
	}

	srcDepMap := make(map[string]string)
	depSrcMap := make(map[string]Dependency)
	for _, d := range depencencies {
		key := d.EventSourceName + "__" + d.EventName
		srcDepMap[key] = d.Name
		depSrcMap[d.Name] = d
	}

	parameters := make(map[string]interface{}, len(deps))
	msgs := make(map[string]*eventSourceMessage)
	for _, dep := range deps {
		// Validate if there's an invalid dependency name
		if _, ok := depSrcMap[dep]; !ok {
			return nil, errors.Errorf("Dependency expression and dependency list do not match, %s is not found", dep)
		}
		parameters[dep] = false
	}

	return &eventSourceMessageHolder{
		lastMeetTime:           int64(0),
		latestGoodMsgTimestamp: int64(0),
		expr:                   expression,
		dependencies:           deps,
		sourceDepMap:           srcDepMap,
		parameters:             parameters,
		msgs:                   msgs,
	}, nil
}

func (mh *eventSourceMessageHolder) getDependencyName(eventSourceName, eventName string) string {
	if depName, ok := mh.sourceDepMap[eventSourceName+"__"+eventName]; ok {
		return depName
	}
	return ""
}

// Reset the parameter and message that a dependency holds
func (mh *eventSourceMessageHolder) reset(depName string) {
	mh.parameters[depName] = false
	delete(mh.msgs, depName)
	if mh.isCleanedUp() {
		mh.lastMeetTime = 0
		mh.latestGoodMsgTimestamp = 0
	}
}

func (mh *eventSourceMessageHolder) resetAll() {
	for k := range mh.msgs {
		delete(mh.msgs, k)
	}
	for k := range mh.parameters {
		mh.parameters[k] = false
	}
	mh.lastMeetTime = 0
	mh.latestGoodMsgTimestamp = 0
}

// Check if all the parameters and messages have been cleaned up
func (mh *eventSourceMessageHolder) isCleanedUp() bool {
	for _, v := range mh.parameters {
		if v == true {
			return false
		}
	}
	return len(mh.msgs) == 0
}

func unique(stringSlice []string) []string {
	if len(stringSlice) == 0 {
		return stringSlice
	}
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
