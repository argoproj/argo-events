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

	"bytes"
	"github.com/argoproj/argo-events/common"
	cronlib "github.com/robfig/cron"
	zlog "github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
	"strings"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// calendarSignal describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calendarSignal struct {
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
	transformerPort string
	namespace       string
	log             zlog.Logger
	clientset       *kubernetes.Clientset
	registeredCalendarSignals []calendarSignal
}


func (c *calendar) WatchGatewayTransformerConfigMap(ctx context.Context, name string) (cache.Controller, error) {
	source := c.newStoreConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					c.log.Info().Str("config-map", name).Msg("detected ConfigMap update. Updating the controller config.")
					err := c.parseCalendars(cm)
					if err != nil {
						c.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					c.log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := c.parseCalendars(newCm)
					if err != nil {
						c.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *calendar) newStoreConfigMapWatch(name string) *cache.ListWatch {
	x := c.clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(c.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(c.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (c *calendar) parseCalendars(cm *apiv1.ConfigMap) error {
	CheckAlreadyRegistered:
		for scheduleKey, scheduleValue := range cm.Data {
			var cal *calendarSignal
			err := yaml.Unmarshal([]byte(scheduleValue), &cal)
			if err != nil {
				c.log.Error().Str("artifact", scheduleKey).Err(err).Msg("failed to parse calendar schedule")
				return err
			}
			c.log.Info().Interface("artifact", *cal).Msg("calendar schedule")
			for _, registeredCalendarSignal := range c.registeredCalendarSignals {
				if cmp.Equal(registeredCalendarSignal, cal) {
					c.log.Warn().Str("schedule", cal.Schedule).Str("interval", string(cal.Interval)).Str("recurrences", strings.Join(cal.Recurrence, ",")).Msg("calendar schedule is already registered")
					goto CheckAlreadyRegistered
				}
			}
			c.registeredCalendarSignals = append(c.registeredCalendarSignals, *cal)
			go c.startCalenderSchedule(cal)
		}
	return nil
}

func (c *calendar) startCalenderSchedule(cal *calendarSignal) error {
	schedule, err := c.resolveSchedule(cal)
	if err != nil {
		return fmt.Errorf("failed to resolve calendar schedule. Err: %+v", err)
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		return fmt.Errorf("failed to resolve calendar schedule. Err: %+v", err)
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
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		log.Printf("expected next calendar event %s", t)
		select {
		case tx := <-timer:
			lastT = tx
			c.log.Info().Msg("event sending")
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				return fmt.Errorf("failed to marshal event. Err: %+v", err)
			} else {
				// dispatch the event to gateway transformer
				_, err = http.Post(fmt.Sprintf("http://localhost:%s", c.transformerPort), "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					return fmt.Errorf("failed to dispatch the event. Err: %+v", err)
				}
				c.log.Info().Msg("event dispatched to gateway transformer")
			}
		}
	}
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

func (c *calendar) resolveSchedule(cal *calendarSignal) (cronlib.Schedule, error) {
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
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// Todo: hardcoded for now, move it to constants
	config := "calendar-gateway-configmap"
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)

	cal := &calendar{
		transformerPort: transformerPort,
		clientset:       clientset,
		namespace:       namespace,
		log:             zlog.New(os.Stdout).With().Logger(),
	}
	_, err = cal.WatchGatewayTransformerConfigMap(context.Background(), config)
	if err != nil {
		panic(fmt.Errorf("failed to watch calendar schedules. Err: %+v", err))
	}
	// wait forever
	select {}
}
