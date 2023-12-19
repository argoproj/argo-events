package alibabacloudmns

import (
	"context"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestValidateEventSource(t *testing.T) {
	listener := &EventListener{
		MNSEventSource: v1alpha1.MNSEventSource{
			Endpoint: "http://123456789.mns.cn-beijing.aliyuncs.com",
		},
	}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "must specify queue name", err.Error())

	listener = &EventListener{
		MNSEventSource: v1alpha1.MNSEventSource{
			Queue: "test-queue",
		},
	}
	err = listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "must specify a endpoint", err.Error())
}
