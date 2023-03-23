package sensor

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/argoproj/argo-events/common"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	stanbase "github.com/argoproj/argo-events/eventbus/stan/base"
	"go.uber.org/zap"
)

type SensorSTAN struct {
	*stanbase.STAN
	sensorName string
}

func NewSensorSTAN(url, clusterID, sensorName string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) *SensorSTAN {
	return &SensorSTAN{
		stanbase.NewSTAN(url, clusterID, auth, logger),
		sensorName,
	}
}

func (n *SensorSTAN) Initialize() error {
	return nil
}

func (n *SensorSTAN) Connect(ctx context.Context, triggerName string, dependencyExpression string, deps []eventbuscommon.Dependency, atLeastOnce bool) (eventbuscommon.TriggerConnection, error) {
	// Generate clientID with hash code
	hashKey := fmt.Sprintf("%s-%s-%s", n.sensorName, triggerName, dependencyExpression)
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(100)))
	hashVal := common.Hasher(hashKey)
	clientID := fmt.Sprintf("client-%v-%v", hashVal, randomNum.Int64())

	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return NewSTANTriggerConn(conn, n.sensorName, triggerName, dependencyExpression, deps), nil
}
