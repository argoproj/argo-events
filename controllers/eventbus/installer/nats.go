package installer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	clientPort  = int32(4222)
	clusterPort = int32(6222)
	monitorPort = int32(8222)

	// annotation key on serverAuthSecret and clientAuthsecret
	authStrategyAnnoKey = "strategy"
	// key of client auth secret
	clientAuthSecretKey = "client-auth"
	// key of server auth secret
	serverAuthSecretKey = "auth"
	// key of stan.conf in the configmap
	configMapKey = "stan-config"

	// default nats streaming version to be installed
	defaultSTANVersion = "0.22.1"
)

// natsInstaller is used create a NATS installation.
type natsInstaller struct {
	client     client.Client
	kubeClient kubernetes.Interface
	eventBus   *v1alpha1.EventBus
	config     *controllers.GlobalConfig
	labels     map[string]string
	logger     *zap.SugaredLogger
}

// NewNATSInstaller returns a new NATS installer
func NewNATSInstaller(client client.Client, eventBus *v1alpha1.EventBus, config *controllers.GlobalConfig, labels map[string]string, kubeClient kubernetes.Interface, logger *zap.SugaredLogger) Installer {
	return &natsInstaller{
		client:     client,
		kubeClient: kubeClient,
		eventBus:   eventBus,
		config:     config,
		labels:     labels,
		logger:     logger.Named("nats"),
	}
}

// Install creats a StatefulSet and a Service for NATS
func (i *natsInstaller) Install(ctx context.Context) (*v1alpha1.BusConfig, error) {
	natsObj := i.eventBus.Spec.NATS
	if natsObj == nil || natsObj.Native == nil {
		return nil, fmt.Errorf("invalid request")
	}

	svc, err := i.createStanService(ctx)
	if err != nil {
		return nil, err
	}
	cm, err := i.createConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	// default to none
	defaultAuthStrategy := v1alpha1.AuthStrategyNone
	authStrategy := natsObj.Native.Auth
	if authStrategy == nil {
		authStrategy = &defaultAuthStrategy
	}
	serverAuthSecret, clientAuthSecret, err := i.createAuthSecrets(ctx, *authStrategy)
	if err != nil {
		return nil, err
	}

	if err := i.createStatefulSet(ctx, svc.Name, cm.Name, serverAuthSecret.Name); err != nil {
		return nil, err
	}
	i.eventBus.Status.MarkDeployed("Succeeded", "NATS is deployed")
	clusterID := generateClusterID(i.eventBus)
	busConfig := &v1alpha1.BusConfig{
		NATS: &v1alpha1.NATSConfig{
			URL:       fmt.Sprintf("nats://%s:%s", generateServiceName(i.eventBus), strconv.Itoa(int(clientPort))),
			ClusterID: &clusterID,
			Auth:      authStrategy,
		},
	}
	if *authStrategy != v1alpha1.AuthStrategyNone {
		busConfig.NATS.AccessSecret = &corev1.SecretKeySelector{
			Key: clientAuthSecretKey,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: clientAuthSecret.Name,
			},
		}
	}
	return busConfig, nil
}

// Uninstall deletes those objects not handeled by cascade deletion.
func (i *natsInstaller) Uninstall(ctx context.Context) error {
	return i.uninstallPVCs(ctx)
}

func (i *natsInstaller) uninstallPVCs(ctx context.Context) error {
	// StatefulSet doesn't clean up PVC, needs to do it separately
	// https://github.com/kubernetes/kubernetes/issues/55045
	log := i.logger
	pvcs, err := i.getPVCs(ctx, i.labels)
	if err != nil {
		log.Errorw("Failed to get PVCs created by nats streaming statefulset when uninstalling", zap.Error(err))
		return err
	}
	for _, pvc := range pvcs {
		err = i.client.Delete(ctx, &pvc)
		if err != nil {
			log.Errorw("Failed to delete pvc when uninstalling", zap.Any("pvcName", pvc.Name), zap.Error(err))
			return err
		}
		log.Infow("Pvc deleted", "pvcName", pvc.Name)
	}
	return nil
}

// Create a service for nats streaming
func (i *natsInstaller) createStanService(ctx context.Context) (*corev1.Service, error) {
	log := i.logger
	svc, err := i.getStanService(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServiceFailed", "Get existing service failed")
		log.Errorw("Error getting existing service", zap.Error(err))
		return nil, err
	}
	expectedSvc, err := i.buildStanService()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildServiceFailed", "Failed to build a service spec")
		log.Errorw("Error building service spec", zap.Error(err))
		return nil, err
	}
	if svc != nil {
		// TODO: potential issue here - if service spec is updated manually, reconciler will not change it back.
		// Revisit it later to see if it is needed to compare the spec.
		if svc.Annotations != nil && svc.Annotations[common.AnnotationResourceSpecHash] != expectedSvc.Annotations[common.AnnotationResourceSpecHash] {
			svc.Spec = expectedSvc.Spec
			svc.SetLabels(expectedSvc.Labels)
			svc.Annotations[common.AnnotationResourceSpecHash] = expectedSvc.Annotations[common.AnnotationResourceSpecHash]
			err = i.client.Update(ctx, svc)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServiceFailed", "Failed to update existing service")
				log.Errorw("Error updating existing service", zap.Error(err))
				return nil, err
			}
			log.Infow("Service is updated", "serviceName", svc.Name)
		}
		return svc, nil
	}
	err = i.client.Create(ctx, expectedSvc)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateServiceFailed", "Failed to create a service")
		log.Errorw("Error creating a service", zap.Error(err))
		return nil, err
	}
	log.Infow("Service is created", "serviceName", expectedSvc.Name)
	return expectedSvc, nil
}

// Create a Configmap for NATS config
func (i *natsInstaller) createConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	log := i.logger
	cm, err := i.getConfigMap(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetConfigMapFailed", "Failed to get existing configmap")
		log.Errorw("Error getting existing configmap", zap.Error(err))
		return nil, err
	}
	expectedCm, err := i.buildConfigMap()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildConfigMapFailed", "Failed to build a configmap spec")
		log.Errorw("Error building configmap spec", zap.Error(err))
		return nil, err
	}
	if cm != nil {
		// TODO: Potential issue about comparing hash
		if cm.Annotations != nil && cm.Annotations[common.AnnotationResourceSpecHash] != expectedCm.Annotations[common.AnnotationResourceSpecHash] {
			cm.Data = expectedCm.Data
			cm.SetLabels(expectedCm.Labels)
			cm.Annotations[common.AnnotationResourceSpecHash] = expectedCm.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, cm)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateConfigMapFailed", "Failed to update existing configmap")
				log.Errorw("Error updating configmap", zap.Error(err))
				return nil, err
			}
			log.Infow("Updated configmap", "configmapName", cm.Name)
		}
		return cm, nil
	}
	err = i.client.Create(ctx, expectedCm)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateConfigMapFailed", "Failed to create configmap")
		log.Errorw("Error creating a configmap", zap.Error(err))
		return nil, err
	}
	log.Infow("Created configmap", "configmapName", expectedCm.Name)
	return expectedCm, nil
}

// create server and client auth secrets
func (i *natsInstaller) createAuthSecrets(ctx context.Context, strategy v1alpha1.AuthStrategy) (*corev1.Secret, *corev1.Secret, error) {
	log := i.logger
	sSecret, err := i.getServerAuthSecret(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServerAuthSecretFailed", "Failed to get existing server auth secret")
		log.Errorw("Error getting existing server auth secret", zap.Error(err))
		return nil, nil, err
	}
	cSecret, err := i.getClientAuthSecret(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetClientAuthSecretFailed", "Failed to get existing client auth secret")
		log.Errorw("Error getting existing client auth secret", zap.Error(err))
		return nil, nil, err
	}
	if strategy != v1alpha1.AuthStrategyNone { // Do not checkout AuthStrategyNone because it only has server auth secret
		if sSecret != nil && cSecret != nil && sSecret.Annotations != nil && cSecret.Annotations != nil {
			if sSecret.Annotations[authStrategyAnnoKey] == string(strategy) && cSecret.Annotations[authStrategyAnnoKey] == string(strategy) {
				// If the secrets are already existing, and strategy didn't change, reuse them without updating.
				return sSecret, cSecret, nil
			}
		}
	}

	switch strategy {
	case v1alpha1.AuthStrategyNone:
		// Clean up client auth secret if existing
		if cSecret != nil {
			err = i.client.Delete(ctx, cSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("DeleteClientAuthSecretFailed", "Failed to delete the client auth secret")
				log.Errorw("Error deleting client auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Info("Deleted server auth secret")
		}
		if sSecret != nil && sSecret.Annotations != nil && sSecret.Annotations[authStrategyAnnoKey] == string(strategy) && len(sSecret.Data[serverAuthSecretKey]) == 0 {
			// If the server auth secret is already existing, strategy didn't change, and the secret is empty string, reuse it without updating.
			return sSecret, nil, nil
		}
		// Only create an empty server auth secret
		expectedSSecret, err := i.buildServerAuthSecret(strategy, "")
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildServerAuthSecretFailed", "Failed to build a server auth secret spec")
			log.Errorw("Error building server auth secret spec", zap.Error(err))
			return nil, nil, err
		}
		if sSecret != nil {
			sSecret.ObjectMeta.Labels = expectedSSecret.Labels
			sSecret.ObjectMeta.Annotations = expectedSSecret.Annotations
			sSecret.Data = expectedSSecret.Data
			err = i.client.Update(ctx, sSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerAuthSecretFailed", "Failed to update the server auth secret")
				log.Errorw("Error updating server auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Infow("Updated server auth secret", "serverAuthSecretName", sSecret.Name)
			return sSecret, nil, nil
		}
		err = i.client.Create(ctx, expectedSSecret)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateServerAuthSecretFailed", "Failed to create a server auth secret")
			log.Errorw("Error creating server auth secret", zap.Error(err))
			return nil, nil, err
		}
		log.Infow("Created server auth secret", "serverAuthSecretName", expectedSSecret.Name)
		return expectedSSecret, nil, nil
	case v1alpha1.AuthStrategyToken:
		token := common.RandomString(64)
		serverAuthText := fmt.Sprintf(`authorization {
  token: "%s"
}`, token)
		clientAuthText := fmt.Sprintf("token: \"%s\"", token)
		// Create server auth secret
		expectedSSecret, err := i.buildServerAuthSecret(strategy, serverAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildServerAuthSecretFailed", "Failed to build a server auth secret spec")
			log.Errorw("Error building server auth secret spec", zap.Error(err))
			return nil, nil, err
		}
		returnedSSecret := expectedSSecret
		if sSecret == nil {
			err = i.client.Create(ctx, expectedSSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("CreateServerAuthSecretFailed", "Failed to create a server auth secret")
				log.Errorw("Error creating server auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Infow("Created server auth secret", "serverAuthSecretName", expectedSSecret.Name)
		} else {
			sSecret.Data = expectedSSecret.Data
			sSecret.SetLabels(expectedSSecret.Labels)
			sSecret.SetAnnotations(expectedSSecret.Annotations)
			err = i.client.Update(ctx, sSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerAuthSecretFailed", "Failed to update the server auth secret")
				log.Errorw("Error updating server auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Infow("Updated server auth secret", "serverAuthSecretName", sSecret.Name)
			returnedSSecret = sSecret
		}
		// create client auth secret
		expectedCSecret, err := i.buildClientAuthSecret(strategy, clientAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildClientAuthSecretFailed", "Failed to build a client auth secret spec")
			log.Errorw("Error building client auth secret spec", zap.Error(err))
			return nil, nil, err
		}
		returnedCSecret := expectedCSecret
		if cSecret == nil {
			err = i.client.Create(ctx, expectedCSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("CreateClientAuthSecretFailed", "Failed to create a client auth secret")
				log.Errorw("Error creating client auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Infow("Created client auth secret", "clientAuthSecretName", expectedCSecret.Name)
		} else {
			cSecret.Data = expectedCSecret.Data
			cSecret.SetLabels(expectedCSecret.Labels)
			cSecret.SetAnnotations(expectedCSecret.Annotations)
			err = i.client.Update(ctx, cSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateClientAuthSecretFailed", "Failed to update the client auth secret")
				log.Errorw("Error updating client auth secret", zap.Error(err))
				return nil, nil, err
			}
			log.Infow("Updated client auth secret", "clientAuthSecretName", cSecret.Name)
			returnedCSecret = cSecret
		}
		return returnedSSecret, returnedCSecret, nil
	default:
		i.eventBus.Status.MarkDeployFailed("UnsupportedAuthStrategy", "Unsupported auth strategy")
		return nil, nil, fmt.Errorf("unsupported auth strategy")
	}
}

// Create a StatefulSet
func (i *natsInstaller) createStatefulSet(ctx context.Context, serviceName, configmapName, authSecretName string) error {
	log := i.logger
	ss, err := i.getStatefulSet(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStatefulSetFailed", "Failed to get existing statefulset")
		log.Errorw("Error getting existing statefulset", zap.Error(err))
		return err
	}
	expectedSs, err := i.buildStatefulSet(serviceName, configmapName, authSecretName)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildStatefulSetFailed", "Failed to build a statefulset spec")
		log.Errorw("Error building statefulset spec", zap.Error(err))
		return err
	}
	if ss != nil {
		if ss.Annotations != nil && ss.Annotations[common.AnnotationResourceSpecHash] == expectedSs.Annotations[common.AnnotationResourceSpecHash] {
			return nil
		}
		ss.SetLabels(expectedSs.Labels)
		ss.Annotations[common.AnnotationResourceSpecHash] = expectedSs.Annotations[common.AnnotationResourceSpecHash]
		ss.Spec = expectedSs.Spec
		if err := i.client.Update(ctx, ss); err != nil {
			i.eventBus.Status.MarkDeployFailed("UpdateStatefulSetFailed", "Failed to update a statefulset")
			log.Errorw("Error updating a statefulset", zap.Error(err))
			return err
		}
		log.Infow("Statefulset is updated", "statefulsetName", ss.Name)
		return nil
	}
	if err := i.client.Create(ctx, expectedSs); err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateStatefulSetFailed", "Failed to create a statefulset")
		log.Errorw("Error creating a statefulset", zap.Error(err))
		return err
	}
	log.Infow("Statefulset is created", "statefulsetName", expectedSs.Name)
	return nil
}

// buildStanService builds a Service for NATS streaming
func (i *natsInstaller) buildStanService() (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateServiceName(i.eventBus),
			Namespace: i.eventBus.Namespace,
			Labels:    i.mergeEventBusLabels(stanServiceLabels(i.labels)),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				// Prefix tcp- to enable clients to connect from
				// an istio-enabled namespace, following:
				// https://github.com/nats-io/nats-operator/issues/88
				// https://github.com/istio/istio/issues/28623
				{Name: "tcp-client", Port: clientPort},
				{Name: "cluster", Port: clusterPort},
				{Name: "monitor", Port: monitorPort},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: i.labels,
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

// buildConfigMap builds a ConfigMap for NATS streaming
func (i *natsInstaller) buildConfigMap() (*corev1.ConfigMap, error) {
	clusterID := generateClusterID(i.eventBus)
	svcName := generateServiceName(i.eventBus)
	ssName := generateStatefulSetName(i.eventBus)
	replicas := i.eventBus.Spec.NATS.Native.GetReplicas()
	if replicas < 3 {
		replicas = 3
	}
	maxAge := common.STANMaxAge
	if i.eventBus.Spec.NATS.Native.MaxAge != nil {
		maxAge = *i.eventBus.Spec.NATS.Native.MaxAge
	}
	_, err := time.ParseDuration(maxAge)
	if err != nil {
		return nil, err
	}
	maxMsgs := common.STANMaxMsgs
	if i.eventBus.Spec.NATS.Native.MaxMsgs != nil {
		maxMsgs = *i.eventBus.Spec.NATS.Native.MaxMsgs
	}
	maxSubs := common.STANMaxSubs
	if i.eventBus.Spec.NATS.Native.MaxSubs != nil {
		maxSubs = *i.eventBus.Spec.NATS.Native.MaxSubs
	}
	maxBytes := common.STANMaxBytes
	if i.eventBus.Spec.NATS.Native.MaxBytes != nil {
		maxBytes = *i.eventBus.Spec.NATS.Native.MaxBytes
	}
	maxPayload := common.STANMaxPayload
	if i.eventBus.Spec.NATS.Native.MaxPayload != nil {
		maxPayload = *i.eventBus.Spec.NATS.Native.MaxPayload
	}
	raftHeartbeatTimeout := common.STANRaftHeartbeatTimeout
	if i.eventBus.Spec.NATS.Native.RaftHeartbeatTimeout != nil {
		raftHeartbeatTimeout = *i.eventBus.Spec.NATS.Native.RaftHeartbeatTimeout
	}
	_, err = time.ParseDuration(raftHeartbeatTimeout)
	if err != nil {
		return nil, err
	}
	raftElectionTimeout := common.STANRaftElectionTimeout
	if i.eventBus.Spec.NATS.Native.RaftElectionTimeout != nil {
		raftElectionTimeout = *i.eventBus.Spec.NATS.Native.RaftElectionTimeout
	}
	_, err = time.ParseDuration(raftElectionTimeout)
	if err != nil {
		return nil, err
	}
	raftLeaseTimeout := common.STANRaftLeaseTimeout
	if i.eventBus.Spec.NATS.Native.RaftLeaseTimeout != nil {
		raftLeaseTimeout = *i.eventBus.Spec.NATS.Native.RaftLeaseTimeout
	}
	_, err = time.ParseDuration(raftLeaseTimeout)
	if err != nil {
		return nil, err
	}
	raftCommitTimeout := common.STANRaftCommitTimeout
	if i.eventBus.Spec.NATS.Native.RaftCommitTimeout != nil {
		raftCommitTimeout = *i.eventBus.Spec.NATS.Native.RaftCommitTimeout
	}
	_, err = time.ParseDuration(raftCommitTimeout)
	if err != nil {
		return nil, err
	}
	peers := []string{}
	routes := []string{}
	for j := 0; j < replicas; j++ {
		peers = append(peers, fmt.Sprintf("\"%s-%s\"", ssName, strconv.Itoa(j)))
		routes = append(routes, fmt.Sprintf("nats://%s-%s.%s.%s.svc:%s", ssName, strconv.Itoa(j), svcName, i.eventBus.Namespace, strconv.Itoa(int(clusterPort))))
	}
	conf := fmt.Sprintf(`http: %s
include ./auth.conf
cluster {
  port: %s
  routes: [%s]
  cluster_advertise: $CLUSTER_ADVERTISE
  connect_retries: 10
}
max_payload: %s
streaming {
  id: %s
  store: file
  dir: /data/stan/store
  cluster {
	node_id: $POD_NAME
	peers: [%s]
	log_path: /data/stan/logs
	raft_heartbeat_timeout: "%s"
	raft_election_timeout: "%s"
	raft_lease_timeout: "%s"
	raft_commit_timeout: "%s"
  }
  store_limits {
    max_age: %s
	max_msgs: %v
	max_bytes: %s
	max_subs: %v
  }
}`, strconv.Itoa(int(monitorPort)), strconv.Itoa(int(clusterPort)), strings.Join(routes, ","), maxPayload, clusterID, strings.Join(peers, ","), raftHeartbeatTimeout, raftElectionTimeout, raftLeaseTimeout, raftCommitTimeout, maxAge, maxMsgs, maxBytes, maxSubs)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateConfigMapName(i.eventBus),
			Labels:    i.mergeEventBusLabels(i.labels),
		},
		Data: map[string]string{
			configMapKey: conf,
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, cm, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return cm, nil
}

// buildServerAuthSecret builds a secret for NATS auth config
// Parameter - authStrategy: will be added to annoations
// Parameter - secret
// Example:
//
//	authorization {
//	  token: "abcd1234"
//	}
func (i *natsInstaller) buildServerAuthSecret(authStrategy v1alpha1.AuthStrategy, secret string) (*corev1.Secret, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateServerAuthSecretName(i.eventBus),
			Labels:      serverAuthSecretLabels(i.labels),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			serverAuthSecretKey: []byte(secret),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildClientAuthSecret builds a secret for NATS client auth
func (i *natsInstaller) buildClientAuthSecret(authStrategy v1alpha1.AuthStrategy, secret string) (*corev1.Secret, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateClientAuthSecretName(i.eventBus),
			Labels:      clientAuthSecretLabels(i.labels),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			clientAuthSecretKey: []byte(secret),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildStatefulSet builds a StatefulSet for nats streaming
func (i *natsInstaller) buildStatefulSet(serviceName, configmapName, authSecretName string) (*appv1.StatefulSet, error) {
	// Use provided serviceName, configMapName to build the spec
	// to avoid issues when naming convention changes
	spec, err := i.buildStatefulSetSpec(serviceName, configmapName, authSecretName)
	if err != nil {
		return nil, err
	}
	ss := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateStatefulSetName(i.eventBus),
			Labels:    i.mergeEventBusLabels(i.labels),
		},
		Spec: *spec,
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, ss, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return ss, nil
}

func (i *natsInstaller) buildStatefulSetSpec(serviceName, configmapName, authSecretName string) (*appv1.StatefulSetSpec, error) {
	stanVersion, err := i.config.GetSTANVersion(defaultSTANVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get nats streaming version, err: %w", err)
	}
	// Streaming requires minimal size 3.
	replicas := i.eventBus.Spec.NATS.Native.Replicas
	if replicas < 3 {
		replicas = 3
	}
	var stanContainerPullPolicy, metricsContainerPullPolicy corev1.PullPolicy
	var stanContainerSecurityContext, metricsContainerSecurityContext *corev1.SecurityContext
	stanContainerResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: apiresource.MustParse("0"),
		},
	}
	containerTmpl := i.eventBus.Spec.NATS.Native.ContainerTemplate
	if containerTmpl != nil {
		stanContainerResources = containerTmpl.Resources
		stanContainerPullPolicy = containerTmpl.ImagePullPolicy
		stanContainerSecurityContext = containerTmpl.SecurityContext
	}
	metricsContainerResources := corev1.ResourceRequirements{}
	metricsContainerTmpl := i.eventBus.Spec.NATS.Native.MetricsContainerTemplate
	if metricsContainerTmpl != nil {
		metricsContainerResources = metricsContainerTmpl.Resources
		metricsContainerPullPolicy = metricsContainerTmpl.ImagePullPolicy
		metricsContainerSecurityContext = metricsContainerTmpl.SecurityContext
	}

	podTemplateLabels := make(map[string]string)
	if i.eventBus.Spec.NATS.Native.Metadata != nil &&
		len(i.eventBus.Spec.NATS.Native.Metadata.Labels) > 0 {
		for k, v := range i.eventBus.Spec.NATS.Native.Metadata.Labels {
			podTemplateLabels[k] = v
		}
	}
	for k, v := range i.labels {
		podTemplateLabels[k] = v
	}
	spec := appv1.StatefulSetSpec{
		Replicas:    &replicas,
		ServiceName: serviceName,
		Selector: &metav1.LabelSelector{
			MatchLabels: i.labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podTemplateLabels,
			},
			Spec: corev1.PodSpec{
				NodeSelector:       i.eventBus.Spec.NATS.Native.NodeSelector,
				Tolerations:        i.eventBus.Spec.NATS.Native.Tolerations,
				SecurityContext:    i.eventBus.Spec.NATS.Native.SecurityContext,
				ImagePullSecrets:   i.eventBus.Spec.NATS.Native.ImagePullSecrets,
				ServiceAccountName: i.eventBus.Spec.NATS.Native.ServiceAccountName,
				PriorityClassName:  i.eventBus.Spec.NATS.Native.PriorityClassName,
				Priority:           i.eventBus.Spec.NATS.Native.Priority,
				Affinity:           i.eventBus.Spec.NATS.Native.Affinity,
				Volumes: []corev1.Volume{
					{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										ConfigMap: &corev1.ConfigMapProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configmapName,
											},
											Items: []corev1.KeyToPath{
												{
													Key:  configMapKey,
													Path: "stan.conf",
												},
											},
										},
									},
									{
										Secret: &corev1.SecretProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: authSecretName,
											},
											Items: []corev1.KeyToPath{
												{
													Key:  serverAuthSecretKey,
													Path: "auth.conf",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "stan",
						Image:           stanVersion.NATSStreamingImage,
						ImagePullPolicy: stanContainerPullPolicy,
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: clientPort},
							{Name: "cluster", ContainerPort: clusterPort},
							{Name: "monitor", ContainerPort: monitorPort},
						},
						Command: []string{"/nats-streaming-server", "-sc", "/etc/stan-config/stan.conf"},
						Env: []corev1.EnvVar{
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							{Name: "CLUSTER_ADVERTISE", Value: "$(POD_NAME)." + generateServiceName(i.eventBus) + ".$(POD_NAMESPACE).svc"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config-volume", MountPath: "/etc/stan-config"},
						},
						Resources:       stanContainerResources,
						SecurityContext: stanContainerSecurityContext,
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(monitorPort)),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
					},
					{
						Name:            "metrics",
						Image:           stanVersion.MetricsExporterImage,
						ImagePullPolicy: metricsContainerPullPolicy,
						Ports: []corev1.ContainerPort{
							{Name: "metrics", ContainerPort: common.EventBusMetricsPort},
						},
						Args:            []string{"-connz", "-routez", "-subz", "-varz", "-channelz", "-serverz", fmt.Sprintf("http://localhost:%s", strconv.Itoa(int(monitorPort)))},
						Resources:       metricsContainerResources,
						SecurityContext: metricsContainerSecurityContext,
					},
				},
			},
		},
	}
	if i.eventBus.Spec.NATS.Native.Metadata != nil {
		spec.Template.SetAnnotations(i.eventBus.Spec.NATS.Native.Metadata.Annotations)
	}
	if i.eventBus.Spec.NATS.Native.Persistence != nil {
		volMode := corev1.PersistentVolumeFilesystem
		pvcName := generatePVCName(i.eventBus)
		// Default volume size
		volSize := apiresource.MustParse("10Gi")
		if i.eventBus.Spec.NATS.Native.Persistence.VolumeSize != nil {
			volSize = *i.eventBus.Spec.NATS.Native.Persistence.VolumeSize
		}
		// Default to ReadWriteOnce
		accessMode := corev1.ReadWriteOnce
		if i.eventBus.Spec.NATS.Native.Persistence.AccessMode != nil {
			accessMode = *i.eventBus.Spec.NATS.Native.Persistence.AccessMode
		}
		spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: pvcName,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						accessMode,
					},
					VolumeMode:       &volMode,
					StorageClassName: i.eventBus.Spec.NATS.Native.Persistence.StorageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: volSize,
						},
					},
				},
			},
		}
		volumeMounts := spec.Template.Spec.Containers[0].VolumeMounts
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: pvcName, MountPath: "/data/stan"})
		spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	} else {
		// When the POD is runasnonroot, it can not create the dir /data/stan
		// Use an emptyDirVolume to workaround the issue
		emptyDirVolName := "stan-data"
		volumes := spec.Template.Spec.Volumes
		volumes = append(volumes, corev1.Volume{Name: emptyDirVolName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
		spec.Template.Spec.Volumes = volumes
		volumeMounts := spec.Template.Spec.Containers[0].VolumeMounts
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: emptyDirVolName, MountPath: "/data/stan"})
		spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	}
	return &spec, nil
}

func (i *natsInstaller) getStanService(ctx context.Context) (*corev1.Service, error) {
	return i.getService(ctx, stanServiceLabels(i.labels))
}

func (i *natsInstaller) getService(ctx context.Context, labels map[string]string) (*corev1.Service, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	sl := &corev1.ServiceList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, i.eventBus) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (i *natsInstaller) getConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cml := &corev1.ConfigMapList{}
	err := i.client.List(ctx, cml, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(i.labels),
	})
	if err != nil {
		return nil, err
	}
	for _, cm := range cml.Items {
		if metav1.IsControlledBy(&cm, i.eventBus) {
			return &cm, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

// get server auth secret
func (i *natsInstaller) getServerAuthSecret(ctx context.Context) (*corev1.Secret, error) {
	return i.getSecret(ctx, serverAuthSecretLabels(i.labels))
}

// get client auth secret
func (i *natsInstaller) getClientAuthSecret(ctx context.Context) (*corev1.Secret, error) {
	return i.getSecret(ctx, clientAuthSecretLabels(i.labels))
}

func (i *natsInstaller) getSecret(ctx context.Context, labels map[string]string) (*corev1.Secret, error) {
	sl, err := i.kubeClient.CoreV1().Secrets(i.eventBus.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector(labels).String()})
	if err != nil {
		return nil, err
	}
	for _, s := range sl.Items {
		if metav1.IsControlledBy(&s, i.eventBus) {
			return &s, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (i *natsInstaller) getStatefulSet(ctx context.Context) (*appv1.StatefulSet, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	ssl := &appv1.StatefulSetList{}
	err := i.client.List(ctx, ssl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(i.labels),
	})
	if err != nil {
		return nil, err
	}
	for _, ss := range ssl.Items {
		if metav1.IsControlledBy(&ss, i.eventBus) {
			return &ss, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

// get PVCs created by streaming statefulset
// they have same labels as the statefulset
func (i *natsInstaller) getPVCs(ctx context.Context, labels map[string]string) ([]corev1.PersistentVolumeClaim, error) {
	pvcl := &corev1.PersistentVolumeClaimList{}
	err := i.client.List(ctx, pvcl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
	})
	if err != nil {
		return nil, err
	}
	return pvcl.Items, nil
}

func (i *natsInstaller) mergeEventBusLabels(given map[string]string) map[string]string {
	result := map[string]string{}
	if i.eventBus.Labels != nil {
		for k, v := range i.eventBus.Labels {
			result[k] = v
		}
	}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func serverAuthSecretLabels(given map[string]string) map[string]string {
	result := map[string]string{"server-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func clientAuthSecretLabels(given map[string]string) map[string]string {
	result := map[string]string{"client-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func stanServiceLabels(given map[string]string) map[string]string {
	result := map[string]string{"stan": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}

func generateServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan-svc", eventBus.Name)
}

func generateConfigMapName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan-configmap", eventBus.Name)
}

func generateServerAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-server", eventBus.Name)
}

func generateClientAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-client", eventBus.Name)
}

func generateStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan", eventBus.Name)
}

func generateClusterID(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s", eventBus.Name)
}

// PVC name used in streaming statefulset
func generatePVCName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("stan-%s-vol", eventBus.Name)
}
