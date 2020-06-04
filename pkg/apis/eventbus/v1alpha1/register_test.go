package v1alpha1

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/eventbus"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResource(t *testing.T) {
	expect := schema.GroupResource{
		Group:    eventbus.Group,
		Resource: "hello",
	}

	got := Resource("hello")

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("unexpected resource (-expects, +got) = %v", diff)
	}
}
