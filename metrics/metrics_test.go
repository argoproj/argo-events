package metrics

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/common/logging"
)

func TestRun(t *testing.T) {
	port := 9090
	m := NewMetrics("test-ns")
	go m.Run(logging.WithLogger(context.Background(), logging.NewArgoEventsLogger()), fmt.Sprintf(":%d", port))
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	assert.Nil(t, err)
	assert.Equal(t, resp.StatusCode, 200)
}
