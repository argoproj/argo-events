package codefresh

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
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

func GetCodefreshConfig(ctx context.Context, namespace string) (*Config, error) {
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

func ReportEventToCodefresh(eventJson []byte, config *Config) error {
	contentType := "application/json"
	url := config.BaseURL + "/2.0/api/events/event-payload"
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(eventJson))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", config.AuthToken)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed reporting to Codefresh, event: %s", string(eventJson)))
	}
	defer res.Body.Close()

	isStatusOK := res.StatusCode >= 200 && res.StatusCode < 300
	if !isStatusOK {
		b, _ := ioutil.ReadAll(res.Body)
		return errors.Errorf("failed reporting to Codefresh, got response: status code %d and body %s, event: %s",
			res.StatusCode, string(b), string(eventJson))
	}

	return nil
}
