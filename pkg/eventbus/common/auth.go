package common

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// IsNATSCredsFile reports whether data is a NATS credentials file (JWT + NKey seed).
func IsNATSCredsFile(data []byte) bool {
	s := string(data)
	return strings.Contains(s, "BEGIN NATS USER JWT") && strings.Contains(s, "BEGIN USER NKEY SEED")
}

// LoadEventBusAuth loads auth from EVENTBUS_NATS_CREDENTIALS when set, otherwise
// from the mounted eventbus auth file. If the payload is a NATS credentials
// file, Strategy is AuthStrategyCredential regardless of the requested strategy.
func LoadEventBusAuth(mountPath string, strategy eventbusv1alpha1.AuthStrategy, logger *zap.SugaredLogger, onChange func()) (*Auth, error) {
	if strategy == "" || strategy == eventbusv1alpha1.AuthStrategyNone {
		return &Auth{Strategy: eventbusv1alpha1.AuthStrategyNone}, nil
	}

	if data := os.Getenv(eventbusv1alpha1.EnvVarEventBusNATSCredentials); data != "" {
		return authFromBytes([]byte(data), strategy, logger)
	}

	authFile := filepath.Join(mountPath, eventbusv1alpha1.EventBusAuthFileName)
	data, err := os.ReadFile(authFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s. err: %w", eventbusv1alpha1.EventBusAuthFileName, err)
	}

	auth, err := authFromBytes(data, strategy, logger)
	if err != nil {
		return nil, err
	}
	if auth.Strategy == eventbusv1alpha1.AuthStrategyCredential || onChange == nil {
		return auth, nil
	}

	v := sharedutil.ViperWithLogging()
	v.SetConfigName("auth")
	v.SetConfigType("yaml")
	v.AddConfigPath(mountPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to load %s. err: %w", eventbusv1alpha1.EventBusAuthFileName, err)
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		onChange()
	})
	return auth, nil
}

func authFromBytes(data []byte, strategy eventbusv1alpha1.AuthStrategy, logger *zap.SugaredLogger) (*Auth, error) {
	if IsNATSCredsFile(data) {
		return &Auth{
			Strategy: eventbusv1alpha1.AuthStrategyCredential,
			Credential: &AuthCredential{
				Credentials: append([]byte(nil), data...),
			},
		}, nil
	}

	v := sharedutil.ViperWithLogging()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("failed to parse eventbus auth. err: %w", err)
	}
	cred := &AuthCredential{}
	if err := v.Unmarshal(cred); err != nil {
		logger.Errorw("failed to unmarshal eventbus auth", zap.Error(err))
		return nil, err
	}
	return &Auth{
		Strategy:   strategy,
		Credential: cred,
	}, nil
}

// ApplyNATSAuthOptions appends nats.Option values for the given event bus auth.
func ApplyNATSAuthOptions(opts []nats.Option, auth *Auth) ([]nats.Option, error) {
	if auth == nil {
		return opts, nil
	}
	switch auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		opts = append(opts, nats.Token(auth.Credential.Token))
	case eventbusv1alpha1.AuthStrategyBasic:
		opts = append(opts, nats.UserInfo(auth.Credential.Username, auth.Credential.Password))
	case eventbusv1alpha1.AuthStrategyCredential:
		if auth.Credential == nil || len(auth.Credential.Credentials) == 0 {
			return nil, fmt.Errorf("credentials are required for credential auth")
		}
		opts = append(opts, nats.UserCredentialBytes(auth.Credential.Credentials))
	case eventbusv1alpha1.AuthStrategyNone, "":
		// no auth
	default:
		return nil, fmt.Errorf("unsupported auth strategy")
	}
	return opts, nil
}

// ApplyNATSAuthToOptions sets username/password/token/JWT on nats.Options.
func ApplyNATSAuthToOptions(opts *nats.Options, auth *Auth) error {
	if auth == nil {
		return nil
	}
	switch auth.Strategy {
	case eventbusv1alpha1.AuthStrategyToken:
		opts.Token = auth.Credential.Token
	case eventbusv1alpha1.AuthStrategyBasic:
		opts.User = auth.Credential.Username
		opts.Password = auth.Credential.Password
	case eventbusv1alpha1.AuthStrategyCredential:
		if auth.Credential == nil || len(auth.Credential.Credentials) == 0 {
			return fmt.Errorf("credentials are required for credential auth")
		}
		return nats.UserCredentialBytes(auth.Credential.Credentials)(opts)
	case eventbusv1alpha1.AuthStrategyNone, "":
		// no auth
	default:
		return fmt.Errorf("unsupported auth strategy")
	}
	return nil
}
