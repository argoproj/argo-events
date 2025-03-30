package eventsource

import (
	"context"

	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventsource-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client controllerClient.Client
	scheme *runtime.Scheme

	eventSourceImage string
	logger           *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client controllerClient.Client, scheme *runtime.Scheme, eventSourceImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, eventSourceImage: eventSourceImage, logger: logger}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	eventSource := &aev1.EventSource{}
	if err := r.client.Get(ctx, req.NamespacedName, eventSource); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: eventsource not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get eventsource ctl", "request", req, "error", err)
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", eventSource.Namespace).With("eventSource", eventSource.Name)
	ctx = logging.WithLogger(ctx, log)
	esCopy := eventSource.DeepCopy()
	reconcileErr := r.reconcile(ctx, esCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}

	// Update the finalizers
	// We need to always do this to ensure that the field ownership is set correctly,
	// during the migration from client-side to server-side apply. Otherewise, users
	// may end up in a state where the finalizer cannot be removed automatically.
	patch := &aev1.EventSource{
		Status: esCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          eventSource.Name,
			Namespace:     eventSource.Namespace,
			Finalizers:    esCopy.Finalizers,
			ManagedFields: nil,
		},
		TypeMeta: eventSource.TypeMeta,
	}
	if len(patch.Finalizers) == 0 {
		patch.Finalizers = nil
	}
	if err := r.client.Patch(
		ctx,
		patch,
		controllerClient.Apply,
		controllerClient.ForceOwnership,
		controllerClient.FieldOwner("argo-events"),
	); err != nil {
		return reconcile.Result{}, err
	}

	// Update the status
	statusPatch := &aev1.EventSource{
		Status: esCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          eventSource.Name,
			Namespace:     eventSource.Namespace,
			ManagedFields: nil,
		},
		TypeMeta: eventSource.TypeMeta,
	}
	if err := r.client.Status().Patch(
		ctx,
		statusPatch,
		controllerClient.Apply,
		controllerClient.ForceOwnership,
		controllerClient.FieldOwner("argo-events"),
	); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventSource *aev1.EventSource) error {
	log := logging.FromContext(ctx)
	if !eventSource.DeletionTimestamp.IsZero() {
		log.Info("deleting eventsource")
		if controllerutil.ContainsFinalizer(eventSource, finalizerName) {
			// Finalizer logic should be added here.
			controllerutil.RemoveFinalizer(eventSource, finalizerName)
		}
		return nil
	}
	controllerutil.AddFinalizer(eventSource, finalizerName)

	eventSource.Status.InitConditions()
	if err := ValidateEventSource(eventSource); err != nil {
		log.Errorw("validation error", zap.Error(err))
		return err
	}
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
