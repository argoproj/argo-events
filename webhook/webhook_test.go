package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alphal1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alphal1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alphal1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func fakeOptions() Options {
	return Options{
		Namespace:       "test-ns",
		DeploymentName:  "events-webhook",
		ClusterRoleName: "argo-events-webhook",
		ServiceName:     "webhook",
		Port:            8443,
		SecretName:      "webhook-certs",
		WebhookName:     "webhook.argo-events.argoproj.io",
	}
}

func fakeValidatingWebhookConfig(opts Options) *admissionregistrationv1.ValidatingWebhookConfiguration {
	sideEffects := admissionregistrationv1.SideEffectClassNone
	failurePolicy := admissionregistrationv1.Ignore
	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.WebhookName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:          opts.WebhookName,
				Rules:         []admissionregistrationv1.RuleWithOperations{{}},
				SideEffects:   &sideEffects,
				FailurePolicy: &failurePolicy,
				ClientConfig:  admissionregistrationv1.WebhookClientConfig{},
			},
		},
	}
}

func contextWithLogger(t *testing.T) context.Context {
	t.Helper()
	return logging.WithLogger(context.Background(), logging.NewArgoEventsLogger())
}

func contextWithLoggerAndCancel(t *testing.T) context.Context {
	t.Helper()
	return logging.WithLogger(signals.SetupSignalHandler(), logging.NewArgoEventsLogger())
}

func fakeAdmissionController(t *testing.T, options Options) *AdmissionController {
	t.Helper()
	ac := &AdmissionController{
		Client:  fakeClient.NewSimpleClientset(),
		Options: options,
		Handlers: map[schema.GroupVersionKind]runtime.Object{
			eventbusv1alphal1.SchemaGroupVersionKind:    &eventbusv1alphal1.EventBus{},
			eventsourcev1alphal1.SchemaGroupVersionKind: &eventsourcev1alphal1.EventSource{},
			sensorv1alphal1.SchemaGroupVersionKind:      &sensorv1alphal1.Sensor{},
		},
		Logger: logging.NewArgoEventsLogger(),
	}
	return ac
}

func TestConnectAllowed(t *testing.T) {
	ac := fakeAdmissionController(t, fakeOptions())
	t.Run("test CONNECT allowed", func(t *testing.T) {
		req := &admissionv1.AdmissionRequest{
			Operation: admissionv1.Connect,
		}
		resp := ac.admit(contextWithLogger(t), req)
		assert.True(t, resp.Allowed)
	})
}

func TestDeleteAllowed(t *testing.T) {
	ac := fakeAdmissionController(t, fakeOptions())
	t.Run("test DELETE allowed", func(t *testing.T) {
		req := &admissionv1.AdmissionRequest{
			Operation: admissionv1.Delete,
		}
		resp := ac.admit(contextWithLogger(t), req)
		assert.True(t, resp.Allowed)
	})
}

func TestUnknownKindFails(t *testing.T) {
	ac := fakeAdmissionController(t, fakeOptions())

	t.Run("test unknown types", func(t *testing.T) {
		req := &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Kind: metav1.GroupVersionKind{
				Group:   "test.test.test",
				Version: "v1alpha1",
				Kind:    "Unknown",
			},
		}
		resp := ac.admit(contextWithLogger(t), req)
		assert.True(t, !resp.Allowed)
	})
}

func TestDefaultClientAuth(t *testing.T) {
	opts := fakeOptions()
	assert.Equal(t, opts.ClientAuth, tls.NoClientCert)
}

func createDeployment(ac *AdmissionController) {
	opts := fakeOptions()
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.DeploymentName,
			Namespace: opts.Namespace,
		},
	}
	_, _ = ac.Client.AppsV1().Deployments(opts.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
}

func createWebhook(ac *AdmissionController, wh *admissionregistrationv1.ValidatingWebhookConfiguration) {
	client := ac.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations()
	_, err := client.Create(context.TODO(), wh, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to create test webhook, %w", err))
	}
}

func TestRun(t *testing.T) {
	opts := fakeOptions()
	ac := fakeAdmissionController(t, opts)
	createDeployment(ac)
	webhook := fakeValidatingWebhookConfig(opts)
	createWebhook(ac, webhook)

	ctx := contextWithLoggerAndCancel(t)
	go func() {
		_ = ac.Run(ctx)
	}()
	_, err := net.Dial("tcp", fmt.Sprintf(":%d", opts.Port))
	assert.NotNil(t, err)
}

func TestConfigureCertWithExistingSecret(t *testing.T) {
	t.Run("test configure cert with existing secret", func(t *testing.T) {
		opts := fakeOptions()
		ac := fakeAdmissionController(t, opts)
		createDeployment(ac)
		ctx := contextWithLogger(t)
		newSecret, err := ac.generateSecret(ctx)
		assert.Nil(t, err)
		_, err = ac.Client.CoreV1().Secrets(opts.Namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		assert.Nil(t, err)

		tlsConfig, caCert, err := ac.configureCerts(ctx, tls.NoClientCert)
		assert.Nil(t, err)
		assert.NotNil(t, tlsConfig)

		expectedCert, err := tls.X509KeyPair(newSecret.Data[secretServerCert], newSecret.Data[secretServerKey])
		assert.Nil(t, err)
		assert.True(t, len(tlsConfig.Certificates) >= 1)

		assert.Equal(t, expectedCert.Certificate, tlsConfig.Certificates[0].Certificate)
		assert.Equal(t, newSecret.Data[secretCACert], caCert)
	})
}
