package leaderelection

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/nats-io/graft"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

type Elector interface {
	RunOrDie(context.Context, LeaderCallbacks)
}

type LeaderCallbacks struct {
	OnStartedLeading func(context.Context)
	OnStoppedLeading func()
}

func NewEventBusElector(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, clusterName string, clusterSize int) (Elector, error) {
	logger := logging.FromContext(ctx)
	var eventBusType apicommon.EventBusType
	var eventBusAuth *eventbusv1alpha1.AuthStrategy
	if eventBusConfig.NATS != nil {
		eventBusType = apicommon.EventBusNATS
		eventBusAuth = eventBusConfig.NATS.Auth
	} else {
		return nil, errors.New("invalid event bus")
	}
	var auth *eventbusdriver.Auth
	cred := &eventbusdriver.AuthCredential{}
	if eventBusAuth == nil || *eventBusAuth == eventbusv1alpha1.AuthStrategyNone {
		auth = &eventbusdriver.Auth{
			Strategy: eventbusv1alpha1.AuthStrategyNone,
		}
	} else {
		v := viper.New()
		v.SetConfigName("auth")
		v.SetConfigType("yaml")
		v.AddConfigPath(common.EventBusAuthFileMountPath)
		err := v.ReadInConfig()
		if err != nil {
			return nil, errors.Errorf("failed to load auth.yaml. err: %+v", err)
		}
		err = v.Unmarshal(cred)
		if err != nil {
			logger.Errorw("failed to unmarshal auth.yaml", zap.Error(err))
			return nil, err
		}
		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			logger.Info("eventbus auth config file changed.")
			err = v.Unmarshal(cred)
			if err != nil {
				logger.Errorw("failed to unmarshal auth.yaml after reloading", zap.Error(err))
			}
		})
		auth = &eventbusdriver.Auth{
			Strategy:    *eventBusAuth,
			Crendential: cred,
		}
	}
	var elector Elector
	switch eventBusType {
	case apicommon.EventBusNATS:
		elector = &natsEventBusElector{
			clusterName: clusterName,
			size:        clusterSize,
			url:         eventBusConfig.NATS.URL,
			auth:        auth,
		}
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return elector, nil
}

type natsEventBusElector struct {
	clusterName string
	size        int
	url         string
	auth        *eventbusdriver.Auth
}

func (e *natsEventBusElector) RunOrDie(ctx context.Context, callbacks LeaderCallbacks) {
	log := logging.FromContext(ctx)
	ci := graft.ClusterInfo{Name: e.clusterName, Size: e.size}
	opts := &nats.DefaultOptions
	opts.Url = e.url
	if e.auth.Strategy == eventbusv1alpha1.AuthStrategyToken {
		opts.Token = e.auth.Crendential.Token
	}
	rpc, err := graft.NewNatsRpc(opts)
	if err != nil {
		log.Fatalw("failed to new Nats Rpc", zap.Error(err))
	}
	errChan := make(chan error)
	stateChangeChan := make(chan graft.StateChange)
	handler := graft.NewChanHandler(stateChangeChan, errChan)
	node, err := graft.New(ci, handler, rpc, "/tmp/graft.log")
	if err != nil {
		log.Fatalw("failed to new a node", zap.Error(err))
	}
	defer node.Close()

	cctx, cancel := context.WithCancel(ctx)
	if node.State() == graft.LEADER {
		log.Info("I'm the LEADER, starting ...")
		go callbacks.OnStartedLeading(cctx)
	} else {
		log.Info("Not the LEADER, stand by ...")
	}

	handleStateChange := func(sc graft.StateChange) {
		switch sc.To {
		case graft.LEADER:
			log.Info("I'm the LEADER, starting ...")
			go callbacks.OnStartedLeading(cctx)
		case graft.FOLLOWER, graft.CANDIDATE:
			log.Infof("Becoming a %v, stand by ...", sc.To)
			if sc.From == graft.LEADER {
				cancel()
				callbacks.OnStoppedLeading()
				cctx, cancel = context.WithCancel(ctx)
			}
		case graft.CLOSED:
			if sc.From == graft.LEADER {
				cancel()
				callbacks.OnStoppedLeading()
			}
			log.Fatal("Leader elector connection was CLOSED")
		default:
			log.Fatalf("Unknown state: %s", sc.To)
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("exiting...")
			cancel()
			return
		case sc := <-stateChangeChan:
			handleStateChange(sc)
		case err := <-errChan:
			log.Errorw("Error happened", zap.Error(err))
		}
	}
}
