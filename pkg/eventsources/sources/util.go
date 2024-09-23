package sources

import (
	"fmt"
)

// Recover recovers from panics in event sources
func Recover(eventName string) {
	if r := recover(); r != nil {
		fmt.Printf("recovered event source %s from panic. recover: %v", eventName, r)
	}
}
