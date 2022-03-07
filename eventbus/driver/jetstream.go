package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Jetstream struct {
	url      string
	clientID string // seems like jetstream doesn't have this notion; we can just have this to uniquely identify ourselves in the log
	auth     *Auth
	//clusterID string
	//durableName string // todo: not sure if we want this here; may not be necessary to store it and it also doesn't apply to publishers
	jetstreamContext nats.JetStreamContext

	logger *zap.SugaredLogger
}

func NewJetstream(url string, clientID string, auth *Auth, logger *zap.SugaredLogger) Driver {
	return &Jetstream{
		url:      url,
		clientID: clientID,
		auth:     auth,
		logger:   logger,
	}
}

func (stream *Jetstream) MakeConnection(clientID string) (Connection, error) {
	log := stream.logger //.With("clientID", stream.clientID)
	conn := &JetstreamConnection{clientID: clientID}
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
	log.Info("Connected to NATS Jetstream server.")
	conn.natsConn = nc
	conn.natsConnected = true

	// Create JetStream Context
	stream.jetstreamContext, err = nc.JetStream()
	if err != nil {
		log.Errorw("Failed to get Jetstream context", zap.Error(err))
		return nil, err
	}
	conn.jsContext = stream.jetstreamContext

	// add the Stream just in case nobody has yet
	_, err = stream.jetstreamContext.AddStream(&nats.StreamConfig{
		Name:     "default", // todo: replace with a const
		Subjects: []string{"default.*"},
	})
	if err != nil {
		log.Errorw("Failed to add Jetstream stream 'default'", zap.Error(err))
		return nil, err
	}

	log.Info("Connected to NATS streaming server.")
	return conn, nil
}

/*
func (stream *Jetstream) Publish(conn Connection, message []byte, event Event) error {
	return conn.Publish(fmt.Sprintf("default.%s.%s", event.EventSourceName, event.EventName), message)
}
*/
