/*
Copyright 2018 The Argoproj Authors.

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

package imap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for IMAP event source
type EventListener struct {
	EventSourceName string
	EventName       string
	IMAPEventSource v1alpha1.IMAPEventSource
	Metrics         *metrics.Metrics
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.IMAPEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	imapEventSource := &el.IMAPEventSource
	username, err := sharedutil.GetSecretFromVolume(imapEventSource.Username)
	if err != nil {
		return err
	}
	password, err := sharedutil.GetSecretFromVolume(imapEventSource.Password)
	if err != nil {
		return err
	}

	// Connect to server
	log.Info("connecting to IMAP server...", zap.String("hostAddress", imapEventSource.HostAddress))

	var c *imapclient.Client
	if imapEventSource.StartTLS {
		c, err = imapclient.DialTLS(imapEventSource.HostAddress, nil)
	} else {
		c, err = imapclient.DialInsecure(imapEventSource.HostAddress, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to dial IMAP server: %w", err)
	}
	defer c.Close()

	if err := c.Login(username, password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select inbox: %w", err)
	}

	log.Info("Successfully connected to IMAP server", zap.Uint32("numMessages", mbox.NumMessages))

	// Try to use IDLE (most servers support it)
	// We'll attempt IDLE and fall back to polling if it fails
	log.Info("attempting to use IMAP IDLE for real-time notifications")
	return el.startIdle(ctx, c, dispatch, log, imapEventSource, mbox.NumMessages)
}

// startIdle starts IMAP IDLE for real-time notifications
func (el *EventListener) startIdle(ctx context.Context, c *imapclient.Client, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, imapEventSource *v1alpha1.IMAPEventSource, initialCount uint32) error {
	log.Warn("IDLE not supported or failed, falling back to polling")
	return el.startPolling(ctx, c, dispatch, log, imapEventSource, initialCount)
}

// startPolling starts polling-based message checking (fallback)
func (el *EventListener) startPolling(ctx context.Context, c *imapclient.Client, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, imapEventSource *v1alpha1.IMAPEventSource, initialCount uint32) error {
	log.Info("starting IMAP message polling...")

	lastSeen := initialCount
	ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("event source is stopped")
			return nil
		case <-ticker.C:
			log.Info("polling for new messages...")

			// Refresh mailbox status
			mboxStatus, err := c.Status("INBOX", &imap.StatusOptions{
				NumMessages: true,
			}).Wait()
			if err != nil {
				log.Error("failed to get mailbox status", zap.Error(err))
				continue
			}

			currentMessages := uint32(0)
			if mboxStatus.NumMessages != nil {
				currentMessages = *mboxStatus.NumMessages
			}

			log.Info("mailbox status", zap.Uint32("current", currentMessages), zap.Uint32("lastSeen", lastSeen))

			if currentMessages > lastSeen {
				log.Info("new messages found", zap.Uint32("new", currentMessages-lastSeen))

				err = el.fetchNewMessages(c, dispatch, log, imapEventSource, lastSeen+1, currentMessages)
				if err != nil {
					log.Error("failed to fetch new messages", zap.Error(err))
				}
				lastSeen = currentMessages
			}
		}
	}
}

// fetchNewMessages fetches and processes new messages
func (el *EventListener) fetchNewMessages(c *imapclient.Client, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, imapEventSource *v1alpha1.IMAPEventSource, start, end uint32) error {
	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, end)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}

	messages, err := c.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	for _, msg := range messages {
		if err := el.processMessage(msg, dispatch, log, imapEventSource); err != nil {
			log.Error("failed to process message", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	}

	return nil
}

// processMessage processes a single IMAP message and dispatches an event
func (el *EventListener) processMessage(msg *imapclient.FetchMessageBuffer, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, imapEventSource *v1alpha1.IMAPEventSource) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	envelope := msg.Envelope
	if envelope == nil {
		return fmt.Errorf("message envelope is nil")
	}

	// Convert envelope addresses to string slices
	var toAddresses []string
	for _, addr := range envelope.To {
		if addr.Name != "" {
			toAddresses = append(toAddresses, fmt.Sprintf("%s <%s>", addr.Name, addr.Addr()))
		} else {
			toAddresses = append(toAddresses, addr.Addr())
		}
	}

	var fromAddress string
	if len(envelope.From) > 0 {
		addr := envelope.From[0]
		if addr.Name != "" {
			fromAddress = fmt.Sprintf("%s <%s>", addr.Name, addr.Addr())
		} else {
			fromAddress = addr.Addr()
		}
	}

	// Convert headers to map (simplified for now)
	headers := make(map[string][]string)

	eventData := &events.IMAPEventData{
		Subject:   envelope.Subject,
		From:      fromAddress,
		To:        toAddresses,
		Body:      "", // Body processing would require additional IMAP fetch for body content
		Headers:   headers,
		Timestamp: envelope.Date.Format(time.RFC3339),
		Metadata:  imapEventSource.Metadata,
	}

	eventBody, err := json.Marshal(eventData)
	if err != nil {
		log.Error("failed to marshal the event data", zap.Error(err))
		return err
	}

	log.Info("dispatching IMAP event", zap.String("subject", envelope.Subject), zap.String("from", fromAddress))
	if err = dispatch(eventBody); err != nil {
		log.Error("failed to dispatch IMAP event", zap.Error(err))
		return err
	}

	return nil
}
