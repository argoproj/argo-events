package eventsource

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventsource-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client client.Client
	scheme *runtime.Scheme

	eventSourceImage string
	logger           *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, eventSourceImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, eventSourceImage: eventSourceImage, logger: logger}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	eventSource := &v1alpha1.EventSource{}
	if err := r.client.Get(ctx, req.NamespacedName, eventSource); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: eventsource not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get eventsource", "request", req, zap.Error(err))
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", eventSource.Namespace).With("eventSource", eventSource.Name)
	ctx = logging.WithLogger(ctx, log)
	esCopy := eventSource.DeepCopy()
	reconcileErr := r.reconcile(ctx, esCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}
	esCopy.Status.LastUpdated = metav1.Now()
	if !equality.Semantic.DeepEqual(eventSource.Finalizers, esCopy.Finalizers) {
		patchYaml := "metadata:\n  finalizers: [" + strings.Join(esCopy.Finalizers, ",") + "]"
		patchJson, _ := yaml.YAMLToJSON([]byte(patchYaml))
		if err := r.client.Patch(ctx, eventSource, client.RawPatch(types.MergePatchType, []byte(patchJson))); err != nil {
			return ctrl.Result{}, err
		}
	}
	if err := r.client.Status().Update(ctx, esCopy); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventSource *v1alpha1.EventSource) error {
	log := logging.FromContext(ctx)
	if !eventSource.DeletionTimestamp.IsZero() {
		log.Info("Deleting eventsource")
		if controllerutil.ContainsFinalizer(eventSource, finalizerName) {
			// We don't add finalizer anymore, keep the removing logic for backward compatibility
			controllerutil.RemoveFinalizer(eventSource, finalizerName)
		}
		return nil
	}

	eventSource.Status.InitConditions()
	eventSource.Status.SetObservedGeneration(eventSource.Generation)
	if err := ValidateEventSource(eventSource); err != nil {
		log.Errorw("Validation error", zap.Error(err))
		eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", err.Error())
		return err
	}
	eventSource.Status.MarkSourcesProvided()
	args := &AdaptorArgs{
		Image:       r.eventSourceImage,
		EventSource: eventSource,
		Labels: map[string]string{
			"controller":                  "eventsource-controller",
			v1alpha1.LabelEventSourceName: eventSource.Name,
			v1alpha1.LabelOwnerName:       eventSource.Name,
		},
	}
	return Reconcile(r.client, args, log)
}
