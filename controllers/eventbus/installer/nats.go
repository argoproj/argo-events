package installer

import (
	"context"
	"fmt"
	"strconv"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"

	"github.com/go-logr/logr"
)

const (
	clientPort  = int32(4222)
	clusterPort = int32(6222)
	monitorPort = int32(8222)
)

// natsInstaller is used create a NATS installation.
type natsInstaller struct {
	client   client.Client
	eventBus *v1alpha1.EventBus
	image    string
	labels   map[string]string
	logger   logr.Logger
}

// NewNATSInstaller returns a new NATS installer
func NewNATSInstaller(client client.Client, eventBus *v1alpha1.EventBus, image string, labels map[string]string, logger logr.Logger) Installer {
	return &natsInstaller{
		client:   client,
		eventBus: eventBus,
		image:    image,
		labels:   labels,
		logger:   logger,
	}
}

// Install creats a StatefulSet and a Service for NATS
func (i *natsInstaller) Install() (*v1alpha1.BusConfig, error) {
	log := i.logger
	ctx := context.Background()
	svc, err := i.getService(ctx)
	if err != nil && !errors.IsNotFound(err) {
		i.eventBus.Status.MarkServiceNotCreated("GetServiceFailed", "Get existing service failed")
		log.Error(err, "error getting existing service")
		return nil, err
	}
	expectedSvc, err := i.makeService()
	if err != nil {
		i.eventBus.Status.MarkServiceNotCreated("MakeServiceFailed", "Failed to build a service spec")
		log.Error(err, "error building service spec")
		return nil, err
	}
	if svc != nil {
		// TODO: potential issue here - if service spec is updated manually, reconciler will not change it back.
		// Revisit it later to see if it is needed to compare the spec.
		if svc.Annotations != nil && svc.Annotations[common.AnnotationResourceSpecHash] != expectedSvc.Annotations[common.AnnotationResourceSpecHash] {
			svc.Spec = expectedSvc.Spec
			svc.Annotations[common.AnnotationResourceSpecHash] = expectedSvc.Annotations[common.AnnotationResourceSpecHash]
			err = i.client.Update(ctx, svc)
			if err != nil {
				i.eventBus.Status.MarkServiceNotCreated("UpdateServiceFailed", "Failed to update existing service")
				log.Error(err, "error updating existing service")
				return nil, err
			}
			log.Info("service is updated", "serviceName", expectedSvc.Name)
		}
	} else {
		err = i.client.Create(ctx, expectedSvc)
		if err != nil {
			i.eventBus.Status.MarkServiceNotCreated("CreateFailed", "Failed to create service")
			log.Error(err, "error creating a service")
			return nil, err
		}
		log.Info("service is created", "serviceName", expectedSvc.Name)
	}
	i.eventBus.Status.MarkServiceCreated("Succeeded", "Succeed to sync the service")

	ss, err := i.getStatefulSet(ctx)
	if err != nil && !errors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStatefulSetFailed", "Failed to get existing statefulset")
		log.Error(err, "error getting existing statefulset")
		return nil, err
	}
	expectedSs, err := i.makeStatefulSet()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("MakeStatefulSetFailed", "Failed to build a statefulset spec")
		log.Error(err, "error building statefulset spec")
		return nil, err
	}
	if ss != nil {
		// TODO: Potential issue here - if statefulset spec is updated manually, reconciler will not change it back.
		// Revisit it later to see if it is needed to compare the spec.
		if ss.Annotations != nil && ss.Annotations[common.AnnotationResourceSpecHash] != expectedSs.Annotations[common.AnnotationResourceSpecHash] {
			ss.Spec = expectedSs.Spec
			ss.Annotations[common.AnnotationResourceSpecHash] = expectedSs.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, ss)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateStatefulSetFailed", "Failed to update existing statefulset")
				log.Error(err, "error updating statefulset")
				return nil, err
			}
			log.Info("statefulset is updated", "statefulsetName", ss.Name)
		}
	} else {
		err := i.client.Create(ctx, expectedSs)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateStatefulSetFailed", "Failed to create a statefulset")
			log.Error(err, "error creating a statefulset")
			return nil, err
		}
		log.Info("statefulset is created", "statefulsetName", expectedSs.Name)
	}
	i.eventBus.Status.MarkDeployed("Succeeded", "StatefulSet is synced")
	i.eventBus.Status.MarkConfigured()
	return &v1alpha1.BusConfig{
		NATS: &v1alpha1.NATSConfig{
			URL: fmt.Sprintf("nats://%s:%s", generateServiceName(i.eventBus), strconv.Itoa(int(clientPort))),
		},
	}, nil
}

// makeService builds a Service for NATS
func (i *natsInstaller) makeService() (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateServiceName(i.eventBus),
			Namespace: i.eventBus.Namespace,
			Labels:    i.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "client", Port: clientPort},
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

// makeStatefulSet builds a StatefulSet for NATS
func (i *natsInstaller) makeStatefulSet() (*appv1.StatefulSet, error) {
	ss := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      fmt.Sprintf("eventbus-nats-%s", i.eventBus.Name),
			Labels:    i.labels,
		},
		Spec: i.makeStatefulSetSpec(),
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, ss, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return ss, nil
}

func (i *natsInstaller) makeStatefulSetSpec() appv1.StatefulSetSpec {
	size := int32(i.eventBus.Spec.NATS.Native.Size)
	if size == 0 {
		size = 1
	}
	return appv1.StatefulSetSpec{
		Replicas:    &size,
		ServiceName: generateServiceName(i.eventBus),
		Selector: &metav1.LabelSelector{
			MatchLabels: i.labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: i.labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nats",
						Image: i.image,
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: clientPort},
							{Name: "cluster", ContainerPort: clusterPort},
							{Name: "monitor", ContainerPort: monitorPort},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(monitorPort)),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
					},
				},
			},
		},
	}
}

func (i *natsInstaller) getLabelSelector() labels.Selector {
	return labels.SelectorFromSet(i.labels)
}

func (i *natsInstaller) getService(ctx context.Context) (*corev1.Service, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	sl := &corev1.ServiceList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: i.getLabelSelector(),
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, i.eventBus) {
			return &svc, nil
		}
	}
	return nil, errors.NewNotFound(schema.GroupResource{}, "")
}

func (i *natsInstaller) getStatefulSet(ctx context.Context) (*appv1.StatefulSet, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	ssl := &appv1.StatefulSetList{}
	err := i.client.List(ctx, ssl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: i.getLabelSelector(),
	})
	if err != nil {
		return nil, err
	}
	for _, ss := range ssl.Items {
		if metav1.IsControlledBy(&ss, i.eventBus) {
			return &ss, nil
		}
	}
	return nil, errors.NewNotFound(schema.GroupResource{}, "")
}

func generateServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-svc", eventBus.Name)
}

func generateStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-nats-%s", eventBus.Name)
}
