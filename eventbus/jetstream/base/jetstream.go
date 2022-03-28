package base

import (
	"bytes"
	"fmt"

	"github.com/argoproj/argo-events/common"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Jetstream struct {
	url  string
	auth *eventbuscommon.Auth
	// clusterID string
	// jetstreamContext nats.JetStreamContext
	MgmtConnection JetstreamConnection

	streamSettings string

	Logger *zap.SugaredLogger
}

func NewJetstream(url string, streamSettings string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*Jetstream, error) {

	js := &Jetstream{
		url:            url,
		auth:           auth,
		Logger:         logger,
		streamSettings: streamSettings,
	}

	return js, nil
}

func (stream *Jetstream) Init() error {
	mgmtConnection, err := stream.MakeConnection()
	if err != nil {
		errStr := fmt.Sprintf("error creating Management Connection for Jetstream stream %+v: %v", stream, err)
		stream.Logger.Error(errStr)
		return errors.New(errStr)
	}
	stream.MgmtConnection = *mgmtConnection
	return nil
}

func (stream *Jetstream) MakeConnection() (*JetstreamConnection, error) {
	log := stream.Logger
	conn := &JetstreamConnection{Logger: stream.Logger}
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

	// unmarshal settings
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(stream.streamSettings)); err != nil {
		return err
	}

	streamConfig := nats.StreamConfig{
		Name:       common.JetStreamStreamName,
		Subjects:   []string{common.JetStreamStreamName + ".*.*"},
		Retention:  nats.LimitsPolicy,
		Discard:    nats.DiscardOld,
		MaxMsgs:    v.GetInt64("maxMsgs"),
		MaxAge:     v.GetDuration("maxAge"),
		MaxBytes:   v.GetInt64("maxBytes"),
		Storage:    nats.FileStorage,
		Replicas:   v.GetInt("replicas"),
		Duplicates: v.GetDuration("duplicates"),
	}
	stream.Logger.Infof("Will use this stream config:\n '%v'", streamConfig)
	options := make([]nats.JSOpt, 0)

	_, err = conn.JSContext.AddStream(&streamConfig, options...)
	if err != nil {
		return errors.Errorf("Failed to add Jetstream stream '%s': %v for connection %+v", common.JetStreamStreamName, err, conn)
	}

	stream.Logger.Infof("Created Jetstream stream '%s' for connection %+v", common.JetStreamStreamName, conn)
	return nil
}
