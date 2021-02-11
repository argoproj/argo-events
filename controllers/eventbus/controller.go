package eventbus

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

	"github.com/argoproj/argo-events/controllers/eventbus/installer"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventbus-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client client.Client
	scheme *runtime.Scheme

	natsStreamingImage string
	natsMetricsImage   string
	logger             *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, natsStreamingImage, natsMetricsImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, natsStreamingImage: natsStreamingImage, natsMetricsImage: natsMetricsImage, logger: logger}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	eventBus := &v1alpha1.EventBus{}
	if err := r.client.Get(ctx, req.NamespacedName, eventBus); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: eventbus not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get eventbus ctl", zap.Any("request", req), zap.Error(err))
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", eventBus.Namespace).With("eventbus", eventBus.Name)
	busCopy := eventBus.DeepCopy()
	reconcileErr := r.reconcile(ctx, busCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}
	if r.needsUpdate(eventBus, busCopy) {
		if err := r.client.Update(ctx, busCopy); err != nil {
			return reconcile.Result{}, err
		}
	}
	if err := r.client.Status().Update(ctx, busCopy); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventBus *v1alpha1.EventBus) error {
	log := r.logger.With("namespace", eventBus.Namespace).With("eventbus", eventBus.Name)
	if !eventBus.DeletionTimestamp.IsZero() {
		log.Info("deleting eventbus")
		if controllerutil.ContainsFinalizer(eventBus, finalizerName) {
			// Finalizer logic should be added here.
			if err := installer.Uninstall(ctx, eventBus, r.client, r.natsStreamingImage, r.natsMetricsImage, log); err != nil {
				log.Errorw("failed to uninstall", zap.Error(err))
				return err
			}
			controllerutil.RemoveFinalizer(eventBus, finalizerName)
		}
		return nil
	}
	controllerutil.AddFinalizer(eventBus, finalizerName)

	eventBus.Status.InitConditions()
	if err := ValidateEventBus(eventBus); err != nil {
		log.Errorw("validation failed", zap.Error(err))
		eventBus.Status.MarkDeployFailed("InvalidSpec", err.Error())
		return err
	}
	return installer.Install(ctx, eventBus, r.client, r.natsStreamingImage, r.natsMetricsImage, log)
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.EventBus) bool {
	if old == nil {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Finalizers, new.Finalizers) {
		return true
	}
	return false
}
