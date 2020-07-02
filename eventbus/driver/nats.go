package driver

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gobwas/glob"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

type natsStreamingConnection struct {
	natsConn *nats.Conn
	stanConn stan.Conn
}

func (nsc *natsStreamingConnection) Close() error {
	if nsc.stanConn != nil {
		err := nsc.stanConn.Close()
		if err != nil {
			return err
		}
	}
	if nsc.natsConn != nil && nsc.natsConn.IsConnected() {
		nsc.natsConn.Close()
	}
	return nil
}

func (nsc *natsStreamingConnection) IsClosed() bool {
	if nsc.natsConn == nil {
		return true
	}
	return nsc.natsConn.IsClosed()
}

func (nsc *natsStreamingConnection) Publish(subject string, data []byte) error {
	return nsc.stanConn.Publish(subject, data)
}

type natsStreaming struct {
	url       string
	auth      *Auth
	clusterID string
	subject   string
	clientID  string

	logger *logrus.Logger
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

func (n *natsStreaming) Connect() (Connection, error) {
	opts := []nats.Option{}
	switch n.auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		n.logger.Info("NATS auth strategy: Token")
		opts = append(opts, nats.Token(n.auth.Crendential.Token))
	case eventbusv1alpha1.AuthStrategyNone:
		n.logger.Info("NATS auth strategy: None")
	default:
		return nil, errors.New("unsupported auth strategy")
	}
	nc, err := nats.Connect(n.url, opts...)
	if err != nil {
		n.logger.Errorf("Failed to connect to NATS server, %v", err)
		return nil, err
	}
	n.logger.Info("Connected to NATS server.")
	sc, err := stan.Connect(n.clusterID, n.clientID, stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			n.logger.Fatalf("Connection lost, reason: %v", reason)
		}))
	if err != nil {
		n.logger.Errorf("Failed to connect to NATS streaming server, %v", err)
		return nil, err
	}
	n.logger.Info("Connected to NATS streaming server.")
	return &natsStreamingConnection{
		natsConn: nc,
		stanConn: sc,
	}, nil
}

func (n *natsStreaming) Publish(conn Connection, message []byte) error {
	return conn.Publish(n.subject, message)
}

// SubscribeCloudEvents is used to subscribe multiple dependency expression
// Parameter - dependencyExpr, example: "(dep1 || dep2) && dep3"
// Parameter - dependencies, array of dependencies information
// Parameter - filter, a function used to filter the message
// Parameter - action, a function to be triggered after all conditions meet
func (n *natsStreaming) SubscribeEventSources(conn Connection, dependencyExpr string, dependencies []Dependency, filter func(string, cloudevents.Event) bool, action func(map[string]cloudevents.Event)) error {
	msgHolder, err := newEventSourceMessageHolder(dependencyExpr, dependencies)
	if err != nil {
		return err
	}
	nsc, ok := conn.(*natsStreamingConnection)
	if !ok {
		return errors.New("not a NATS streaming connection")
	}
	// use clientID as durable name?
	durableName := n.clientID
	sub, err := nsc.stanConn.Subscribe(n.subject, func(m *stan.Msg) {
		n.processEventSourceMsg(m, msgHolder, filter, action)
	}, stan.DurableName(durableName),
		stan.SetManualAckMode(),
		stan.StartAt(pb.StartPosition_NewOnly),
		stan.AckWait(1*time.Second),
		stan.MaxInflight(len(msgHolder.depNames)+2))
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
		_ = m.Ack()
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

	depName, err := msgHolder.getDependencyName(event.Source(), event.Subject())
	if err != nil {
		n.logger.Errorf("Failed to get the dependency name, discarding it... err: %v", err)
		_ = m.Ack()
		return
	}

	// ACK all the old messages after conditions meet
	if m.Timestamp <= msgHolder.latestGoodMsgTimestamp {
		if depName != "" {
			msgHolder.reset(depName)
		}
		_ = m.Ack()
		return
	}

	// Start a new round
	if depName == "" || !filter(depName, *event) {
		// message not interested
		_ = m.Ack()
		return
	}
	if existingMsg, ok := msgHolder.msgs[depName]; ok {
		if m.Timestamp == existingMsg.timestamp {
			// Redelivered latest messge, return
			return
		} else if m.Timestamp < existingMsg.timestamp {
			// Redelivered old message, ack and return
			_ = m.Ack()
			return
		}
	}
	// New message, set and check
	msgHolder.msgs[depName] = &eventSourceMessage{seq: m.Sequence, timestamp: m.Timestamp, event: event}
	msgHolder.parameters[depName] = true

	// Check if there's any message older than 3 days, which is the default exiration time of event bus messages.
	now := time.Now().UnixNano()
	for k, v := range msgHolder.msgs {
		if (now - v.timestamp) > 3*24*60*60*1000000000 {
			msgHolder.reset(k)
			return
		}
	}

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
	_ = m.Ack()
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
	depNames               []string
	// Mapping of [eventSourceName + eventName]dependencyName
	sourceDepMap map[string]string
	parameters   map[string]interface{}
	msgs         map[string]*eventSourceMessage
}

func newEventSourceMessageHolder(dependencyExpr string, dependencies []Dependency) (*eventSourceMessageHolder, error) {
	dependencyExpr = strings.ReplaceAll(dependencyExpr, "-", "\\-")
	expression, err := govaluate.NewEvaluableExpression(dependencyExpr)
	if err != nil {
		return nil, err
	}
	deps := unique(expression.Vars())
	if len(dependencyExpr) == 0 {
		return nil, errors.Errorf("no dependencies found: %s", dependencyExpr)
	}

	srcDepMap := make(map[string]string)
	for _, d := range dependencies {
		key := d.EventSourceName + "__" + d.EventName
		srcDepMap[key] = d.Name
	}

	parameters := make(map[string]interface{}, len(deps))
	msgs := make(map[string]*eventSourceMessage)
	for _, dep := range deps {
		parameters[dep] = false
	}

	return &eventSourceMessageHolder{
		lastMeetTime:           int64(0),
		latestGoodMsgTimestamp: int64(0),
		expr:                   expression,
		depNames:               deps,
		sourceDepMap:           srcDepMap,
		parameters:             parameters,
		msgs:                   msgs,
	}, nil
}

func (mh *eventSourceMessageHolder) getDependencyName(eventSourceName, eventName string) (string, error) {
	for k, v := range mh.sourceDepMap {
		sourceGlob, err := glob.Compile(k)
		if err != nil {
			return "", err
		}
		if sourceGlob.Match(eventSourceName + "__" + eventName) {
			return v, nil
		}
	}
	return "", nil
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
