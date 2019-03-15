package common

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func getFakePodSharedIndexInformer(clientset kubernetes.Interface) cache.SharedIndexInformer {
	// NewListWatchFromClient doesn't work with fake client.
	// ref: https://github.com/kubernetes/client-go/issues/352
	return cache.NewSharedIndexInformer(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.CoreV1().Pods("").List(options)
		},
		WatchFunc: clientset.CoreV1().Pods("").Watch,
	}, &corev1.Pod{}, 1*time.Second, cache.Indexers{})
}

func getInformerFactory(clientset kubernetes.Interface, queue workqueue.RateLimitingInterface, done chan struct{}) *ArgoEventInformerFactory {
	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	ownerInformer := getFakePodSharedIndexInformer(clientset)
	go ownerInformer.Run(done)
	return &ArgoEventInformerFactory{
		OwnerGroupVersionKind: schema.GroupVersionKind{Version: "v1", Kind: "Pod"},
		OwnerInformer:         ownerInformer,
		SharedInformerFactory: informerFactory,
		Queue: queue,
	}
}

func getCommonPodSpec() corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "whalesay",
				Image: "docker/whalesay:latest",
			},
		},
	}
}

func getPod(owner *corev1.Pod) *corev1.Pod {
	var ownerReferneces = []metav1.OwnerReference{}
	if owner != nil {
		ownerReferneces = append(ownerReferneces, *metav1.NewControllerRef(owner, owner.GroupVersionKind()))
	}
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("pod-%d", rand.Uint32()),
			OwnerReferences: ownerReferneces,
		},
		Spec: getCommonPodSpec(),
	}
}

func getService(owner *corev1.Pod) *corev1.Service {
	var ownerReferneces = []metav1.OwnerReference{}
	if owner != nil {
		ownerReferneces = append(ownerReferneces, *metav1.NewControllerRef(owner, owner.GroupVersionKind()))
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("svc-%d", rand.Uint32()),
			OwnerReferences: ownerReferneces,
		},
		Spec: corev1.ServiceSpec{},
	}
}

func TestNewPodInformer(t *testing.T) {
	done := make(chan struct{})
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	namespace := "namespace"
	clientset := fake.NewSimpleClientset()
	factory := getInformerFactory(clientset, queue, done)
	convey.Convey("Given an informer factory", t, func() {
		convey.Convey("Get a new gateway pod informer and make sure its not nil", func() {
			podInformer := factory.NewPodInformer()
			convey.So(podInformer, convey.ShouldNotBeNil)

			convey.Convey("Handle event", func() {
				go podInformer.Informer().Run(done)
				ownerPod := getPod(nil)
				ownerPod, err := clientset.CoreV1().Pods(namespace).Create(ownerPod)
				convey.So(err, convey.ShouldBeNil)
				cache.WaitForCacheSync(done, factory.OwnerInformer.HasSynced)

				convey.Convey("Not enqueue owner key on creation", func() {
					pod := getPod(ownerPod)
					pod, err := clientset.CoreV1().Pods(namespace).Create(pod)
					convey.So(err, convey.ShouldBeNil)
					cache.WaitForCacheSync(done, podInformer.Informer().HasSynced)

					convey.So(queue.Len(), convey.ShouldEqual, 0)

					convey.Convey("Not enqueue owner key on update", func() {
						pod.Labels = map[string]string{"foo": "bar"}
						_, err = clientset.CoreV1().Pods(namespace).Update(pod)
						convey.So(err, convey.ShouldBeNil)
						cache.WaitForCacheSync(done, podInformer.Informer().HasSynced)

						convey.So(queue.Len(), convey.ShouldEqual, 0)
					})

					convey.Convey("Enqueue owner key on deletion", func() {
						err := clientset.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})
						convey.So(err, convey.ShouldBeNil)
						cache.WaitForCacheSync(done, podInformer.Informer().HasSynced)

						convey.So(queue.Len(), convey.ShouldEqual, 1)
						key, _ := queue.Get()
						queue.Done(key)
						convey.So(key, convey.ShouldEqual, fmt.Sprintf("%s/%s", namespace, ownerPod.Name))
					})
				})
			})
		})
	})
}

func TestNewServiceInformer(t *testing.T) {
	done := make(chan struct{})
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	namespace := "namespace"
	clientset := fake.NewSimpleClientset()
	factory := getInformerFactory(clientset, queue, done)
	convey.Convey("Given an informer factory", t, func() {
		convey.Convey("Get a new gateway service informer and make sure its not nil", func() {
			svcInformer := factory.NewServiceInformer()
			convey.So(svcInformer, convey.ShouldNotBeNil)

			convey.Convey("Handle event", func() {
				go svcInformer.Informer().Run(done)
				ownerPod := getPod(nil)
				ownerPod, err := clientset.CoreV1().Pods(namespace).Create(ownerPod)
				convey.So(err, convey.ShouldBeNil)
				cache.WaitForCacheSync(done, factory.OwnerInformer.HasSynced)

				convey.Convey("Not enqueue owner key on creation", func() {
					service := getService(ownerPod)
					service, err := clientset.CoreV1().Services(namespace).Create(service)
					convey.So(err, convey.ShouldBeNil)
					cache.WaitForCacheSync(done, svcInformer.Informer().HasSynced)
					convey.So(queue.Len(), convey.ShouldEqual, 0)

					convey.Convey("Not enqueue owner key on update", func() {
						service.Labels = map[string]string{"foo": "bar"}
						service, err = clientset.CoreV1().Services(namespace).Update(service)
						convey.So(err, convey.ShouldBeNil)
						cache.WaitForCacheSync(done, svcInformer.Informer().HasSynced)
						convey.So(queue.Len(), convey.ShouldEqual, 0)
					})

					convey.Convey("Enqueue owner key on deletion", func() {
						err := clientset.CoreV1().Services(namespace).Delete(service.Name, &metav1.DeleteOptions{})
						convey.So(err, convey.ShouldBeNil)
						cache.WaitForCacheSync(done, svcInformer.Informer().HasSynced)

						convey.So(queue.Len(), convey.ShouldEqual, 1)
						key, _ := queue.Get()
						queue.Done(key)
						convey.So(key, convey.ShouldEqual, fmt.Sprintf("%s/%s", namespace, ownerPod.Name))
					})
				})
			})
		})
	})
}
