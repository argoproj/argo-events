package eventbus

import (
	"context"

	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/reconciler"
	"github.com/argoproj/argo-events/pkg/reconciler/eventbus/installer"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventbus-controller"

	finalizerName = ControllerName
)

type eventBusReconciler struct {
	client     controllerClient.Client
	kubeClient kubernetes.Interface
	scheme     *runtime.Scheme

	config *reconciler.GlobalConfig
	logger *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client controllerClient.Client, kubeClient kubernetes.Interface, scheme *runtime.Scheme, config *reconciler.GlobalConfig, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &eventBusReconciler{client: client, scheme: scheme, config: config, kubeClient: kubeClient, logger: logger}
}

func (r *eventBusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	// Update the finalizers
	// We need to always do this to ensure that the field ownership is set correctly,
	// during the migration from client-side to server-side apply. Otherewise, users
	// may end up in a state where the finalizer cannot be removed automatically.
	patch := &v1alpha1.EventBus{
		Status: busCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          eventBus.Name,
			Namespace:     eventBus.Namespace,
			Finalizers:    busCopy.Finalizers,
			ManagedFields: nil,
		},
		TypeMeta: eventBus.TypeMeta,
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
	statusPatch := &v1alpha1.EventBus{
		Status: busCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          eventBus.Name,
			Namespace:     eventBus.Namespace,
			ManagedFields: nil,
		},
		TypeMeta: eventBus.TypeMeta,
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
func (r *eventBusReconciler) reconcile(ctx context.Context, eventBus *v1alpha1.EventBus) error {
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
