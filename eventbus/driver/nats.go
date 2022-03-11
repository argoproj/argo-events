package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NATStreaming struct {
	url       string
	auth      *Auth
	clusterID string

	logger *zap.SugaredLogger
}

// NewNATSStreaming returns a nats streaming driver
func NewNATSStreaming(url string, clusterID string, auth *Auth, logger *zap.SugaredLogger) *NATStreaming {
	return &NATStreaming{
		url:       url,
		clusterID: clusterID,
		auth:      auth,
		logger:    logger,
	}
}

func (n *NATStreaming) MakeConnection(clientID string) (*NATSStreamingConnection, error) {
	log := n.logger.With("clientID", clientID)
	conn := &NATSStreamingConnection{clientID: clientID, Logger: n.logger}
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
	conn.NATSConn = nc
	conn.NATSConnected = true

	sc, err := stan.Connect(n.clusterID, clientID, stan.NatsConn(nc), stan.Pings(5, 60),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			conn.STANConnected = false
			log.Errorw("NATS streaming connection lost", zap.Error(err))
		}))
	if err != nil {
		log.Errorw("Failed to connect to NATS streaming server", zap.Error(err))
		return nil, err
	}
	log.Info("Connected to NATS streaming server.")
	conn.STANConn = sc
	conn.STANConnected = true
	return conn, nil
}
