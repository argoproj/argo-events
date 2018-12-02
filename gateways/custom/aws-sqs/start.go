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

package aws_sqs

import (
	"fmt"
	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/sqs"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// getSecrets retrieves the secret value from the secret in namespace with name and key
func getSecrets(client kubernetes.Interface, namespace string, name, key string) (string, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret '%s' does not have the key '%s'", name, key)
	}
	return string(val), nil
}

func (ce *AWSSQSConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	as, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *as).Msg("aws sqs configuration")

	go ce.listenEvent(as, config)

	for {
		select {
		case _, ok :=<-config.StartChan:
			if ok {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
				config.Active = true
			}

		case data, ok :=<-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			config.DoneChan <- struct{}{}
		}
	}
}

func (ce *AWSSQSConfigExecutor) listenEvent(as *AWSSQSConfig, config *gateways.ConfigContext) {
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	// at this point configuration is successfully running
	config.StartChan <- struct{}{}

	intervalDuration, err := time.ParseDuration(as.Frequency)
	if err != nil {

	}

	schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
	var next Next
	next = func(last time.Time) time.Time {
		return schedule.Next(last)
	}

	// retrieve access key id and secret access key
	accessKey, err := getSecrets(ce.Clientset, ce.Namespace, as.AccessKey.Name, as.AccessKey.Key)
	if err != nil {
		config.ErrChan <- err
		return
	}
	secretKey, err := getSecrets(ce.Clientset, ce.Namespace, as.SecretKey.Name, as.SecretKey.Key)
	if err != nil {
		config.ErrChan <- err
		return
	}

	auth := aws.Auth{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	conn := sqs.New(auth, aws.GetRegion(as.Region))
	queue, err := conn.GetQueue(as.Queue)
	if err != nil {
		ce.Log.Error().Err(err).Str("config-name", config.Data.Src).Str("queue", as.Queue).Msg("failed to get queue")
		config.ErrChan <- err
		return
	}

	lastT := time.Now()
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			msg, err := queue.ReceiveMessage(1)
			if err != nil {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("failed to get message")
				config.ErrChan <- err
				continue
			}
			config.DataChan <- []byte(msg.Messages[0].Body)
			// delete message
			_, err = queue.DeleteMessage(&msg.Messages[0])
			if err != nil {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("failed to delete message")
				config.ErrChan <- err
			}
		case <-config.DoneChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
