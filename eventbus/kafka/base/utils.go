package base

import (
	"fmt"
	"time"
)

func EventKey(source string, subject string) string {
	return fmt.Sprintf("%s.%s", source, subject)
}

// Batch returns a read only channel that receives values from the
// input channel batched together into a slice. A value is sent to
// the output channel when the slice reaches n elements, or d time
// has elapsed, whichever happens first. Ordering is maintained.
func Batch[T any](n int, d time.Duration, in <-chan T) <-chan []T {
	out := make(chan []T, 1)

	go func() {
		batch := []T{}
		timer := time.NewTimer(d)
		timer.Stop()

		defer close(out)
		defer timer.Stop()

		for {
			select {
			case item, ok := <-in:
				if !ok {
					return
				}
				if len(batch) == 0 {
					timer.Reset(d)
				}
				if batch = append(batch, item); len(batch) == n {
					timer.Stop()
					out <- batch
					batch = nil
				}
			case <-timer.C:
				if len(batch) > 0 {
					out <- batch
					batch = nil
				}
			}
		}
	}()

	return out
}
