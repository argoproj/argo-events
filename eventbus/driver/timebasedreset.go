package driver

import (
	"context"
	"time"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cronlib "github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// EventBus drivers use this information to know when a time-based reset should occur
type TimeBasedReset struct {
	lastResetTime time.Time     // previous times that a reset should've been triggered
	channel       chan struct{} // channel for instructing the EventBus driver to do a reset going forward
}

func CreateTimeBasedReset(ctx context.Context, criteria []v1alpha1.ConditionsResetCriteria) TimeBasedReset { // TODO: pointer or no pointer?
	logger := logging.FromContext(ctx)

	reset := TimeBasedReset{
		channel: make(chan struct{}),
	}

	// go through each cron-defined trigger time
	// for each:
	// 1. set up a cron job to fire
	// 2. store previous time cron job would have fired so we can verify no dependencies that NATS is sending were from before that time

	for _, c := range criteria {

		// set up Cron job to handle the next time this will fire
		if c.ByTime == nil {
			continue
		}
		opts := []cronlib.Option{
			cronlib.WithParser(cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)),
			cronlib.WithChain(cronlib.Recover(cronlib.DefaultLogger)),
		}
		if c.ByTime.Timezone != "" {
			location, err := time.LoadLocation(c.ByTime.Timezone)
			if err != nil {
				logger.Errorw("failed to load timezone", zap.Error(err))
				continue
			}
			opts = append(opts, cronlib.WithLocation(location))
		}
		cr := cronlib.New(opts...)
		_, err := cr.AddFunc(c.ByTime.Cron, func() {
			reset.channel <- struct{}{}
		})
		if err != nil {
			logger.Errorw("failed to add cron schedule", zap.Error(err))
			continue
		}
		cr.Start()

		logger.Debugf("just started cron job for ConditionsReset; entries=%v", cr.Entries())
		if len(cr.Entries()) > 0 {
			firstEntry := cr.Entry(1) // there should only be one
			specSchedule, castOk := firstEntry.Schedule.(*cronlib.SpecSchedule)
			if !castOk {
				// TODO
			} else {
				reset.lastResetTime = max(reset.lastResetTime, specSchedule.prev(time.Now()))
			}

		}
	}

	return reset
}
