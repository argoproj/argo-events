package webhook

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-openapi/inflect"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	clientadmissionregistrationv1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"

	"github.com/argoproj/argo-events/common/logging"
	commontls "github.com/argoproj/argo-events/common/tls"
	"github.com/argoproj/argo-events/webhook/validator"
)

const (
	secretServerKey  = "server-key.pem"
	secretServerCert = "server-cert.pem"
	secretCACert     = "ca-cert.pem"

	certOrg = "io.argoproj"
)

// Options contains the configuration for the webhook
type Options struct {
	// WebhookName is the name of the webhook
	WebhookName string

	// ServiceName is the service name of the webhook.
	ServiceName string

	// DeploymentName is the deployment name of the webhook.
	DeploymentName string

	// SecretName is the name of k8s secret that contains the webhook
	// server key/cert and corresponding CA cert that signed them. The
	// server key/cert are used to serve the webhook and the CA cert
	// is provided to k8s apiserver during admission controller
	// registration.
	SecretName string

	// Namespace is the namespace in which everything above lives
	Namespace string

	// Port where the webhook is served. Per k8s admission
	// registration requirements this should be 443 unless there is
	// only a single port for the service.
	Port int

	// ClientAuthType declares the policy the webhook server will follow for
	// TLS Client Authentication.
	// The default value is tls.NoClientCert.
	ClientAuth tls.ClientAuthType
}

// AdmissionController implements a webhook to validate specs
type AdmissionController struct {
	Client   kubernetes.Interface
	Options  Options
	Handlers map[schema.GroupVersionKind]runtime.Object

	Logger *zap.SugaredLogger
}

// Run implements the admission controller run loop.
func (ac *AdmissionController) Run(ctx context.Context) error {
	logger := ac.Logger
	tlsConfig, caCert, err := ac.configureCerts(ctx, ac.Options.ClientAuth)
	if err != nil {
		logger.Errorw("Could not configure admission webhook certs", zap.Error(err))
		return err
	}
	server := &http.Server{
		Handler:   ac,
		Addr:      fmt.Sprintf(":%v", ac.Options.Port),
		TLSConfig: tlsConfig,
	}
	cl := ac.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations()
	if err := ac.register(ctx, cl, caCert); err != nil {
		logger.Errorw("Failed to register webhook", zap.Error(err))
		return err
	}
	logger.Info("Successfully registered webhook")

	serverStartErrCh := make(chan struct{})
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Errorw("ListenAndServeTLS for admission webhook errored out", zap.Error(err))
			close(serverStartErrCh)
		}
	}()
	select {
	case <-ctx.Done():
		return server.Close()
	case <-serverStartErrCh:
		return errors.New("webhook server failed to start")
	}
}

// Register registers the external admission webhook for pilot
// configuration types.
func (ac *AdmissionController) register(
	ctx context.Context, client clientadmissionregistrationv1.ValidatingWebhookConfigurationInterface, caCert []byte) error {
	failurePolicy := admissionregistrationv1.Fail

	var rules []admissionregistrationv1.RuleWithOperations
	for gvk := range ac.Handlers {
		plural := strings.ToLower(inflect.Pluralize(gvk.Kind))

		rules = append(rules, admissionregistrationv1.RuleWithOperations{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{gvk.Group},
				APIVersions: []string{gvk.Version},
				Resources:   []string{plural},
			},
		})
	}

	// sort
	sort.Slice(rules, func(i, j int) bool {
		lhs, rhs := rules[i], rules[j]
		if lhs.APIGroups[0] != rhs.APIGroups[0] {
			return lhs.APIGroups[0] < rhs.APIGroups[0]
		}
		if lhs.APIVersions[0] != rhs.APIVersions[0] {
			return lhs.APIVersions[0] < rhs.APIVersions[0]
		}
		return lhs.Resources[0] < rhs.Resources[0]
	})

	sideEffects := admissionregistrationv1.SideEffectClassNone

	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{{
			Name:                    ac.Options.WebhookName,
			Rules:                   rules,
			SideEffects:             &sideEffects,
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				Service: &admissionregistrationv1.ServiceReference{
					Namespace: ac.Options.Namespace,
					Name:      ac.Options.ServiceName,
				},
				CABundle: caCert,
			},
			FailurePolicy: &failurePolicy,
		}},
	}
	deployment, err := ac.Client.AppsV1().Deployments(ac.Options.Namespace).Get(ctx, ac.Options.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch webhook deployment")
	}
	deploymentRef := metav1.NewControllerRef(deployment, appsv1.SchemeGroupVersion.WithKind("Deployment"))
	webhook.OwnerReferences = append(webhook.OwnerReferences, *deploymentRef)

	_, err = client.Create(ctx, webhook, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create a webhook")
		}
		ac.Logger.Info("Webhook already exists")
		configuredWebhook, err := client.Get(ctx, ac.Options.WebhookName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to retrieve webhook")
		}
		if !reflect.DeepEqual(configuredWebhook.Webhooks, webhook.Webhooks) {
			ac.Logger.Info("Updating webhook")
			// Set the ResourceVersion as required by update.
			webhook.ObjectMeta.ResourceVersion = configuredWebhook.ObjectMeta.ResourceVersion
			if _, err := client.Update(ctx, webhook, metav1.UpdateOptions{}); err != nil {
				return errors.Wrap(err, "failed to update webhook")
			}
		} else {
			ac.Logger.Info("Webhook is valid")
		}
	} else {
		ac.Logger.Info("Created a webhook")
	}
	return nil
}

// ServeHTTP implements the external admission webhook for mutating
// serving resources.
func (ac *AdmissionController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ac.Logger.Infof("Webhook ServeHTTP request=%#v", r)

	// content type validation
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "invalid Content-Type, want `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var review admissionv1.AdmissionReview
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusBadRequest)
		return
	}
	logger := ac.Logger.With("kind", fmt.Sprint(review.Request.Kind)).
		With("namespace", review.Request.Namespace).
		With("name", review.Request.Name).
		With("operation", fmt.Sprint(review.Request.Operation)).
		With("resource", fmt.Sprint(review.Request.Resource)).
		With("subResource", fmt.Sprint(review.Request.SubResource)).
		With("userInfo", fmt.Sprint(review.Request.UserInfo))

	reviewResponse := ac.admit(logging.WithLogger(r.Context(), logger), review.Request)
	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
	}
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}

	logger.Infof("AdmissionReview for %s: %v/%v response=%v",
		review.Request.Kind, review.Request.Namespace, review.Request.Name, reviewResponse)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("could encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (ac *AdmissionController) admit(ctx context.Context, request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log := logging.FromContext(ctx)
	switch request.Operation {
	case admissionv1.Create, admissionv1.Update:
	default:
		log.Infof("Operation not interested: %v %v", request.Kind, request.Operation)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}
	validator, err := validator.GetValidator(ctx, ac.Client, request.Kind, request.OldObject.Raw, request.Object.Raw)
	if err != nil {
		return makeErrorStatus("failed to get a validator: %v", err)
	}

	switch request.Operation {
	case admissionv1.Create:
		result, message, err := validator.ValidateCreate(ctx)
		if err != nil {
			return makeErrorStatus("Object CREATE validation failed: %v", err)
		}
		if result {
			return &admissionv1.AdmissionResponse{Allowed: true}
		}
		return makeErrorStatus(message)
	case admissionv1.Update:
		result, message, err := validator.ValidateUpdate(ctx)
		if err != nil {
			return makeErrorStatus("Object UPDATE validation failed: %v", err)
		}
		if result {
			return &admissionv1.AdmissionResponse{Allowed: true}
		}
		return makeErrorStatus(message)
	default:
		return &admissionv1.AdmissionResponse{Allowed: true}
	}
}

func makeErrorStatus(reason string, args ...interface{}) *admissionv1.AdmissionResponse {
	result := apierrors.NewBadRequest(fmt.Sprintf(reason, args...)).Status()
	return &admissionv1.AdmissionResponse{
		Result:  &result,
		Allowed: false,
	}
}

// Generate cert secret
func (ac *AdmissionController) generateSecret(ctx context.Context) (*corev1.Secret, error) {
	hosts := []string{}
	hosts = append(hosts, fmt.Sprintf("%s.%s.svc.cluster.local", ac.Options.ServiceName, ac.Options.Namespace))
	hosts = append(hosts, fmt.Sprintf("%s.%s.svc", ac.Options.ServiceName, ac.Options.Namespace))
	serverKey, serverCert, caCert, err := commontls.CreateCerts(certOrg, hosts, time.Now().Add(10*365*24*time.Hour))
	if err != nil {
		return nil, err
	}
	deployment, err := ac.Client.AppsV1().Deployments(ac.Options.Namespace).Get(ctx, ac.Options.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch webhook deployment")
	}
	deploymentRef := metav1.NewControllerRef(deployment, appsv1.SchemeGroupVersion.WithKind("Deployment"))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Options.SecretName,
			Namespace: ac.Options.Namespace,
		},
		Data: map[string][]byte{
			secretServerKey:  serverKey,
			secretServerCert: serverCert,
			secretCACert:     caCert,
		},
	}
	secret.OwnerReferences = append(secret.OwnerReferences, *deploymentRef)
	return secret, nil
}

// getOrGenerateKeyCertsFromSecret creates CERTs if not existing and store in a secret
func (ac *AdmissionController) getOrGenerateKeyCertsFromSecret(ctx context.Context) (serverKey, serverCert, caCert []byte, err error) {
	secret, err := ac.Client.CoreV1().Secrets(ac.Options.Namespace).Get(ctx, ac.Options.SecretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, nil, err
		}
		// No existing secret, creating one
		newSecret, err := ac.generateSecret(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		secret, err = ac.Client.CoreV1().Secrets(newSecret.Namespace).Create(ctx, newSecret, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, nil, nil, err
		}
		// Something else might have created, try fetching it one more time
		secret, err = ac.Client.CoreV1().Secrets(ac.Options.Namespace).Get(ctx, ac.Options.SecretName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var ok bool
	if serverKey, ok = secret.Data[secretServerKey]; !ok {
		return nil, nil, nil, errors.New("server key missing")
	}
	if serverCert, ok = secret.Data[secretServerCert]; !ok {
		return nil, nil, nil, errors.New("server cert missing")
	}
	if caCert, ok = secret.Data[secretCACert]; !ok {
		return nil, nil, nil, errors.New("ca cert missing")
	}
	return serverKey, serverCert, caCert, nil
}

// GetAPIServerExtensionCACert gets the K8s aggregate apiserver
// client CA cert used by validator. This certificate is provided
// kubernetes. We do not control its name or location.
func (ac *AdmissionController) getAPIServerExtensionCACert(ctx context.Context) ([]byte, error) {
	const name = "extension-apiserver-authentication"
	c, err := ac.Client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	const caFileName = "requestheader-client-ca-file"
	pem, ok := c.Data[caFileName]
	if !ok {
		return nil, errors.Errorf("cannot find %s in ConfigMap %s: ConfigMap.Data is %#v", caFileName, name, c.Data)
	}
	return []byte(pem), nil
}

func (ac *AdmissionController) configureCerts(ctx context.Context, clientAuth tls.ClientAuthType) (*tls.Config, []byte, error) {
	var apiServerCACert []byte
	if clientAuth >= tls.VerifyClientCertIfGiven {
		var err error
		apiServerCACert, err = ac.getAPIServerExtensionCACert(ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	serverKey, serverCert, caCert, err := ac.getOrGenerateKeyCertsFromSecret(ctx)
	if err != nil {
		return nil, nil, err
	}
	tlsConfig, err := makeTLSConfig(serverCert, serverKey, apiServerCACert, clientAuth)
	if err != nil {
		return nil, nil, err
	}
	return tlsConfig, caCert, nil
}

// makeTLSConfig makes a TLS configuration suitable for use with the server
func makeTLSConfig(serverCert, serverKey, caCert []byte, clientAuthType tls.ClientAuthType) (*tls.Config, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   clientAuthType,
	}, nil
}
