package sensor

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTrackFetchErrorThrottlesRepeatedError guards against a regression where pullSubscribe
// advanced its "last logged at" timestamp on every erroring loop iteration, not only when a
// log line was actually emitted. In a tight retry loop (the same fetch error recurring many
// times a second) that made the elapsed time since the last log always ~0, so the 10-second
// throttle window could never be reached: a sustained, unchanging fetch error logged exactly
// once and then went silent forever.
func TestTrackFetchErrorThrottlesRepeatedError(t *testing.T) {
	fatalErr := errors.New("nats: invalid subscription")
	start := time.Now()

	var state fetchErrState
	var loggedCount int

	// simulate a tight retry loop hammering the same fatal error every 5ms for 12 simulated
	// seconds. A correct throttle must still fire a second time once 10s of real elapsed
	// time has passed, even though consecutive iterations are only milliseconds apart.
	const tick = 5 * time.Millisecond
	iterations := int(12*time.Second/tick) + 1
	for i := 0; i < iterations; i++ {
		var logged bool
		state, logged = trackFetchError(state, fatalErr, start.Add(time.Duration(i)*tick))
		if logged {
			loggedCount++
		}
	}
	require.Equal(t, 2, loggedCount, "a sustained error must log once at onset and again after the 10s throttle window elapses, even under a tight retry loop")

	// a genuinely different error message logs immediately, regardless of timing
	otherErr := errors.New("nats: bad subscription")
	_, logged := trackFetchError(state, otherErr, start.Add(12*time.Second+2*time.Millisecond))
	require.True(t, logged, "a change in error message should log immediately")
}
