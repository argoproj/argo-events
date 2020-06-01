package eventbus

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/controllers/eventbus/installer"
	"github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"

	"github.com/go-logr/logr"
)

const (
	// ControllerName is name of the controller
	ControllerName = "eventbus-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client client.Client
	scheme *runtime.Scheme

	images map[common.EventBusType]string
	logger logr.Logger
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, images map[common.EventBusType]string, logger logr.Logger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, images: images, logger: logger}
}

func (r *reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	eventBus := &v1alpha1.EventBus{}
	if err := r.client.Get(ctx, req.NamespacedName, eventBus); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Info("WARNING: eventbus not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Error(err, "unable to get eventbus ctl", "request", req)
		return ctrl.Result{}, err
	}
	log := r.logger.WithValues("namespace", eventBus.Namespace).WithValues("eventbus", eventBus.Name)
	obj := eventBus.DeepCopyObject()
	busCopy, ok := obj.(*v1alpha1.EventBus)
	if !ok {
		return ctrl.Result{}, errors.New("convert error")
	}
	reconcileErr := r.reconcile(ctx, busCopy)
	if reconcileErr != nil {
		log.Error(reconcileErr, "reconcile error")
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
	log := r.logger.WithValues("namespace", eventBus.Namespace).WithValues("eventbus", eventBus.Name)
	if !eventBus.DeletionTimestamp.IsZero() {
		log.Info("deleting eventbus")
		// Finalizer logic should be added here.
		r.removeFinalizer(eventBus)
		return nil
	}
	r.addFinalizer(eventBus)

	eventBus.Status.InitConditions()
	if nats := eventBus.Spec.NATS; nats != nil {
		if nats.Exotic != nil {
			eventBus.Status.Config = v1alpha1.BusConfig{
				NATS: nats.Exotic,
			}
			eventBus.Status.MarkDeployed("Skipped", "Skip deployment because of using exotic config.")
			eventBus.Status.MarkConfigured()
			log.Info("use exotic config")
		} else if nats.Native != nil {
			image, ok := r.images[common.EventBusNATS]
			if !ok {
				return errors.New("nats image not found")
			}
			ins := installer.NewNATSInstaller(r.client, eventBus, image, getLabels(eventBus), log)
			busConfig, err := ins.Install()
			if err != nil {
				log.Error(err, "NATS installation error")
				return err
			}
			eventBus.Status.Config = *busConfig
		}
	} else {
		return errors.New("invalid eventbus spec")
	}
	return nil
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

func getLabels(bus *v1alpha1.EventBus) map[string]string {
	return map[string]string{
		"controller":    ControllerName,
		"eventbus-name": bus.Name,
	}
}
