package common

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSetObjectMeta(t *testing.T) {
	convey.Convey("Given an object, set meta", t, func() {
		groupVersionKind := schema.GroupVersionKind{
			Group:   "grp",
			Version: "ver",
			Kind:    "kind",
		}
		ctx := ChildResourceContext{
			SchemaGroupVersionKind:            groupVersionKind,
			LabelOwnerName:                    "foo",
			LabelKeyOwnerControllerInstanceID: "id",
			AnnotationOwnerResourceHashName:   "hash",
			InstanceID:                        "ID",
		}
		owner := corev1.Pod{}
		pod := corev1.Pod{}
		ref := metav1.NewControllerRef(&owner, groupVersionKind)

		err := ctx.SetObjectMeta(&owner, &pod)
		convey.So(err, convey.ShouldBeEmpty)
		convey.So(pod.Labels["foo"], convey.ShouldEqual, "")
		convey.So(pod.Labels["id"], convey.ShouldEqual, "ID")
		convey.So(pod.Annotations, convey.ShouldContainKey, "hash")
		convey.So(pod.Name, convey.ShouldEqual, "")
		convey.So(pod.GenerateName, convey.ShouldEqual, "")
		convey.So(pod.OwnerReferences, convey.ShouldContain, *ref)
	})
}
