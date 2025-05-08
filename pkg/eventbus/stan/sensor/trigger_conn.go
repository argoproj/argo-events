package sensor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gobwas/glob"
	stan "github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"go.uber.org/zap"

	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	stanbase "github.com/argoproj/argo-events/pkg/eventbus/stan/base"
)

type STANTriggerConn struct {
	*stanbase.STANConnection

	sensorName           string
	triggerName          string
	dependencyExpression string
	deps                 []eventbuscommon.Dependency
}

func NewSTANTriggerConn(conn *stanbase.STANConnection, sensorName string, triggerName string, dependencyExpression string, deps []eventbuscommon.Dependency) *STANTriggerConn {
	n := &STANTriggerConn{conn, sensorName, triggerName, dependencyExpression, deps}
	n.Logger = n.Logger.With("triggerName", n.triggerName).With("clientID", n.ClientID)
	return n
}

func (n *STANTriggerConn) String() string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("STANTriggerConn{ClientID:%s,Sensor:%s,Trigger:%s}", n.ClientID, n.sensorName, n.triggerName)
}

func (conn *STANTriggerConn) IsClosed() bool {
	return conn == nil || conn.STANConnection.IsClosed()
}

func (conn *STANTriggerConn) Close() error {
	if conn == nil {
		return fmt.Errorf("can't close STAN trigger connection, STANTriggerConn is nil")
	}
	return conn.STANConnection.Close()
}

// Subscribe is used to subscribe to multiple event source dependencies
// Parameter - ctx, context
// Parameter - conn, eventbus connection
// Parameter - group, queue group name
// Parameter - closeCh, channel to indicate to close the subscription
// Parameter - resetConditionsCh, channel to indicate to reset trigger conditions
// Parameter - lastResetTime, the last time reset would have occurred, if any
// Parameter - dependencyExpr, example: "(dep1 || dep2) && dep3"
// Parameter - dependencies, array of dependencies information
// Parameter - filter, a function used to filter the message
// Parameter - action, a function to be triggered after all conditions meet
func (n *STANTriggerConn) Subscribe(
	ctx context.Context,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	defaultSubject *string) error {
	if n == nil {
		return fmt.Errorf("Subscribe() failed; STANTriggerConn is nil")
	}

	log := n.Logger

	if defaultSubject == nil {
		log.Error("can't subscribe over NATS streaming: defaultSubject not set")
	}

	msgHolder, err := newEventSourceMessageHolder(log, n.dependencyExpression, n.deps, lastResetTime)
	if err != nil {
		return err
	}
	// use group name as durable name
	group, err := n.getGroupNameFromClientID(n.ClientID)
	if err != nil {
		return err
	}
	durableName := group
	sub, err := n.STANConn.QueueSubscribe(*defaultSubject, group, func(m *stan.Msg) {
		n.processEventSourceMsg(m, msgHolder, transform, filter, action, log)
	}, stan.DurableName(durableName),
		stan.SetManualAckMode(),
		stan.StartAt(pb.StartPosition_NewOnly),
		stan.AckWait(1*time.Second),
		stan.MaxInflight(len(msgHolder.depNames)+2))
	if err != nil {
		log.Errorf("failed to subscribe to subject %s", *defaultSubject)
		return err
	}
	log.Infof("Subscribed to subject %s using durable name %s", *defaultSubject, durableName)

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
				msgHolder.setLastResetTime(time.Now())
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info("exiting, unsubscribing and closing connection...")
			_ = sub.Close()
			log.Infof("subscription on subject %s closed", *defaultSubject)
			daemonStopCh <- struct{}{}
			wg.Wait()
			return nil
		case <-closeCh:
			log.Info("closing subscription...")
			_ = sub.Close()
			log.Infof("subscription on subject %s closed", *defaultSubject)
			daemonStopCh <- struct{}{}
			wg.Wait()
			return nil
		}
	}
}

func (n *STANTriggerConn) processEventSourceMsg(m *stan.Msg, msgHolder *eventSourceMessageHolder, transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error), filter func(dependencyName string, event cloudevents.Event) bool, action func(map[string]cloudevents.Event), log *zap.SugaredLogger) {
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

	log.Debugf("New incoming Event Source Message, dependency name=%s", depName)

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
		log.Debugf("not interested in dependency %s", depName)
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

	// Acknowledge any old messages that occurred before the last reset (standard reset after trigger or conditional reset)
	if m.Timestamp <= msgHolder.getLastResetTime().UnixNano() {
		if depName != "" {
			msgHolder.reset(depName)
		}
		msgHolder.ackAndCache(m, event.ID())

		log.Debugf("reset and acked dependency=%s due to message time occurred before reset, m.Timestamp=%d, msgHolder.getLastResetTime()=%d",
			depName, m.Timestamp, msgHolder.getLastResetTime().UnixNano())
		return
	}
	// make sure that everything has been cleared within a certain amount of time
	if msgHolder.fullResetTimeout() {
		log.Infof("ATTENTION: Resetting the flags because they didn't get cleared before the timeout: msgHolder=%+v", msgHolder)
		msgHolder.resetAll()
	}

	now := time.Now().Unix()

	// Start a new round
	if existingMsg, ok := msgHolder.msgs[depName]; ok {
		if m.Timestamp == existingMsg.timestamp {
			// Re-delivered latest message, update delivery timestamp and return
			existingMsg.lastDeliveredTime = now
			msgHolder.msgs[depName] = existingMsg
			log.Debugf("Updating timestamp for dependency=%s", depName)
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
		log.Infow("trigger conditions not met", zap.Any("meetDependencies", meetDeps), zap.Any("meetEvents", meetMsgIds))
		return
	}

	msgHolder.setLastResetTime(time.Unix(m.Timestamp/1e9, m.Timestamp%1e9))
	// Trigger actions
	messages := make(map[string]cloudevents.Event)
	for k, v := range msgHolder.msgs {
		messages[k] = *v.event
	}
	log.Debugf("Triggering actions for client %s", n.ClientID)

	action(messages)

	msgHolder.reset(depName)
	msgHolder.ackAndCache(m, event.ID())
}

func (n *STANTriggerConn) getGroupNameFromClientID(clientID string) (string, error) {
	log := n.Logger.With("clientID", n.ClientID)
	// take off the last part: clientID should have a dash at the end and we can remove that part
	strs := strings.Split(clientID, "-")
	if len(strs) < 2 {
		err := fmt.Errorf("expected client ID to contain dash: %s", clientID)
		log.Error(err)
		return "", err
	}
	return strings.Join(strs[:len(strs)-1], "-"), nil
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
	lastResetTime time.Time
	// if we reach this time, we reset everything (occurs 60 seconds after lastResetTime)
	resetTimeout int64
	expr         *govaluate.EvaluableExpression
	depNames     []string
	// Mapping of [eventSourceName + eventName]dependencyName
	sourceDepMap map[string]string
	parameters   map[string]interface{}
	msgs         map[string]*eventSourceMessage
	// A sync map used to cache the message IDs, it is used to guarantee Exact Once triggering
	smap        *sync.Map
	lock        sync.RWMutex
	timeoutLock sync.RWMutex

	logger *zap.SugaredLogger
}

func newEventSourceMessageHolder(logger *zap.SugaredLogger, dependencyExpr string, dependencies []eventbuscommon.Dependency, lastResetTime time.Time) (*eventSourceMessageHolder, error) {
	dependencyExpr = strings.ReplaceAll(dependencyExpr, "-", "\\-")
	expression, err := govaluate.NewEvaluableExpression(dependencyExpr)
	if err != nil {
		return nil, err
	}
	deps := unique(expression.Vars())
	if len(dependencyExpr) == 0 {
		return nil, fmt.Errorf("no dependencies found: %s", dependencyExpr)
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
		lastResetTime: lastResetTime,
		expr:          expression,
		depNames:      deps,
		sourceDepMap:  srcDepMap,
		parameters:    parameters,
		msgs:          msgs,
		smap:          new(sync.Map),
		lock:          sync.RWMutex{},
		logger:        logger,
	}, nil
}

func (mh *eventSourceMessageHolder) getLastResetTime() time.Time {
	mh.lock.RLock()
	defer mh.lock.RUnlock()
	return mh.lastResetTime
}

func (mh *eventSourceMessageHolder) setLastResetTime(t time.Time) {
	{
		mh.lock.Lock() // since this can be called asynchronously as part of a ConditionReset, we need to lock this code
		defer mh.lock.Unlock()
		mh.lastResetTime = t
	}
	mh.setResetTimeout(t.Add(time.Second * 60).Unix()) // failsafe condition: determine if we for some reason we haven't acknowledged all dependencies within 60 seconds of the lastResetTime
}

func (mh *eventSourceMessageHolder) setResetTimeout(t int64) {
	mh.timeoutLock.Lock() // since this can be called asynchronously as part of a ConditionReset, we need to lock this code
	defer mh.timeoutLock.Unlock()
	mh.resetTimeout = t
}

func (mh *eventSourceMessageHolder) getResetTimeout() int64 {
	mh.timeoutLock.RLock()
	defer mh.timeoutLock.RUnlock()
	return mh.resetTimeout
}

// failsafe condition after lastResetTime
func (mh *eventSourceMessageHolder) fullResetTimeout() bool {
	resetTimeout := mh.getResetTimeout()
	return resetTimeout != 0 && time.Now().Unix() > resetTimeout
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
		mh.setResetTimeout(0)
	}
}

func (mh *eventSourceMessageHolder) resetAll() {
	for k := range mh.msgs {
		delete(mh.msgs, k)
	}

	for k := range mh.parameters {
		mh.parameters[k] = false
	}
	mh.setResetTimeout(0)
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
