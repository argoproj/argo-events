package sources

import (
	"fmt"

	"github.com/argoproj/argo-events/common"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Recover recovers from panics in event sources
func Recover(eventName string) {
	if r := recover(); r != nil {
		fmt.Printf("recovered event source %s from panic. recover: %v", eventName, r)
	}
}

// Connect is a general connection helper
func Connect(backoff *wait.Backoff, conn func() error) error {
	if backoff == nil {
		backoff = &common.DefaultRetry
	}
	err := wait.ExponentialBackoff(*backoff, func() (bool, error) {
		if err := conn(); err != nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
