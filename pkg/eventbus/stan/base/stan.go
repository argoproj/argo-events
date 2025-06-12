package base

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"go.uber.org/zap"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
)

type STAN struct {
	url       string
	auth      *eventbuscommon.Auth
	clusterID string

	logger *zap.SugaredLogger
}

// NewSTAN returns a nats streaming driver
func NewSTAN(url string, clusterID string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) *STAN {
	return &STAN{
		url:       url,
		clusterID: clusterID,
		auth:      auth,
		logger:    logger,
	}
}

func (n *STAN) MakeConnection(clientID string) (*STANConnection, error) {
	log := n.logger.With("clientID", clientID)
	conn := &STANConnection{ClientID: clientID, Logger: n.logger}
	opts := []nats.Option{
		// Do not reconnect here but handle reconnection outside
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
		opts = append(opts, nats.Token(n.auth.Credential.Token))
	case eventbusv1alpha1.AuthStrategyNone:
		log.Info("NATS auth strategy: None")
	default:
		return nil, fmt.Errorf("unsupported auth strategy")
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
