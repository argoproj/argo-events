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
package email

import (
	"context"
	"encoding/json"
	"fmt"

	notifications "github.com/argoproj/notifications-engine/pkg/services"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
)

type EmailTrigger struct {
	// Sensor refer to the sensor object
	Sensor *v1alpha1.Sensor
	// Trigger refers to the trigger resource
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
	// emailSvc refers to the Email notification service.
	emailSvc notifications.NotificationService
}

// NewEmailTrigger returns a new Email trigger context
func NewEmailTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*EmailTrigger, error) {
	emailTrigger := trigger.Template.Email
	smtpPassword, err := common.GetSecretFromVolume(emailTrigger.SMTPPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the smtp password, %w", err)
	}
	emailSvc := notifications.NewEmailService(
		notifications.EmailOptions{
			Host:     emailTrigger.Host,
			Port:     int(emailTrigger.Port),
			Username: emailTrigger.Username,
			Password: smtpPassword,
			From:     emailTrigger.From,
		},
	)
	return &EmailTrigger{
		Sensor:   sensor,
		Trigger:  trigger,
		Logger:   logger.With(logging.LabelTriggerType, apicommon.EmailTrigger),
		emailSvc: emailSvc,
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *EmailTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.EmailTrigger
}

func (t *EmailTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Email, nil
}

func (t *EmailTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the Email trigger resource, %w", err)
	}
	parameters := t.Trigger.Template.Email.Parameters

	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.Email.Parameters, events)
		if err != nil {
			return nil, err
		}

		var st *v1alpha1.EmailTrigger
		if err := json.Unmarshal(updatedResourceBytes, &st); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated Email trigger resource after applying resource parameters, %w", err)
		}

		return st, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (t *EmailTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	t.Logger.Info("executing EmailTrigger")
	_, ok := resource.(*v1alpha1.EmailTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to marshal the Email trigger resource")
	}

	emailTrigger := t.Trigger.Template.Email

	t.Logger.Infow("sending emails...", zap.Any("to", emailTrigger.To))

	notification := notifications.Notification{
		Message: emailTrigger.Body,
		Email: &notifications.EmailNotification{
			Subject: emailTrigger.Subject,
			Body:    emailTrigger.Body,
		},
	}
	destination := notifications.Destination{
		Service:   "email",
		Recipient: fmt.Sprint(emailTrigger.To),
	}
	err := t.emailSvc.Send(notification, destination)
	if err != nil {
		t.Logger.Errorw("unable to send emails to emailIds", zap.Any("to", emailTrigger.To), zap.Error(err))
		return nil, fmt.Errorf("failed to send emails to emailIds %v, %w", emailTrigger.To, err)
	}

	t.Logger.Infow("message successfully sent to emailIds", zap.Any("message", emailTrigger.Body), zap.Any("to", emailTrigger.To))
	t.Logger.Info("finished executing EmailTrigger")
	return nil, nil
}

// No Policies for EmailTrigger
func (t *EmailTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
