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
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"strings"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

type calendar struct {
	events          chan metav1.Time
	transformerPort string
	namespace       string
	log             zlog.Logger
	clientset       *kubernetes.Clientset
	schedule        string
	interval        string
	recurrences     []string
}

func (c *calendar) startCalender() {
	schedule, err := c.resolveSchedule()
	if err != nil {
		panic("failed to resolve calendar schedule")
	}
	exDates, err := common.ParseExclusionDates(c.recurrences)
	if err != nil {
		panic("failed to resolve calendar schedule")
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
				c.log.Warn().Err(err).Msg("failed to get event bytes")
			} else {
				// dispatch the event to gateway transformer
				_, err = http.Post(fmt.Sprintf("http://localhost:%s", c.transformerPort), "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					c.log.Warn().Err(err).Msg("failed to dispatch the event")
				}
				c.log.Info().Msg("event sent")
			}
		}
	}
}

func (c *calendar) handleEvents(events chan metav1.Time, next Next) {
	defer close(events)
	eventTimer := c.getEventTimer(next)
	for t := range eventTimer {
		event := metav1.Time{Time: t}
		payload, err := event.Marshal()
		if err != nil {
			c.log.Warn().Err(err).Msg("failed to get event bytes")
		} else {
			// dispatch the event to gateway transformer
			_, err = http.Post(fmt.Sprintf("http://localhost:%s", c.transformerPort), "application/octet-stream", bytes.NewReader(payload))
			if err != nil {
				c.log.Warn().Err(err).Msg("failed to dispatch the event")
			}
		}
		events <- event
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

func (cal *calendar) resolveSchedule() (cronlib.Schedule, error) {
	if cal.schedule != "" {
		schedule, err := cronlib.Parse(cal.schedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule %s from calendar signal. Cause: %+v", cal.schedule, err.Error())
		}
		return schedule, nil
	} else if cal.interval != "" {
		intervalDuration, err := time.ParseDuration(cal.interval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval %s from calendar signal. Cause: %+v", cal.interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	} else {
		return nil, fmt.Errorf("calendar signal must contain either a schedule or interval")
	}
}

// validate the calender configuration
func (c *calendar) validate(configMapName string) (*map[string]string, error) {
	configmap, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		c.log.Panic().Str("configmap", configMapName).Err(err).Msg("failed to get configmap")
	}
	configData := configmap.Data
	if configData["interval"] != "" && configData["scheduler"] != "" {
		c.log.Error().Msg("invalid configuration. can't have both interval and schedule set")
	}
	if configData["interval"] == "" && configData["scheduler"] == "" {
		c.log.Error().Msg("invalid configuration. either interval or schedule set")
	}
	return &configData, err
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
	events := make(chan metav1.Time)

	cal := &calendar{
		transformerPort: transformerPort,
		clientset:       clientset,
		events:          events,
		namespace:       namespace,
		log:             zlog.New(os.Stdout).With().Logger(),
	}

	// validate the configuration
	configData, err := cal.validate(config)
	if err != nil {
		panic("invalid configuration")
	}
	cal.schedule = (*configData)["schedule"]
	cal.interval = (*configData)["interval"]
	cal.recurrences = strings.Split((*configData)["recurrences"], ",")
	cal.startCalender()
}
