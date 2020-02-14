/*
Copyright 2020 BlackRock, Inc.

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
package custom_trigger

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

type CustomTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *logrus.Logger
	// gRPCClient is the gRPC client for the custom trigger server
	gRPCClient *grpc.ClientConn
}

func NewCustomTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger, customTriggerClients map[string]*grpc.ClientConn) (*CustomTrigger, error) {
	customTrigger := &CustomTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger,
	}

	ct := trigger.Template.CustomTrigger

	if conn, ok := customTriggerClients[trigger.Template.Name]; ok {
		if conn.GetState() == connectivity.Ready {
			customTrigger.gRPCClient = conn
			return customTrigger, nil
		}
		delete(customTriggerClients, trigger.Template.Name)
	}

	opt := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
	}

	if ct.Secure {
		creds, err := credentials.NewClientTLSFromFile(ct.CertFilePath, ct.ServerNameOverride)
		if err != nil {
			return nil, err
		}
		opt = append(opt, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.Dial(
		ct.ServerURL,
		opt...,
	)
	if err != nil {
		return nil, err
	}

	connBackoff := common.GetConnectionBackoff(nil)

	if err = wait.ExponentialBackoff(*connBackoff, func() (done bool, err error) {
		if conn.GetState() == connectivity.Ready {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, err
	}

	customTrigger.gRPCClient = conn
	customTriggerClients[trigger.Template.Name] = conn
	return customTrigger, nil
}
