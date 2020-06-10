package driver

import (
	"os"
	"os/signal"
	"time"

	"github.com/Knetic/govaluate"
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
	clientID  string

	natsConn *nats.Conn
	stanConn stan.Conn
	logger   *logrus.Logger
}

// NewNATSStreaming returns a nats streaming driver
func NewNATSStreaming(url, clusterID, clientID string, auth *Auth, logger *logrus.Logger) Driver {
	return &natsStreaming{url: url, clusterID: clusterID, clientID: clientID, auth: auth, logger: logger}
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

func (n *natsStreaming) Publish(subject string, message []byte) error {
	return n.stanConn.Publish(subject, message)
}

func (n *natsStreaming) Subscribe(subject string, action func([]byte)) error {
	filterFunc := func(string, []byte) bool { return true }
	actionFunc := func(msgs map[string][]byte) { action(msgs[subject]) }
	return n.SubscribeSubjects(subject, filterFunc, actionFunc)
}

// SubscribeSubjects is used to subscribe multiple subjects with dependency expression
// Parameter - subjectExpr, example: "(subject1 || subject2) && subject3"
// Parameter - filter, a function used to filter the message
// Parameter - action, a function to be triggered after all conditions meet
func (n *natsStreaming) SubscribeSubjects(subjectExpr string, filter func(string, []byte) bool, action func(map[string][]byte)) error {
	msgHolder, err := newMessageHolder(subjectExpr)
	if err != nil {
		return err
	}
	subscriptionMap := make(map[string]stan.Subscription)
	for _, subject := range msgHolder.subjects {
		// use clientID as durable name?
		durableName := n.clientID
		sub, err := n.stanConn.Subscribe(subject, func(m *stan.Msg) {
			n.processMsg(m, msgHolder, filter, action)
		}, stan.DurableName(durableName),
			stan.SetManualAckMode(),
			stan.StartAt(pb.StartPosition_NewOnly),
			stan.AckWait(1*time.Second),
			stan.MaxInflight(2))
		if err != nil {
			n.logger.Errorf("failed to subscribe to subject %s", subject)
			return err
		}
		subscriptionMap[subject] = sub
		n.logger.Infof("Subscribed to subject %s ...", subject)
	}

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			n.logger.Info("Received an interrupt, unsubscribing and closing connection...")
			for k, v := range subscriptionMap {
				_ = v.Close()
				n.logger.Infof("subscription on subject %s closed", k)
			}
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	return nil
}

func (n *natsStreaming) processMsg(m *stan.Msg, msgHolder *messageHolder, filter func(subject string, message []byte) bool, action func(map[string][]byte)) error {
	// ACK all the old messages after conditions meet
	if m.Timestamp <= msgHolder.lastMeetTime {
		msgHolder.reset(m.Subject)
		m.Ack()
		return nil
	}

	// Start a new round
	if !filter(m.Subject, m.Data) {
		// message not interested
		m.Ack()
		return nil
	}
	if existingMsg, ok := msgHolder.msgs[m.Subject]; ok {
		if m.Sequence == existingMsg.seq {
			// Redelivered latest messge, return
			return nil
		} else if m.Sequence < existingMsg.seq {
			// Redelivered old message, ack and return
			m.Ack()
			return nil
		}
	}
	// New message, set and check
	msgHolder.msgs[m.Subject] = &message{seq: m.Sequence, timestamp: m.Timestamp, data: m.Data}
	msgHolder.parameters[m.Subject] = true

	result, err := msgHolder.expr.Evaluate(msgHolder.parameters)
	if err != nil {
		n.logger.Errorf("failed to evaluate subject expression: %v", err)
		return err
	}
	if result != true {
		return nil
	}
	msgHolder.lastMeetTime = m.Timestamp
	// Trigger actions
	messages := make(map[string][]byte)
	for k, v := range msgHolder.msgs {
		messages[k] = v.data
	}
	n.logger.Infof("Triggering actions for client %s", n.clientID)

	go action(messages)

	msgHolder.reset(m.Subject)
	m.Ack()
	return nil
}

// message is used by messageHolder to hold the latest message
type message struct {
	seq       uint64
	timestamp int64
	data      []byte
}

// messageHolder is a struct used to hold the message information of subscribed subjects
type messageHolder struct {
	lastMeetTime int64
	expr         *govaluate.EvaluableExpression
	subjects     []string
	parameters   map[string]interface{}
	msgs         map[string]*message
}

func newMessageHolder(subjectExpr string) (*messageHolder, error) {
	expression, err := govaluate.NewEvaluableExpression(subjectExpr)
	if err != nil {
		return nil, err
	}
	subjects := unique(expression.Vars())
	if len(subjects) == 0 {
		return nil, errors.Errorf("no subjects found: %s", subjectExpr)
	}
	parameters := make(map[string]interface{}, len(subjects))
	msgs := make(map[string]*message)
	for _, subject := range subjects {
		parameters[subject] = false
	}
	return &messageHolder{
		lastMeetTime: int64(0),
		expr:         expression,
		subjects:     subjects,
		parameters:   parameters,
		msgs:         msgs,
	}, nil
}

// Reset the parameter and message that a subject holds
func (mh *messageHolder) reset(subject string) {
	mh.parameters[subject] = false
	delete(mh.msgs, subject)
	if mh.isCleanedUp() {
		mh.lastMeetTime = 0
	}
}

// Check if all the parameters and messages have been cleaned up
func (mh *messageHolder) isCleanedUp() bool {
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
