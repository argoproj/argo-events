package leaderelection

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nats-io/graft"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

var (
	eventBusAuthFileMountPath = aev1.EventBusAuthFileMountPath
)

type Elector interface {
	RunOrDie(context.Context, LeaderCallbacks)
}

type LeaderCallbacks struct {
	OnStartedLeading func(context.Context)
	OnStoppedLeading func()
}

func NewElector(ctx context.Context, eventBusConfig aev1.BusConfig, clusterName string, clusterSize int, namespace string, leasename string, hostname string) (Elector, error) {
	switch {
	case eventBusConfig.Kafka != nil || strings.ToLower(os.Getenv(aev1.EnvVarLeaderElection)) == "k8s":
		return newKubernetesElector(namespace, leasename, hostname)
	case eventBusConfig.NATS != nil:
		return newEventBusElector(ctx, eventBusConfig.NATS.Auth, clusterName, clusterSize, eventBusConfig.NATS.URL, nil)
	case eventBusConfig.JetStream != nil:
		if eventBusConfig.JetStream.AccessSecret != nil {
			return newEventBusElector(ctx, &aev1.AuthStrategyBasic, clusterName, clusterSize, eventBusConfig.JetStream.URL, eventBusConfig.JetStream.TLS)
		} else {
			return newEventBusElector(ctx, &aev1.AuthStrategyNone, clusterName, clusterSize, eventBusConfig.JetStream.URL, eventBusConfig.JetStream.TLS)
		}
	default:
		return nil, fmt.Errorf("invalid event bus")
	}
}

func newEventBusElector(ctx context.Context, authStrategy *aev1.AuthStrategy, clusterName string, clusterSize int, url string, tls *aev1.TLSConfig) (Elector, error) {
	auth, err := getEventBusAuth(ctx, authStrategy)
	if err != nil {
		return nil, err
	}

	return &natsEventBusElector{
		clusterName: clusterName,
		size:        clusterSize,
		url:         url,
		auth:        auth,
		tls:         tls,
	}, nil
}

func getEventBusAuth(ctx context.Context, authStrategy *aev1.AuthStrategy) (*eventbuscommon.Auth, error) {
	logger := logging.FromContext(ctx)

	var auth *eventbuscommon.Auth

	if authStrategy == nil || *authStrategy == aev1.AuthStrategyNone {
		auth = &eventbuscommon.Auth{
			Strategy: aev1.AuthStrategyNone,
		}
	} else {
		v := sharedutil.ViperWithLogging()
		v.SetConfigName("auth")
		v.SetConfigType("yaml")
		v.AddConfigPath(eventBusAuthFileMountPath)

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to load auth.yaml. err: %w", err)
		}

		cred := &eventbuscommon.AuthCredential{}
		if err := v.Unmarshal(cred); err != nil {
			logger.Errorw("failed to unmarshal auth.yaml", zap.Error(err))
			return nil, err
		}

		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			// Auth file changed, let it restart.
			logger.Fatal("Eventbus auth config file changed, exiting..")
		})

		auth = &eventbuscommon.Auth{
			Strategy:   *authStrategy,
			Credential: cred,
		}
	}

	return auth, nil
}

type natsEventBusElector struct {
	clusterName string
	size        int
	url         string
	auth        *eventbuscommon.Auth
	tls         *aev1.TLSConfig
}

func (e *natsEventBusElector) RunOrDie(ctx context.Context, callbacks LeaderCallbacks) {
	log := logging.FromContext(ctx)
	ci := graft.ClusterInfo{Name: e.clusterName, Size: e.size}
	opts := nats.GetDefaultOptions()
	// Will never give up
	opts.MaxReconnect = -1
	opts.Url = e.url
	switch e.auth.Strategy {
	case aev1.AuthStrategyToken:
		opts.Token = e.auth.Credential.Token
	case aev1.AuthStrategyBasic:
		opts.User = e.auth.Credential.Username
		opts.Password = e.auth.Credential.Password
	}

	if e.tls != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(e.tls)
		if err != nil {
			log.Fatalw("failed to load tls configuration", zap.Error(err))
		}
		opts.TLSConfig = tlsConfig
	} else {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rpc, err := graft.NewNatsRpc(&opts)
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
	defer cancel()

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
			return
		case sc := <-stateChangeChan:
			handleStateChange(sc)
		case err := <-errChan:
			log.Errorw("Error happened", zap.Error(err))
		}
	}
}

type kubernetesElector struct {
	namespace string
	leasename string
	hostname  string
}

func newKubernetesElector(namespace string, leasename string, hostname string) (Elector, error) {
	return &kubernetesElector{
		namespace: namespace,
		leasename: leasename,
		hostname:  hostname,
	}, nil
}

func (e *kubernetesElector) RunOrDie(ctx context.Context, callbacks LeaderCallbacks) {
	logger := logging.FromContext(ctx)

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalw("Failed to retrieve kubernetes config", zap.Error(err))
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatalw("Failed to create kubernetes client", zap.Error(err))
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      e.leasename,
			Namespace: e.namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: e.hostname,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			ctx, cancel := context.WithCancel(ctx)
			leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
				Lock:            lock,
				ReleaseOnCancel: true,
				LeaseDuration:   5 * time.Second,
				RenewDeadline:   2 * time.Second,
				RetryPeriod:     1 * time.Second,
				Callbacks: leaderelection.LeaderCallbacks{
					OnStartedLeading: callbacks.OnStartedLeading,
					OnStoppedLeading: callbacks.OnStoppedLeading,
				},
			})

			// When the leader is lost, leaderelection.RunOrDie will
			// cease blocking and we will cancel the context. This
			// will halt all eventsource/sensor go routines.
			cancel()
		}
	}
}
