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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/mail"
	"strings"
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

	es := &el.IMAPEventSource

	username, err := sharedutil.GetSecretFromVolume(es.Username)
	if err != nil {
		return err
	}
	password, err := sharedutil.GetSecretFromVolume(es.Password)
	if err != nil {
		return err
	}

	mailbox := es.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	pollInterval := 30 * time.Second
	if es.PollInterval != "" {
		if d, err := time.ParseDuration(es.PollInterval); err == nil {
			pollInterval = d
		} else {
			log.Warn("invalid pollInterval, using default", zap.Error(err))
		}
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := el.run(ctx, dispatch, log, es, username, password, mailbox, pollInterval); err != nil {
			log.Error("IMAP event source failed, reconnecting", zap.Error(err))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
			}
			continue
		}
		return nil
	}
}

// run performs a single connection session.
func (el *EventListener) run(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, es *v1alpha1.IMAPEventSource, username, password, mailbox string, pollInterval time.Duration) error {
	updateCh := make(chan struct{}, 1)
	opts := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				select {
				case updateCh <- struct{}{}:
				default:
				}
			},
		},
	}

	var c *imapclient.Client
	var err error
	if err := sharedutil.DoWithRetry(es.ConnectionBackoff, func() error {
		if es.StartTLS {
			c, err = imapclient.DialTLS(es.HostAddress, opts)
		} else {
			c, err = imapclient.DialInsecure(es.HostAddress, opts)
		}
		if err != nil {
			return fmt.Errorf("failed to dial IMAP server: %w", err)
		}
		if err := c.Login(username, password).Wait(); err != nil {
			c.Close()
			return fmt.Errorf("failed to login: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	defer c.Close()

	mbox, err := c.Select(mailbox, nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}
	log.Info("connected to IMAP server", zap.String("mailbox", mailbox), zap.Uint32("numMessages", mbox.NumMessages))

	// Determine starting UID. For UnseenOnly, process existing unseen; otherwise ignore existing.
	lastUID := imap.UID(0)
	if !es.UnseenOnly {
		if data, err := c.UIDSearch(&imap.SearchCriteria{}, &imap.SearchOptions{ReturnMax: true}).Wait(); err == nil {
			lastUID = imap.UID(data.Max)
		}
	}

	idleSupported := true

	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := el.fetchNew(ctx, c, dispatch, log, es, lastUID, &lastUID); err != nil {
			return err
		}

		if idleSupported {
			idleCmd, err := c.Idle()
			if err != nil {
				idleSupported = false
				log.Warn("IDLE not supported, falling back to polling", zap.Error(err))
			} else {
				select {
				case <-ctx.Done():
					_ = idleCmd.Close()
					return nil
				case <-updateCh:
				case <-time.After(pollInterval):
				}
				if err := idleCmd.Close(); err != nil {
					log.Debug("idle close", zap.Error(err))
				}
				continue
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(pollInterval):
		}
	}
}

// fetchNew fetches and dispatches messages with UID greater than lastUID.
func (el *EventListener) fetchNew(ctx context.Context, c *imapclient.Client, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, es *v1alpha1.IMAPEventSource, lastUID imap.UID, lastUIDPtr *imap.UID) error {
	criteria := &imap.SearchCriteria{}
	if lastUID > 0 {
		uidSet := imap.UIDSet{}
		uidSet.AddRange(lastUID+1, 0)
		criteria.UID = []imap.UIDSet{uidSet}
	}
	if es.UnseenOnly {
		criteria.NotFlag = []imap.Flag{imap.FlagSeen}
	}

	data, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return fmt.Errorf("uid search failed: %w", err)
	}
	uidSet, _ := data.All.(imap.UIDSet)
	nums, _ := uidSet.Nums()
	if len(nums) == 0 {
		return nil
	}

	fetchOpts := &imap.FetchOptions{
		Envelope:      true,
		UID:           true,
		InternalDate:  true,
		BodyStructure: &imap.FetchItemBodyStructure{},
		BodySection:   []*imap.FetchItemBodySection{{}},
	}

	messages, err := c.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		var attachments []events.IMAPAttachment
		if es.IncludeAttachments && msg.BodyStructure != nil {
			attachments = el.fetchAttachments(c, msg, log)
		}

		if err := el.processMessage(ctx, msg, dispatch, log, es, attachments); err != nil {
			log.Error("failed to process message", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			continue
		}
		if msg.UID > *lastUIDPtr {
			*lastUIDPtr = msg.UID
		}
		el.postProcess(c, msg, es, log)
	}
	return nil
}

// processMessage processes a single IMAP message and dispatches an event
func (el *EventListener) processMessage(ctx context.Context, msg *imapclient.FetchMessageBuffer, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger, es *v1alpha1.IMAPEventSource, attachments []events.IMAPAttachment) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	envelope := msg.Envelope
	if envelope == nil {
		return fmt.Errorf("message envelope is nil")
	}

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

	// Parse body and headers from fetched RFC822 content.
	var body string
	headers := make(map[string][]string)
	for _, data := range msg.BodySection {
		if len(data) == 0 {
			continue
		}
		mr, err := mail.ReadMessage(bytes.NewReader(data))
		if err == nil {
			for k, v := range mr.Header {
				headers[k] = v
			}
			b, _ := io.ReadAll(mr.Body)
			body = string(b)
		} else {
			body = string(data)
		}
		break
	}

	eventData := &events.IMAPEventData{
		Subject:     envelope.Subject,
		From:        fromAddress,
		To:          toAddresses,
		Body:        body,
		Headers:     headers,
		Timestamp:   envelope.Date.Format(time.RFC3339),
		Metadata:    es.Metadata,
		Attachments: attachments,
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

// postProcess performs mailbox post-processing actions (mark seen, delete, move)
// after a message has been successfully dispatched.
func (el *EventListener) postProcess(c *imapclient.Client, msg *imapclient.FetchMessageBuffer, es *v1alpha1.IMAPEventSource, log *zap.SugaredLogger) {
	uidSet := imap.UIDSetNum(msg.UID)

	if es.MarkSeen {
		if _, err := c.Store(uidSet, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Collect(); err != nil {
			log.Warn("failed to mark message seen", zap.Error(err))
		}
	}

	if es.MoveToMailbox != "" {
		cmd := c.Move(uidSet, es.MoveToMailbox)
		if _, err := cmd.Wait(); err != nil {
			log.Warn("failed to move message", zap.String("mailbox", es.MoveToMailbox), zap.Error(err))
		}
		return // moved message is no longer in this mailbox; skip delete
	}

	if es.Delete {
		if _, err := c.Store(uidSet, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}, nil).Collect(); err != nil {
			log.Warn("failed to mark message deleted", zap.Error(err))
		}
		// We intentionally do not EXPUNGE here to avoid affecting other concurrent sessions.
	}
}

// fetchAttachments finds attachment parts via BodyStructure and fetches their content.
func (el *EventListener) fetchAttachments(c *imapclient.Client, msg *imapclient.FetchMessageBuffer, log *zap.SugaredLogger) []events.IMAPAttachment {
	if msg.BodyStructure == nil {
		return nil
	}

	var atts []events.IMAPAttachment
	msg.BodyStructure.Walk(func(path []int, part imap.BodyStructure) bool {
		sp, ok := part.(*imap.BodyStructureSinglePart)
		if !ok {
			return true
		}
		filename := sp.Filename()
		if filename == "" {
			if d := sp.Disposition(); d != nil {
				filename = d.Params["filename"]
			}
		}
		if filename == "" {
			return true
		}

		// fetch this part's body section
		sect := &imap.FetchItemBodySection{Part: path, Peek: true}
		opts := &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{sect}}
		uidSet := imap.UIDSetNum(msg.UID)
		fetched, err := c.Fetch(uidSet, opts).Collect()
		if err != nil || len(fetched) == 0 {
			log.Debug("failed to fetch attachment section", zap.Error(err), zap.String("filename", filename))
			// still record metadata
			atts = append(atts, events.IMAPAttachment{
				Filename:    filename,
				ContentType: sp.MediaType(),
				Size:        int64(sp.Size),
			})
			return true
		}
		var data []byte
		for _, fmsg := range fetched {
			if b, ok := fmsg.BodySection[sect]; ok {
				data = b
				break
			}
		}
		decoded, err := decodeTransfer(sp.Encoding, data)
		if err != nil {
			log.Debug("decode attachment failed, using raw", zap.Error(err))
			decoded = data
		}
		atts = append(atts, events.IMAPAttachment{
			Filename:    filename,
			ContentType: sp.MediaType(),
			Size:        int64(sp.Size),
			Data:        decoded,
		})
		return true
	})
	return atts
}

// decodeTransfer decodes body section data according to Content-Transfer-Encoding.
func decodeTransfer(encoding string, data []byte) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "", "7bit", "8bit", "binary":
		return data, nil
	case "base64":
		// data from fetch may include newlines etc.
		return base64.StdEncoding.DecodeString(string(data))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(bytes.NewReader(data)))
	default:
		return data, fmt.Errorf("unknown transfer encoding: %s", encoding)
	}
}
