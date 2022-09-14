package eventsource

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/codefresh"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
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

	cfClient *codefresh.Client
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, eventSourceImage string, logger *zap.SugaredLogger, cfClient *codefresh.Client) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, eventSourceImage: eventSourceImage, logger: logger, cfClient: cfClient}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	eventSource := &v1alpha1.EventSource{}
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
		r.cfClient.ReportError(errors.Wrap(reconcileErr, "reconcile error"), codefresh.ErrorContext{
			ObjectMeta: eventSource.ObjectMeta,
			TypeMeta:   eventSource.TypeMeta,
		})
	}
	if r.needsUpdate(eventSource, esCopy) {
		if err := r.client.Update(ctx, esCopy); err != nil {
			return reconcile.Result{}, err
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
			"controller":                "eventsource-controller",
			common.LabelEventSourceName: eventSource.Name,
			common.LabelOwnerName:       eventSource.Name,
		},
	}
	return Reconcile(r.client, args, log)
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.EventSource) bool {
	if old == nil {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Finalizers, new.Finalizers) {
		return true
	}
	return false
}
