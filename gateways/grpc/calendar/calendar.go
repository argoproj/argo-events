/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
	gwProto "github.com/argoproj/argo-events/gateways/grpc/proto"
	"github.com/ghodss/yaml"
	cronlib "github.com/robfig/cron"
	zlog "github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"os"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence,omitempty" protobuf:"bytes,3,rep,name=recurrence"`
}

type calendar struct {
	// log is json output logger for gateway
	log zlog.Logger
}

func (c *calendar) RunGateway(config *gwProto.GatewayConfig, eventStream gwProto.GatewayExecutor_RunGatewayServer) error {

	var cal *calSchedule
	err := yaml.Unmarshal([]byte(config.Config), &cal)
	if err != nil {
		c.log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse configuration")
		return err
	}

	schedule, err := c.resolveSchedule(cal)
	if err != nil {
		c.log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
		return err
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		c.log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
		return err
	}

	var next Next
	next = func(last time.Time) time.Time {
		nextT := schedule.Next(last)
		nextYear := nextT.Year()
		nextMonth := nextT.Month()
		nextDay := nextT.Day()
		for _, exDate := range exDates {
			// if exDate == nextEvent, then we need to skip this and get the next
			if exDate.Year() == nextYear && exDate.Month() == nextMonth && exDate.Day() == nextDay {
				return next(nextT)
			}
		}
		return nextT
	}

	lastT := time.Now()
calendarConfigRunner:
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		log.Printf("expected next calendar event %s", t)
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				c.log.Error().Err(err).Msg("failed to marshal event")
			} else {
				eventStream.Send(&gwProto.Event{
					Data: payload,
				})
			}
		case <-eventStream.Context().Done():
			break calendarConfigRunner
		}
	}
	return nil
}

func (c *calendar) getEventTimer(next Next) <-chan time.Time {
	lastT := time.Now()
	eventTimer := make(chan time.Time)
	go func() {
		defer close(eventTimer)
		for {
			t := next(lastT)
			timer := time.After(time.Until(t))
			log.Printf("expected next calendar event %s", t)
			select {
			case tx := <-timer:
				eventTimer <- tx
				lastT = tx
			}
		}
	}()
	return eventTimer
}

func (c *calendar) resolveSchedule(cal *calSchedule) (cronlib.Schedule, error) {
	if cal.Schedule != "" {
		schedule, err := cronlib.Parse(cal.Schedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule %s from calendar signal. Cause: %+v", cal.Schedule, err.Error())
		}
		return schedule, nil
	} else if cal.Interval != "" {
		intervalDuration, err := time.ParseDuration(cal.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval %s from calendar signal. Cause: %+v", cal.Interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	} else {
		return nil, fmt.Errorf("calendar signal must contain either a schedule or interval")
	}
}

func main() {
	rpcServerPort, ok := os.LookupEnv(common.GatewayProcessorServerPort)
	if !ok {
		panic("gateway rpc server port is not provided")
	}

	w := &calendar{
		log: zlog.New(os.Stdout).With().Logger(),
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", rpcServerPort))
	if err != nil {
		w.log.Fatal().Err(err).Msg("server failed to listen")
	}
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	gwProto.RegisterGatewayExecutorServer(grpcServer, w)
	w.log.Info().Str("port", rpcServerPort).Msg("gRPC server started listening...")
	grpcServer.Serve(lis)
}
