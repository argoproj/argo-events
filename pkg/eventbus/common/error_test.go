package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_error(t *testing.T) {
	err := fmt.Errorf("error")
	var ebErr *EventBusError
	assert.False(t, errors.As(err, &ebErr))
	err = fmt.Errorf("err1, %w", err)
	assert.False(t, errors.As(err, &ebErr))
	err = NewEventBusError(err)
	assert.True(t, errors.As(err, &ebErr))
	err = fmt.Errorf("err3, %w", err)
	assert.True(t, errors.As(err, &ebErr))
	err = fmt.Errorf("err4, %w", err)
	assert.True(t, errors.As(err, &ebErr))
	err = fmt.Errorf("err5, %w", err)
	assert.True(t, errors.As(err, &ebErr))
}
