package driver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gobwas/glob"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type natsStreamingConnection struct {
	natsConn *nats.Conn
	stanConn stan.Conn

	natsConnected bool
	stanConnected bool
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
	if nsc.natsConn == nil || nsc.stanConn == nil || !nsc.natsConnected || !nsc.stanConnected || nsc.natsConn.IsClosed() {
		return true
	}
	return false
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

	logger *zap.SugaredLogger
}

// NewNATSStreaming returns a nats streaming driver
func NewNATSStreaming(url, clusterID, subject, clientID string, auth *Auth, logger *zap.SugaredLogger) Driver {
	return &natsStreaming{
		url:       url,
		clusterID: clusterID,
		subject:   subject,
		clientID:  clientID,
		auth:      auth,
		logger:    logger,
	}
}

func (n *natsStreaming) Connect() (Connection, error) {
	log := n.logger.With("clientID", n.clientID)
	conn := &natsStreamingConnection{}
	opts := []nats.Option{
		// Do not reconnect here but handle reconnction outside
		nats.NoReconnect(),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			conn.natsConnected = false
			log.Errorw("NATS connection lost", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nnc *nats.Conn) {
			conn.natsConnected = true
			log.Info("Reconnected to NATS server")
		}),
	}
	switch n.auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		log.Info("NATS auth strategy: Token")
		opts = append(opts, nats.Token(n.auth.Crendential.Token))
	case eventbusv1alpha1.AuthStrategyNone:
		log.Info("NATS auth strategy: None")
	default:
		return nil, errors.New("unsupported auth strategy")
	}
	nc, err := nats.Connect(n.url, opts...)
	if err != nil {
		log.Errorw("Failed to connect to NATS server", zap.Error(err))
		return nil, err
	}
	log.Info("Connected to NATS server.")
	conn.natsConn = nc
	conn.natsConnected = true

	sc, err := stan.Connect(n.clusterID, n.clientID, stan.NatsConn(nc), stan.Pings(5, 60),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			conn.stanConnected = false
			log.Errorw("NATS streaming connection lost", zap.Error(err))
		}))
	if err != nil {
		log.Errorw("Failed to connect to NATS streaming server", zap.Error(err))
		return nil, err
	}
	log.Info("Connected to NATS streaming server.")
	conn.stanConn = sc
	conn.stanConnected = true
	return conn, nil
}

func (n *natsStreaming) Publish(conn Connection, message []byte) error {
	return conn.Publish(n.subject, message)
}

// SubscribeEventSources is used to subscribe multiple event source dependencies
// Parameter - ctx, context
// Parameter - conn, eventbus connection
// Parameter - group, queue group name
// Parameter - closeCh, channel to indicate to close the subscription
// Parameter - resetConditionsCh, channel to indicate to reset trigger conditions
// Parameter - dependencyExpr, example: "(dep1 || dep2) && dep3"
// Parameter - dependencies, array of dependencies information
// Parameter - filter, a function used to filter the message
// Parameter - action, a function to be triggered after all conditions meet
func (n *natsStreaming) SubscribeEventSources(ctx context.Context, conn Connection, group string, closeCh <-chan struct{}, resetConditionsCh <-chan struct{}, dependencyExpr string, dependencies []Dependency, transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error), filter func(string, cloudevents.Event) bool, action func(map[string]cloudevents.Event)) error {
	log := n.logger.With("clientID", n.clientID)
	msgHolder, err := newEventSourceMessageHolder(dependencyExpr, dependencies)
	if err != nil {
		return err
	}
	nsc, ok := conn.(*natsStreamingConnection)
	if !ok {
		return errors.New("not a NATS streaming connection")
	}
	// use group name as durable name
	durableName := group
	sub, err := nsc.stanConn.QueueSubscribe(n.subject, group, func(m *stan.Msg) {
		n.processEventSourceMsg(m, msgHolder, transform, filter, action, log)
	}, stan.DurableName(durableName),
		stan.SetManualAckMode(),
		stan.StartAt(pb.StartPosition_NewOnly),
		stan.AckWait(1*time.Second),
		stan.MaxInflight(len(msgHolder.depNames)+2))
	if err != nil {
		log.Errorf("failed to subscribe to subject %s", n.subject)
		return err
	}
	log.Infof("Subscribed to subject %s ...", n.subject)

	// Daemon to evict cache and reset trigger conditions
	wg := &sync.WaitGroup{}
	daemonStopCh := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("starting ExactOnce cache clean up daemon ...")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-daemonStopCh:
				log.Info("exiting ExactOnce cache clean up daemon...")
				return
			case <-ticker.C:
				now := time.Now().UnixNano()
				num := 0
				msgHolder.smap.Range(func(key, value interface{}) bool {
					v := value.(int64)
					// Evict cached ID older than 5 minutes
					if now-v > 5*60*1000*1000*1000 {
						msgHolder.smap.Delete(key)
						num++
						log.Debugw("cached ID evicted", "id", key)
					}
					return true
				})
				log.Debugf("finished evicting %v cached IDs, time cost: %v ms", num, (time.Now().UnixNano()-now)/1000/1000)
			case <-resetConditionsCh:
				log.Info("reset conditions")
				msgHolder.setLastResetTime(time.Now().Unix())
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info("existing, unsubscribing and closing connection...")
			_ = sub.Close()
			log.Infof("subscription on subject %s closed", n.subject)
			daemonStopCh <- struct{}{}
			wg.Wait()
			return nil
		case <-closeCh:
			log.Info("closing subscription...")
			_ = sub.Close()
			log.Infof("subscription on subject %s closed", n.subject)
			daemonStopCh <- struct{}{}
			wg.Wait()
			return nil
		}
	}
}

func (n *natsStreaming) processEventSourceMsg(m *stan.Msg, msgHolder *eventSourceMessageHolder, transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error), filter func(dependencyName string, event cloudevents.Event) bool, action func(map[string]cloudevents.Event), log *zap.SugaredLogger) {
	var event *cloudevents.Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		log.Errorf("Failed to convert to a cloudevent, discarding it... err: %v", err)
		_ = m.Ack()
		return
	}

	depName, err := msgHolder.getDependencyName(event.Source(), event.Subject())
	if err != nil {
		log.Errorf("Failed to get the dependency name, discarding it... err: %v", err)
		_ = m.Ack()
		return
	}

	if depName == "" {
		_ = m.Ack()
		return
	}

	event, err = transform(depName, *event)
	if err != nil {
		log.Errorw("failed to apply event transformation", zap.Error(err))
		_ = m.Ack()
		return
	}

	if !filter(depName, *event) {
		// message not interested
		_ = m.Ack()
		return
	}

	// NATS Streaming guarantees At Least Once delivery,
	// so need to check if the message is duplicate
	if _, ok := msgHolder.smap.Load(event.ID()); ok {
		log.Infow("ATTENTION: Duplicate delivered message detected", "message", m)
		_ = m.Ack()
		return
	}

	// Clean up old messages before starting a new round
	if msgHolder.getLastResetTime() > 0 {
		// ACK all the old messages after conditions meet
		if m.Timestamp <= msgHolder.latestGoodMsgTimestamp {
			if depName != "" {
				msgHolder.reset(depName)
			}
			msgHolder.ackAndCache(m, event.ID())
			return
		}

		// Old redelivered messages should be able to be acked in 60 seconds.
		// Reset if the flag didn't get cleared in that period for some reasons.
		if time.Now().Unix()-msgHolder.getLastResetTime() > 60 {
			msgHolder.resetAll()
			log.Info("ATTENTION: Reset the flags because they didn't get cleared in 60 seconds...")
		}
		return
	}

	now := time.Now().Unix()

	// Start a new round
	if existingMsg, ok := msgHolder.msgs[depName]; ok {
		if m.Timestamp == existingMsg.timestamp {
			// Re-delivered latest messge, update delivery timestamp and return
			existingMsg.lastDeliveredTime = now
			msgHolder.msgs[depName] = existingMsg
			return
		} else if m.Timestamp < existingMsg.timestamp {
			// Re-delivered old message, ack and return
			msgHolder.ackAndCache(m, event.ID())
			log.Debugw("Dropping this message because later ones also satisfy", "eventID", event.ID())
			return
		}
	}
	// New message, set and check
	msgHolder.msgs[depName] = &eventSourceMessage{seq: m.Sequence, timestamp: m.Timestamp, event: event, lastDeliveredTime: now}
	msgHolder.parameters[depName] = true
	if msgHolder.latestGoodMsgTimestamp < m.Timestamp {
		msgHolder.latestGoodMsgTimestamp = m.Timestamp
	}

	// Check if there's any stale message being held.
	// Stale message could be message age has been longer than NATS streaming max message age,
	// which means it has ben deleted from NATS server side, but it's still held here.
	// Use last delivery timestamp to determine that.
	for k, v := range msgHolder.msgs {
		// Since the message is not acked, the server will keep re-sending it.
		// If a message being held didn't get re-delivered in the last 10 minutes, treat it as stale.
		if (now - v.lastDeliveredTime) > 10*60 {
			msgHolder.reset(k)
		}
	}

	result, err := msgHolder.expr.Evaluate(msgHolder.parameters)
	if err != nil {
		log.Errorf("failed to evaluate dependency expression: %v", err)
		// TODO: how to handle this situation?
		return
	}
	if result != true {
		// Log current meet dependency information
		meetDeps := []string{}
		meetMsgIds := []string{}
		for k, v := range msgHolder.msgs {
			meetDeps = append(meetDeps, k)
			meetMsgIds = append(meetMsgIds, v.event.ID())
		}
		log.Infow("trigger conditions do not meet", zap.Any("meetDependencies", meetDeps), zap.Any("meetEvents", meetMsgIds))
		return
	}
	msgHolder.setLastResetTime(now)
	// Trigger actions
	messages := make(map[string]cloudevents.Event)
	for k, v := range msgHolder.msgs {
		messages[k] = *v.event
	}
	log.Debugf("Triggering actions for client %s", n.clientID)

	action(messages)

	msgHolder.reset(depName)
	msgHolder.ackAndCache(m, event.ID())
}

// eventSourceMessage is used by messageHolder to hold the latest message
type eventSourceMessage struct {
	seq       uint64
	timestamp int64
	event     *cloudevents.Event
	// timestamp of last delivered
	lastDeliveredTime int64
}

// eventSourceMessageHolder is a struct used to hold the message information of subscribed dependencies
type eventSourceMessageHolder struct {
	// time that resets conditions, usually the time all conditions meet,
	// or the time getting an external signal to reset.
	lastResetTime int64
	// timestamp of last msg
	latestGoodMsgTimestamp int64
	expr                   *govaluate.EvaluableExpression
	depNames               []string
	// Mapping of [eventSourceName + eventName]dependencyName
	sourceDepMap map[string]string
	parameters   map[string]interface{}
	msgs         map[string]*eventSourceMessage
	// A sync map used to cache the message IDs, it is used to guarantee Exact Once triggering
	smap *sync.Map
	lock sync.RWMutex
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
		lastResetTime:          int64(0),
		latestGoodMsgTimestamp: int64(0),
		expr:                   expression,
		depNames:               deps,
		sourceDepMap:           srcDepMap,
		parameters:             parameters,
		msgs:                   msgs,
		smap:                   new(sync.Map),
		lock:                   sync.RWMutex{},
	}, nil
}

func (mh *eventSourceMessageHolder) getLastResetTime() int64 {
	mh.lock.RLock()
	defer mh.lock.RUnlock()
	return mh.lastResetTime
}

func (mh *eventSourceMessageHolder) setLastResetTime(t int64) {
	mh.lock.Lock()
	defer mh.lock.Unlock()
	mh.lastResetTime = t
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

// Ack the stan message and cache the ID to make sure Exact Once triggering
func (mh *eventSourceMessageHolder) ackAndCache(m *stan.Msg, id string) {
	_ = m.Ack()
	mh.smap.Store(id, time.Now().UnixNano())
}

// Reset the parameter and message that a dependency holds
func (mh *eventSourceMessageHolder) reset(depName string) {
	mh.parameters[depName] = false
	delete(mh.msgs, depName)
	if mh.isCleanedUp() {
		mh.setLastResetTime(0)
	}
}

func (mh *eventSourceMessageHolder) resetAll() {
	for k := range mh.msgs {
		delete(mh.msgs, k)
	}
	for k := range mh.parameters {
		mh.parameters[k] = false
	}
	mh.setLastResetTime(0)
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
