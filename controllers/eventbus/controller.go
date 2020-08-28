package eventbus

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

func (r *reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	eventBus := &v1alpha1.EventBus{}
	if err := r.client.Get(ctx, req.NamespacedName, eventBus); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: eventbus not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get eventbus ctl", "request", req, "error", err)
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", eventBus.Namespace).With("eventbus", eventBus.Name)
	busCopy := eventBus.DeepCopy()
	reconcileErr := r.reconcile(ctx, busCopy)
	if reconcileErr != nil {
		log.Desugar().Error("reconcile error", zap.Error(reconcileErr))
	}
	if r.needsUpdate(eventBus, busCopy) {
		if err := r.client.Update(ctx, busCopy); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventBus *v1alpha1.EventBus) error {
	log := r.logger.With("namespace", eventBus.Namespace).With("eventbus", eventBus.Name)
	if !eventBus.DeletionTimestamp.IsZero() {
		log.Info("deleting eventbus")
		// Finalizer logic should be added here.
		err := installer.Uninstall(eventBus, r.client, r.natsStreamingImage, r.natsMetricsImage, log)
		if err != nil {
			log.Errorw("failed to uninstall", "error", err)
			return nil
		}
		r.removeFinalizer(eventBus)
		return nil
	}
	r.addFinalizer(eventBus)

	eventBus.Status.InitConditions()
	return installer.Install(eventBus, r.client, r.natsStreamingImage, r.natsMetricsImage, log)
}

func (r *reconciler) addFinalizer(s *v1alpha1.EventBus) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Insert(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) removeFinalizer(s *v1alpha1.EventBus) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Delete(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.EventBus) bool {
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
