package codefresh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
)

const (
	cfConfigMapName       = "codefresh-cm"
	cfBaseURLConfigMapKey = "base-url"
	cfSecretName          = "codefresh-token"
	cfAuthSecretKey       = "token"
)

type Config struct {
	BaseURL   string
	AuthToken string
}

type API struct {
	ctx           context.Context
	logger        *zap.SugaredLogger
	cfConfig      *Config
	client        *http.Client
	isInitialised bool
}

var withRetry = common.Connect // alias

func NewAPI(ctx context.Context, namespace string) (*API, error) {
	logger := logging.FromContext(ctx)
	config, err := getCodefreshConfig(ctx, namespace)
	if err != nil {
		return &API{logger: logger}, err
	}

	return &API{
		ctx:           ctx,
		logger:        logger,
		cfConfig:      config,
		isInitialised: true,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (a *API) shouldReportEvent(event cloudevents.Event) bool {
	whitelist := map[apicommon.EventSourceType]bool{
		apicommon.GithubEvent:          true,
		apicommon.GitlabEvent:          true,
		apicommon.BitbucketServerEvent: true,
		apicommon.CalendarEvent:        true,
	}

	return whitelist[apicommon.EventSourceType(event.Type())]
}

func (a *API) ReportEvent(event cloudevents.Event) {
	if !a.shouldReportEvent(event) {
		return
	}

	if !a.isInitialised {
		a.logger.Warnw("WARNING: skipping reporting of an event to Codefresh", zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
		return
	}

	eventJson, err := json.Marshal(event)
	if err != nil {
		a.logger.Errorw("failed to report an event to Codefresh", zap.Error(err), zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
		return
	}

	url := a.cfConfig.BaseURL + "/2.0/api/events/event-payload"
	err = a.sendJSON(eventJson, url)
	if err != nil {
		a.logger.Errorw("failed to report an event to Codefresh", zap.Error(err), zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
	} else {
		a.logger.Infow("succeeded to report an event to Codefresh", zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
	}
}

func (a *API) ReportError(originalError error) {
	originalErrorMsg := originalError.Error()
	if !a.isInitialised {
		a.logger.Warnw("WARNING: skipping reporting of an error to Codefresh", zap.String("originalError", originalErrorMsg))
		return
	}

	errorJson, err := json.Marshal(originalErrorMsg)
	if err != nil {
		a.logger.Errorw("failed to report an error to Codefresh", zap.Error(err), zap.String("originalError", originalErrorMsg))
		return
	}

	url := a.cfConfig.BaseURL + "/2.0/api/events/error"
	err = a.sendJSON(errorJson, url)
	if err != nil {
		a.logger.Errorw("failed to report an error to Codefresh", zap.Error(err), zap.String("originalError", originalErrorMsg))
	} else {
		a.logger.Infow("succeeded to report an error to Codefresh", zap.String("originalError", originalErrorMsg))
	}
}

func (a *API) sendJSON(jsonBody []byte, url string) error {
	return withRetry(&common.DefaultBackoff, func() error {
		req, err := http.NewRequestWithContext(a.ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", a.cfConfig.AuthToken)

		res, err := a.client.Do(req)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed reporting to Codefresh, event: %s", string(jsonBody)))
		}
		defer res.Body.Close()

		isStatusOK := res.StatusCode >= 200 && res.StatusCode < 300
		if !isStatusOK {
			b, _ := ioutil.ReadAll(res.Body)
			return errors.Errorf("failed reporting to Codefresh, got response: status code %d and body %s, original request body: %s",
				res.StatusCode, string(b), string(jsonBody))
		}

		return nil
	})
}

func getCodefreshConfig(ctx context.Context, namespace string) (*Config, error) {
	kubeClient, err := common.CreateKubeClient()
	if err != nil {
		return nil, err
	}
	baseURL, err := getCodefreshBaseURL(ctx, kubeClient, namespace)
	if err != nil {
		return nil, err
	}
	token, err := getCodefreshAuthToken(ctx, kubeClient, namespace)
	if err != nil {
		return nil, err
	}

	return &Config{
		BaseURL:   baseURL,
		AuthToken: token,
	}, nil
}

func getCodefreshAuthToken(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	cfSecretSelector := &corev1.SecretKeySelector{
		Key: cfAuthSecretKey,
		LocalObjectReference: corev1.LocalObjectReference{
			Name: cfSecretName,
		},
	}

	return common.GetSecretValue(ctx, kubeClient, namespace, cfSecretSelector)
}

func getCodefreshBaseURL(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	cfConfigMapSelector := &corev1.ConfigMapKeySelector{
		Key: cfBaseURLConfigMapKey,
		LocalObjectReference: corev1.LocalObjectReference{
			Name: cfConfigMapName,
		},
	}

	return common.GetConfigMapValue(ctx, kubeClient, namespace, cfConfigMapSelector)
}
