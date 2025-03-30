package common

import (
	"reflect"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EventBusConfigStatusChangedPredidicate implements a status change predicate functor.
type EventBusConfigStatusChangedPredicate struct {
	predicate.TypedFuncs[*aev1.EventBus]
}

// Update implements default UpdateEvent filter for validating status change
func (EventBusConfigStatusChangedPredicate) Update(e event.TypedUpdateEvent[*aev1.EventBus]) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	// Get status.config fields using reflection since different objects have different status types
	oldStatus := reflect.ValueOf(e.ObjectOld).Elem().FieldByName("Status")
	newStatus := reflect.ValueOf(e.ObjectNew).Elem().FieldByName("Status")

	if !oldStatus.IsValid() || !newStatus.IsValid() {
		return false
	}

	oldConfig := oldStatus.FieldByName("Config")
	newConfig := newStatus.FieldByName("Config")

	if !oldConfig.IsValid() || !newConfig.IsValid() {
		return false
	}

	return !reflect.DeepEqual(oldConfig.Interface(), newConfig.Interface())
}
