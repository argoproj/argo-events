package util

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestRetryableKubeAPIError(t *testing.T) {
	errUnAuth := errors.NewUnauthorized("reason")
	errNotFound := errors.NewNotFound(aev1.Resource("sensor"), "hello")
	errForbidden := errors.NewForbidden(aev1.Resource("sensor"), "hello", nil)
	errInvalid := errors.NewInvalid(aev1.GroupKind("core/data"), "hello", nil)
	errMethodNotSupported := errors.NewMethodNotSupported(aev1.Resource("sensor"), "action")

	assert.True(t, IsRetryableKubeAPIError(errUnAuth))
	assert.False(t, IsRetryableKubeAPIError(errNotFound))
	assert.False(t, IsRetryableKubeAPIError(errForbidden))
	assert.False(t, IsRetryableKubeAPIError(errInvalid))
	assert.False(t, IsRetryableKubeAPIError(errMethodNotSupported))
}

func TestConnect(t *testing.T) {
	err := DoWithRetry(nil, func() error {
		return fmt.Errorf("new error")
	})
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "new error"))

	err = DoWithRetry(nil, func() error {
		return nil
	})
	assert.Nil(t, err)
}

func TestConnectDurationString(t *testing.T) {
	start := time.Now()
	count := 2
	err := DoWithRetry(nil, func() error {
		if count == 0 {
			return nil
		} else {
			count--
			return fmt.Errorf("new error")
		}
	})
	end := time.Now()
	elapsed := end.Sub(start)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, elapsed >= 2*time.Second)
}

func TestConnectRetry(t *testing.T) {
	factor := aev1.NewAmount("1.0")
	jitter := aev1.NewAmount("1")
	duration := aev1.FromInt64(1000000000)
	backoff := aev1.Backoff{
		Duration: &duration,
		Factor:   &factor,
		Jitter:   &jitter,
		Steps:    5,
	}
	count := 2
	start := time.Now()
	err := DoWithRetry(&backoff, func() error {
		if count == 0 {
			return nil
		} else {
			count--
			return fmt.Errorf("new error")
		}
	})
	end := time.Now()
	elapsed := end.Sub(start)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, elapsed >= 2*time.Second)
}

func TestRetryFailure(t *testing.T) {
	factor := aev1.NewAmount("1.0")
	jitter := aev1.NewAmount("1")
	duration := aev1.FromString("1s")
	backoff := aev1.Backoff{
		Duration: &duration,
		Factor:   &factor,
		Jitter:   &jitter,
		Steps:    2,
	}
	err := DoWithRetry(&backoff, func() error {
		return fmt.Errorf("this is an error")
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "after retries")
	assert.Contains(t, err.Error(), "this is an error")
}

func TestConvert2WaitBackoff(t *testing.T) {
	factor := aev1.NewAmount("1.0")
	jitter := aev1.NewAmount("1")
	duration := aev1.FromString("1s")
	backoff := aev1.Backoff{
		Duration: &duration,
		Factor:   &factor,
		Jitter:   &jitter,
		Steps:    2,
	}
	waitBackoff, err := Convert2WaitBackoff(&backoff)
	assert.NoError(t, err)
	assert.Equal(t, wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.0,
		Jitter:   1.0,
		Steps:    2,
	}, *waitBackoff)
}
