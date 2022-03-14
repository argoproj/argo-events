package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Jetstream struct {
	url  string
	auth *Auth
	// clusterID string
	//jetstreamContext nats.JetStreamContext
	mgmtConnection JetstreamConnection

	streamSettings string

	logger *zap.SugaredLogger
}

func NewJetstream(url string, auth *Auth, logger *zap.SugaredLogger) (*Jetstream, error) {

	// todo: need to pass streamSettings into this function
	streamSettings := ""
	js := &Jetstream{
		url:            url,
		auth:           auth,
		logger:         logger,
		streamSettings: streamSettings,
	}

	mgmtConnection, err := js.MakeConnection("mgmt")
	js.mgmtConnection = *mgmtConnection

	return js, err
}

func (stream *Jetstream) MakeConnection(clientID string) (*JetstreamConnection, error) {
	log := stream.logger.With("clientID", clientID)
	conn := &JetstreamConnection{clientID: clientID, Logger: stream.logger}
	// todo: duplicate below - reduce?
	opts := []nats.Option{
		// Do not reconnect here but handle reconnction outside
		nats.NoReconnect(),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			conn.NATSConnected = false
			log.Errorw("NATS connection lost", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nnc *nats.Conn) {
			conn.NATSConnected = true
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
	conn.NATSConn = nc
	conn.NATSConnected = true

	// Create JetStream Context
	conn.JSContext, err = nc.JetStream()
	if err != nil {
		log.Errorw("Failed to get Jetstream context", zap.Error(err))
		return nil, err
	}

	err = stream.CreateStream(conn)
	if err != nil {
		log.Errorw("Failed to create Stream", zap.Error(err))
		return nil, err
	}

	log.Info("Connected to NATS Jetstream server.")
	return conn, nil
}

func (stream *Jetstream) CreateStream(conn *JetstreamConnection) error {
	if conn == nil {
		return errors.New("Can't create Stream on nil connection")
	}
	var err error

	options := make([]nats.JSOpt, 0)
	// todo: create a JSOpt for each setting that the user specifies

	streamName := "default"
	_, err = conn.JSContext.AddStream(&nats.StreamConfig{
		Name: streamName, // todo: replace with a const
		//Subjects: []string{"default.*"},
		Subjects: []string{"default.*.*"},
	}, options...)
	if err != nil {
		return errors.Errorf("Failed to add Jetstream stream '%s': %v for connection %+v", streamName, err, conn)
	}

	stream.logger.Infof("Created Jetstream stream '%s' for connection %+v", streamName, conn)
	return nil
}
