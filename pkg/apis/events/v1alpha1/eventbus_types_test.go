package v1alpha1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
)

func TestEventBusStatusIsReady(t *testing.T) {
	tests := []struct {
		name   string
		s      *EventBusStatus
		expect bool
	}{
		{
			name: "uninitialized",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				return s
			}(),
			expect: false,
		},
		{
			name: "initialized",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				return s
			}(),
			expect: false,
		},
		{
			name: "mark deployed",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkDeployed("test", "test")
				return s
			}(),
			expect: false,
		},
		{
			name: "mark service created, deployed and configured",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkDeployed("test", "test")
				s.MarkConfigured()
				return s
			}(),
			expect: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.IsReady()
			if diff := cmp.Diff(test.expect, got); diff != "" {
				t.Errorf("%s: unexpected condition (-expect, +got) = %v", test.name, diff)
			}
		})
	}
}

func TestEventBusStatusGetCondition(t *testing.T) {
	tests := []struct {
		name       string
		s          *EventBusStatus
		qCondition ConditionType
		expect     *Condition
	}{
		{
			name:       "uninitialized",
			s:          &EventBusStatus{},
			qCondition: EventBusConditionDeployed,
			expect:     nil,
		},
		{
			name: "initialized",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				return s
			}(),
			qCondition: EventBusConditionDeployed,
			expect: &Condition{
				Status: corev1.ConditionUnknown,
				Type:   EventBusConditionDeployed,
			},
		},
		{
			name: "mark deployed",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkDeployed("test", "test")
				return s
			}(),
			qCondition: EventBusConditionDeployed,
			expect: &Condition{
				Status:  corev1.ConditionTrue,
				Type:    EventBusConditionDeployed,
				Reason:  "test",
				Message: "test",
			},
		},
		{
			name: "mark deploy failed",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkDeployFailed("test", "test")
				return s
			}(),
			qCondition: EventBusConditionDeployed,
			expect: &Condition{
				Status:  corev1.ConditionFalse,
				Type:    EventBusConditionDeployed,
				Reason:  "test",
				Message: "test",
			},
		},
		{
			name: "mark configured",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkConfigured()
				return s
			}(),
			qCondition: EventBusConditionConfigured,
			expect: &Condition{
				Status:  corev1.ConditionTrue,
				Type:    EventBusConditionConfigured,
				Reason:  "",
				Message: "",
			},
		},
		{
			name: "mark not configured",
			s: func() *EventBusStatus {
				s := &EventBusStatus{}
				s.InitConditions()
				s.MarkNotConfigured("test", "test")
				return s
			}(),
			qCondition: EventBusConditionConfigured,
			expect: &Condition{
				Status:  corev1.ConditionFalse,
				Type:    EventBusConditionConfigured,
				Reason:  "test",
				Message: "test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.Status.GetCondition(test.qCondition)
			ignoreFields := cmpopts.IgnoreFields(Condition{},
				"LastTransitionTime")
			if diff := cmp.Diff(test.expect, got, ignoreFields); diff != "" {
				t.Errorf("unexpected condition (-expect, +got) = %v", diff)
			}
		})
	}
}
