package eventbus

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// Validator validates EventBus object
type Validator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle funcion is the real logic of validation
func (v *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	eb := &v1alpha1.EventBus{}
	err := v.decoder.Decode(req, eb)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	oldObj := &v1alpha1.EventBus{}
	err = v.Client.Get(ctx, types.NamespacedName{Namespace: eb.Namespace, Name: eb.Name}, oldObj)
	if err != nil && apierrors.IsNotFound(err) {
		return admission.Allowed("")
	}
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if oldObj.Spec.NATS != nil {
		if eb.Spec.NATS == nil {
			return admission.Errored(http.StatusBadRequest, errors.New("\"spec.nats\" is not allowed to be removed"))
		}
		if oldObj.Spec.NATS.Native != nil {
			if eb.Spec.NATS.Native == nil {
				return admission.Errored(http.StatusBadRequest, errors.New("\"spec.nats.native\" is not allowed to be removed"))
			}
			ebNative := eb.Spec.NATS.Native
			oldNative := oldObj.Spec.NATS.Native
			if *ebNative.Auth != *oldNative.Auth {
				return admission.Errored(http.StatusBadRequest, errors.New("\"spec.nats.native.auth\" is not allowed to be changed"))
			}
		} else if oldObj.Spec.NATS.Exotic != nil {
			if eb.Spec.NATS.Exotic == nil {
				return admission.Errored(http.StatusBadRequest, errors.New("\"spec.nats.exotic\" is not allowed to be removed"))
			}
		}
	}
	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (v *Validator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
