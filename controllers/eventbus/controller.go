package eventbus

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/controllers"
	"github.com/argoproj/argo-events/controllers/eventbus/installer"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventbus-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client     client.Client
	kubeClient kubernetes.Interface
	scheme     *runtime.Scheme

	config *controllers.GlobalConfig
	logger *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, kubeClient kubernetes.Interface, scheme *runtime.Scheme, config *controllers.GlobalConfig, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, config: config, kubeClient: kubeClient, logger: logger}
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
	ctx = logging.WithLogger(ctx, log)
	busCopy := eventBus.DeepCopy()
	reconcileErr := r.reconcile(ctx, busCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}

	// client.Update() mutates busCopy so we need to make an
	// additional copy in order to update the status.
	statusCopy := busCopy.DeepCopy()

	if r.needsUpdate(eventBus, busCopy) {
		// Use a DeepCopy to update, because it will be mutated afterwards, with empty Status.
		if err := r.client.Update(ctx, busCopy.DeepCopy()); err != nil {
			return reconcile.Result{}, err
		}
	}
	if err := r.client.Status().Update(ctx, statusCopy); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventBus *v1alpha1.EventBus) error {
	log := logging.FromContext(ctx)
	if !eventBus.DeletionTimestamp.IsZero() {
		log.Info("deleting eventbus")
		if controllerutil.ContainsFinalizer(eventBus, finalizerName) {
			// Finalizer logic should be added here.
			if err := installer.Uninstall(ctx, eventBus, r.client, r.kubeClient, r.config, log); err != nil {
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
		eventBus.Status.MarkNotConfigured("InvalidSpec", err.Error())
		return err
	} else {
		eventBus.Status.MarkConfigured()
	}
	return installer.Install(ctx, eventBus, r.client, r.kubeClient, r.config, log)
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
