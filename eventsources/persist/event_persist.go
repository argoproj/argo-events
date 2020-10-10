package persist

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sync"

	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

type EventPersist interface {
	Save(event *Event) error
	Get(key string) (*Event, error)
	IsEnabled() bool
}

type Event struct {
	EventKey     string
	EventPayload string
}

type ConfigMapPersist struct {
	kubeClient       kubernetes.Interface
	name             string
	namespace        string
	createIfNotExist bool
	lock             sync.Mutex
}

func createConfigmap(client kubernetes.Interface, name, namespace string) (*v1.ConfigMap, error) {
	cm := v1.ConfigMap{}
	cm.Name = name
	cm.Namespace = namespace
	return client.CoreV1().ConfigMaps(namespace).Create(&cm)
}

func NewConfigMapPersist(client kubernetes.Interface, configmap *v1alpha1.ConfigMapPersistence, namespace string) (EventPersist, error) {
	if configmap == nil {
		return nil, fmt.Errorf("persistence configuration is nil")
	}
	_, err := client.CoreV1().ConfigMaps(namespace).Get(configmap.Name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) && configmap.CreateIfNotExist {
			_, err = createConfigmap(client, configmap.Name, namespace)
			if err != nil {
				if !apierr.IsAlreadyExists(err) {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	cmp := ConfigMapPersist{
		kubeClient:       client,
		name:             configmap.Name,
		namespace:        namespace,
		createIfNotExist: configmap.CreateIfNotExist,
		lock:             sync.Mutex{},
	}
	return &cmp, nil
}

func (cmp *ConfigMapPersist) IsEnabled() bool {
	return true
}

func (cmp *ConfigMapPersist) Save(event *Event) error {
	if event == nil {
		return errors.Errorf("event object is nil")
	}
	cmp.lock.Lock()
	defer cmp.lock.Unlock()
	err := wait.ExponentialBackoff(retry.DefaultBackoff, func() (done bool, err error) {
		cm, err := cmp.kubeClient.CoreV1().ConfigMaps(cmp.namespace).Get(cmp.name, metav1.GetOptions{})
		if err != nil {
			if apierr.IsNotFound(err) && cmp.createIfNotExist {
				cm, err = createConfigmap(cmp.kubeClient, cmp.name, cmp.namespace)
				if err != nil {
					return err == nil, err
				}
			} else {
				return err == nil, err
			}
		}

		if len(cm.Data) == 0 {
			cm.Data = make(map[string]string)
		}

		cm.Data[event.EventKey] = event.EventPayload
		_, err = cmp.kubeClient.CoreV1().ConfigMaps(cmp.namespace).Update(cm)

		return err == nil, err
	})

	if err != nil {
		return err
	}
	return nil
}

func (cmp *ConfigMapPersist) Get(key string) (*Event, error) {
	cmp.lock.Lock()
	defer cmp.lock.Unlock()
	cm, err := cmp.kubeClient.CoreV1().ConfigMaps(cmp.namespace).Get(cmp.name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	payload, exist := cm.Data[key]
	if !exist {
		return nil, nil
	}
	return &Event{EventKey: key, EventPayload: payload}, nil
}

type NullPersistence struct {
}

func (n *NullPersistence) Save(event *Event) error {
	return nil
}

func (n *NullPersistence) Get(key string) (*Event, error) {
	return nil, nil
}

func (cmp *NullPersistence) IsEnabled() bool {
	return false
}
