package eventsource

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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

func (r *reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
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
	evCopy := eventSource.DeepCopy()
	reconcileErr := r.reconcile(ctx, evCopy)
	if reconcileErr != nil {
		log.Desugar().Error("reconcile error", zap.Error(reconcileErr))
	}
	if r.needsUpdate(eventSource, evCopy) {
		if err := r.client.Update(ctx, evCopy); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventSource *v1alpha1.EventSource) error {
	log := r.logger.With("namespace", eventSource.Namespace).With("eventSource", eventSource.Name)
	if !eventSource.DeletionTimestamp.IsZero() {
		log.Info("deleting eventsource")
		// Finalizer logic should be added here.
		r.removeFinalizer(eventSource)
		return nil
	}
	r.addFinalizer(eventSource)

	eventSource.Status.InitConditions()
	err := ValidateEventSource(eventSource)
	if err != nil {
		log.Desugar().Error("validation error", zap.Error(err))
		return err
	}
	args := &AdaptorArgs{
		Image:       r.eventSourceImage,
		EventSource: eventSource,
		Labels: map[string]string{
			"controller":                "eventsource-controller",
			common.LabelEventSourceName: eventSource.Name,
		},
	}
	return Reconcile(r.client, args, log)
}

func (r *reconciler) addFinalizer(s *v1alpha1.EventSource) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Insert(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) removeFinalizer(s *v1alpha1.EventSource) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Delete(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.EventSource) bool {
	if old == nil {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Status, new.Status) {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Finalizers, new.Finalizers) {
		return true
	}
	return false
}
