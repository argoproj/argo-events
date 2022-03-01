package driver

import (
	"context"
	"time"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type jetstreamConnection struct {
	natsConn  *nats.Conn
	jsContext nats.JetStreamContext

	natsConnected bool
}

func (jsc *jetstreamConnection) Close() error {

	if jsc.natsConn != nil && jsc.natsConn.IsConnected() {
		jsc.natsConn.Close()
	}
	return nil
}

func (jsc *jetstreamConnection) IsClosed() bool {
	if jsc.natsConn == nil || !jsc.natsConnected || jsc.natsConn.IsClosed() {
		return true
	}
	return false
}

func (jsc *jetstreamConnection) Publish(subject string, data []byte) error {
	// todo: On the publishing side you can avoid duplicate message ingestion using the Message Deduplication feature.
	_, err := jsc.jsContext.Publish(subject, data)
	return err
}

type jetstream struct {
	url  string
	auth *Auth
	//clusterID string
	//subject   string  todo: decide if we want some default subject for publishers
	//clientID string   todo: seems like jetstream doesn't have this notion; if we want to identify it for ourselves we can
	//durableName string // todo: not sure if we want this here; may not be necessary to store it and it also doesn't apply to publishers
	jetstreamContext nats.JetStreamContext

	logger *zap.SugaredLogger
}

func NewJetstream(url, auth *Auth, logger *zap.SugaredLogger) Driver {
	return &jetstream{
		url:    url,
		auth:   auth,
		logger: logger,
	}
}

func (stream *jetstream) Connect() (Connection, error) {
	log := stream.logger.With("clientID", stream.clientID)
	conn := &jetstreamConnection{}
	// todo: duplicate below - reduce?
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
	switch stream.auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		log.Info("NATS auth strategy: Token")
		opts = append(opts, nats.Token(stream.auth.Crendential.Token))
	case eventbusv1alpha1.AuthStrategyNone:
		log.Info("NATS auth strategy: None")
	default:
		return nil, errors.New("unsupported auth strategy")
	}
	nc, err := nats.Connect(stream.url, opts...)
	if err != nil {
		log.Errorw("Failed to connect to NATS server", zap.Error(err))
		return nil, err
	}
	log.Info("Connected to NATS server.")
	conn.natsConn = nc
	conn.natsConnected = true

	// Create JetStream Context
	stream.jetstreamContext, err = nc.JetStream()
	if err != nil {
		// tbd
	}
	conn.jsContext = stream.jetstreamContext

	// add the Stream just in case nobody has yet
	_, err = stream.jetstreamContext.AddStream(&nats.StreamConfig{
		Name:     "default", // todo: replace with a const
		Subjects: []string{"default.*"},
	})
	if err != nil {
		// tbd
	}

	// todo: when we subscribe later we can specify durable name there (look at SubOpt in js.go)
	// maybe also use AckAll()? need to look for all SubOpt

	log.Info("Connected to NATS streaming server.")
	return conn, nil
}

// todo: determine if we want the Publisher to be able to start up its connection with a default
// subject
func (stream *jetstream) Publish(conn Connection, subject string, message []byte) error {
	return conn.Publish(subject, message)
}

func (stream *jetstream) SubscribeEventSources(ctx context.Context, conn Connection, subject string, group string, closeCh <-chan struct{}, resetConditionsCh <-chan struct{}, lastResetTime time.Time, dependencyExpr string, dependencies []Dependency, transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error), filter func(string, cloudevents.Event) bool, action func(map[string]cloudevents.Event)) error {
	log := stream.logger.With("clientID", stream.clientID)
	//stream.durableName = group

	// Create a Consumer
	_, err := stream.jetstreamContext.AddConsumer("default", &nats.ConsumerConfig{
		Durable: group,
	})
	if err != nil {
		// tbd
	}
}
