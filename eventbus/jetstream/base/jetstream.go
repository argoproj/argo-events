package base

import (
	"bytes"
	"crypto/tls"
	"fmt"

	"github.com/argoproj/argo-events/common"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Jetstream struct {
	url  string
	auth *eventbuscommon.Auth

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
		return fmt.Errorf(errStr)
	}
	err = stream.CreateStream(mgmtConnection)
	if err != nil {
		stream.Logger.Errorw("Failed to create Stream", zap.Error(err))
		return err
	}
	stream.MgmtConnection = *mgmtConnection

	return nil
}

func (stream *Jetstream) MakeConnection() (*JetstreamConnection, error) {
	log := stream.Logger
	conn := &JetstreamConnection{Logger: stream.Logger}

	opts := []nats.Option{
		// todo: try out Jetstream's auto-reconnection capability
		nats.NoReconnect(),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			conn.NATSConnected = false
			log.Errorw("NATS connection lost", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nnc *nats.Conn) {
			conn.NATSConnected = true
			log.Info("Reconnected to NATS server")
		}),
		nats.Secure(&tls.Config{
			InsecureSkipVerify: true,
		}),
	}

	switch stream.auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		log.Info("NATS auth strategy: Token")
		opts = append(opts, nats.Token(stream.auth.Credential.Token))
	case eventbusv1alpha1.AuthStrategyBasic:
		log.Info("NATS auth strategy: Basic")
		opts = append(opts, nats.UserInfo(stream.auth.Credential.Username, stream.auth.Credential.Password))
	case eventbusv1alpha1.AuthStrategyNone:
		log.Info("NATS auth strategy: None")
	default:
		return nil, fmt.Errorf("unsupported auth strategy")
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

	log.Info("Connected to NATS Jetstream server.")
	return conn, nil
}

func (stream *Jetstream) CreateStream(conn *JetstreamConnection) error {
	if conn == nil {
		return fmt.Errorf("Can't create Stream on nil connection")
	}
	var err error

	// before we add the Stream first let's check to make sure it doesn't already exist
	streamInfo, err := conn.JSContext.StreamInfo(common.JetStreamStreamName)
	if streamInfo != nil && err == nil {
		stream.Logger.Infof("No need to create Stream '%s' as it already exists", common.JetStreamStreamName)
		return nil
	}
	if err != nil && err != nats.ErrStreamNotFound {
		stream.Logger.Warnf(`Error calling StreamInfo for Stream '%s' (this can happen if another Jetstream client "
		is trying to create the Stream at the same time): %v`, common.JetStreamStreamName, err)
	}

	// unmarshal settings
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(stream.streamSettings)); err != nil {
		return err
	}

	v.SetDefault("retention", 0) // Limits
	v.SetDefault("discard", 0)   // DiscardOld

	retentionPolicy, err := intToRetentionPolicy(v.GetInt("retention"))
	if err != nil {
		stream.Logger.Errorf("invalid retention policy: %s, error: %v", retentionPolicy, err)
		return err
	}

	discardPolicy, err := intToDiscardPolicy(v.GetInt("discard"))
	if err != nil {
		stream.Logger.Errorf("invalid discard policy: %s, error: %v", discardPolicy, err)
		return err
	}
	streamConfig := nats.StreamConfig{
		Name:       common.JetStreamStreamName,
		Subjects:   []string{common.JetStreamStreamName + ".*.*"},
		Retention:  retentionPolicy,
		Discard:    discardPolicy,
		MaxMsgs:    v.GetInt64("maxMsgs"),
		MaxAge:     v.GetDuration("maxAge"),
		MaxBytes:   v.GetInt64("maxBytes"),
		Storage:    nats.FileStorage,
		Replicas:   v.GetInt("replicas"),
		Duplicates: v.GetDuration("duplicates"),
	}
	stream.Logger.Infof("Will use this stream config:\n '%v'", streamConfig)

	connectErr := common.DoWithRetry(nil, func() error { // exponential backoff if it fails the first time
		_, err = conn.JSContext.AddStream(&streamConfig)
		if err != nil {
			errStr := fmt.Sprintf(`Failed to add Jetstream stream '%s'for connection %+v: err=%v`,
				common.JetStreamStreamName, conn, err)
			return fmt.Errorf(errStr)
		} else {
			return nil
		}
	})
	if connectErr != nil {
		return connectErr
	}

	stream.Logger.Infof("Created Jetstream stream '%s' for connection %+v", common.JetStreamStreamName, conn)
	return nil
}

func intToRetentionPolicy(i int) (nats.RetentionPolicy, error) {
	if i < 0 || i > int(nats.WorkQueuePolicy) {
		// Handle invalid value, return a default value or panic
		return -1, fmt.Errorf("invalid int for RetentionPolicy: %d", i)
	}
	return nats.RetentionPolicy(i), nil
}

func intToDiscardPolicy(i int) (nats.DiscardPolicy, error) {
	if i < 0 || i > int(nats.DiscardNew) {
		return -1, fmt.Errorf("invalid int for DiscardPolicy: %d", i)
	}
	return nats.DiscardPolicy(i), nil
}
