package common

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func ApplyEventLabels(labels map[string]string, events map[string]*v1alpha1.Event) error {
	count := 0
	for _, val := range events {
		decodedID, err := hex.DecodeString(val.Context.ID)
		if err != nil {
			return errors.Wrap(err, "failed to decode event ID")
		}

		labelName := fmt.Sprintf("events.argoproj.io/event-%d", count)
		labels[labelName] = string(decodedID)

		count++
	}

	return nil
}