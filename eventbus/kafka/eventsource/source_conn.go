package eventsource

import (
	"context"

	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
)

type KafkaSourceConnection struct {
	*base.KafkaConnection
}

func (c *KafkaSourceConnection) Publish(ctx context.Context, msg common.Message) error {
	return nil
}
