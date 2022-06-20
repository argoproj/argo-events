package codefresh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var withRetry = common.Connect // alias

type config struct {
	baseURL   string
	authToken string
}

type API struct {
	ctx           context.Context
	logger        *zap.SugaredLogger
	cfConfig      *config
	client        *http.Client
	isInitialised bool
}

type ErrorContext struct {
	metav1.ObjectMeta
	metav1.TypeMeta
}

type object struct {
	Group     string            `json:"group"`
	Version   string            `json:"version"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type errorContext struct {
	Object object `json:"object"`
}

type errorPayload struct {
	ErrMsg  string       `json:"errMsg"`
	Context errorContext `json:"context"`
}

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

	url := a.cfConfig.baseURL + "/2.0/api/events/event-payload"
	err = a.sendJSON(eventJson, url)
	if err != nil {
		a.logger.Errorw("failed to report an event to Codefresh", zap.Error(err), zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
	} else {
		a.logger.Infow("succeeded to report an event to Codefresh", zap.String(logging.LabelEventName, event.Subject()),
			zap.String(logging.LabelEventSourceType, event.Type()), zap.String("eventID", event.ID()))
	}
}

func constructErrorPayload(errMsg string, errContext ErrorContext) errorPayload {
	gvk := errContext.GroupVersionKind()

	return errorPayload{
		ErrMsg: errMsg,
		Context: errorContext{
			Object: object{
				Name:      errContext.Name,
				Namespace: errContext.Namespace,
				Group:     gvk.Group,
				Version:   gvk.Version,
				Kind:      gvk.Kind,
				Labels:    errContext.Labels,
			},
		},
	}
}

func (a *API) ReportError(originalErr error, errContext ErrorContext) {
	originalErrMsg := originalErr.Error()
	if !a.isInitialised {
		a.logger.Warnw("WARNING: skipping reporting of an error to Codefresh", zap.String("originalError", originalErrMsg))
		return
	}

	errPayloadJson, err := json.Marshal(constructErrorPayload(originalErrMsg, errContext))
	if err != nil {
		a.logger.Errorw("failed to report an error to Codefresh", zap.Error(err), zap.String("originalError", originalErrMsg))
		return
	}

	url := a.cfConfig.baseURL + "/2.0/api/events/error"
	err = a.sendJSON(errPayloadJson, url)
	if err != nil {
		a.logger.Errorw("failed to report an error to Codefresh", zap.Error(err), zap.String("originalError", originalErrMsg))
	} else {
		a.logger.Infow("succeeded to report an error to Codefresh", zap.String("originalError", originalErrMsg))
	}
}

func (a *API) sendJSON(jsonBody []byte, url string) error {
	return withRetry(&common.DefaultBackoff, func() error {
		req, err := http.NewRequestWithContext(a.ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", a.cfConfig.authToken)

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

func getCodefreshConfig(ctx context.Context, namespace string) (*config, error) {
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

	return &config{
		baseURL:   baseURL,
		authToken: token,
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
