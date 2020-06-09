package hdfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestValidateEventSource(t *testing.T) {
	listener := &EventListener{
		Logger: common.NewArgoEventsLogger(),
	}

	valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
		Id:    "1",
		Name:  "hdfs",
		Value: nil,
		Type:  "sq",
	})
	assert.Equal(t, false, valid.IsValid)
	assert.Equal(t, common.ErrEventSourceTypeMismatch("hdfs"), valid.Reason)

	content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", gateways.EventSourceDir, "hdfs.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.HDFS)

	for name, value := range eventSource.Spec.HDFS {
		fmt.Println(name)
		content, err := yaml.Marshal(value)
		assert.Nil(t, err)
		valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
			Id:    "1",
			Name:  "hdfs",
			Value: content,
			Type:  "hdfs",
		})
		fmt.Println(valid.Reason)
		assert.Equal(t, true, valid.IsValid)
	}
}
