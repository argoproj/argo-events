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

package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/fsevent"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for sftp event source
type EventListener struct {
	EventSourceName string
	EventName       string
	SFTPEventSource v1alpha1.SFTPEventSource
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
	return v1alpha1.SFTPEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	username, err := sharedutil.GetSecretFromVolume(el.SFTPEventSource.Username)
	if err != nil {
		return fmt.Errorf("username not found, %w", err)
	}
	address, err := sharedutil.GetSecretFromVolume(el.SFTPEventSource.Address)
	if err != nil {
		return fmt.Errorf("address not found, %w", err)
	}

	var authMethod ssh.AuthMethod
	var hostKeyCallback ssh.HostKeyCallback

	if el.SFTPEventSource.SSHKeySecret != nil {
		sshKeyPath, err := sharedutil.GetSecretVolumePath(el.SFTPEventSource.SSHKeySecret)
		if err != nil {
			return fmt.Errorf("failed to get SSH key from mounted volume, %w", err)
		}
		sshKey, err := os.ReadFile(sshKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read ssh key file. err: %+v", err)
		}
		signer, err := ssh.ParsePrivateKey(sshKey)
		if err != nil {
			return fmt.Errorf("failed to parse private ssh key. err: %+v", err)
		}
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey(sshKey)
		if err != nil {
			return fmt.Errorf("failed to parse public ssh key. err: %+v", err)
		}
		authMethod = ssh.PublicKeys(signer)
		hostKeyCallback = ssh.FixedHostKey(publicKey)
	} else {
		password, err := sharedutil.GetSecretFromVolume(el.SFTPEventSource.Password)
		if err != nil {
			return fmt.Errorf("password not found, %w", err)
		}
		authMethod = ssh.Password(password)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sftpConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
	}

	var sshClient *ssh.Client
	err = sharedutil.DoWithRetry(nil, func() error {
		var err error
		sshClient, err = ssh.Dial("tcp", address, sftpConfig)
		return err
	})
	if err != nil {
		return fmt.Errorf("dialing sftp address %s: %w", address, err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("new sftp client: %w", err)
	}
	defer sftpClient.Close()

	if err := el.listenEvents(ctx, sftpClient, dispatch, log); err != nil {
		log.Error("failed to listen to events", zap.Error(err))
		return err
	}
	return nil
}

// listenEvents listen to sftp related events.
func (el *EventListener) listenEvents(ctx context.Context, sftpClient *sftp.Client, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	sftpEventSource := &el.SFTPEventSource

	log.Info("identifying new files in sftp...")
	startingFiles, err := sftpNonDirFiles(sftpClient, sftpEventSource.WatchPathConfig.Directory)
	if err != nil {
		return fmt.Errorf("failed to read directory %s for %s, %w", sftpEventSource.WatchPathConfig.Directory, el.GetEventName(), err)
	}

	// TODO: do we need some sort of stateful mechanism to capture changes between event source restarts?
	// This would allow loading startingFiles from store/cache rather than initializing starting files from  remote sftp source

	var pathRegexp *regexp.Regexp
	if sftpEventSource.WatchPathConfig.PathRegexp != "" {
		log.Infow("matching file path with configured regex...", zap.Any("regex", sftpEventSource.WatchPathConfig.PathRegexp))
		pathRegexp, err = regexp.Compile(sftpEventSource.WatchPathConfig.PathRegexp)
		if err != nil {
			return fmt.Errorf("failed to match file path with configured regex %s for %s, %w", sftpEventSource.WatchPathConfig.PathRegexp, el.GetEventName(), err)
		}
	}

	processOne := func(event fsnotify.Event) error {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		log.Infow("sftp event", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))

		fileEvent := fsevent.Event{Name: event.Name, Op: fsevent.NewOp(event.Op.String()), Metadata: el.SFTPEventSource.Metadata}
		payload, err := json.Marshal(fileEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal the event to the fs event, %w", err)
		}
		log.Infow("dispatching sftp event on data channel...", zap.Any("event-type", event.Op.String()), zap.Any("descriptor-name", event.Name))
		if err = dispatch(payload); err != nil {
			return fmt.Errorf("failed to dispatch an sftp event, %w", err)
		}
		return nil
	}

	maybeProcess := func(fi fs.FileInfo, op fsnotify.Op) error {
		matched := false
		relPath := strings.TrimPrefix(fi.Name(), sftpEventSource.WatchPathConfig.Directory)
		if sftpEventSource.WatchPathConfig.Path != "" && sftpEventSource.WatchPathConfig.Path == relPath {
			matched = true
		} else if pathRegexp != nil && pathRegexp.MatchString(relPath) {
			matched = true
		}
		if matched && sftpEventSource.EventType == op.String() {
			if err = processOne(fsnotify.Event{
				Name: fi.Name(),
				Op:   op,
			}); err != nil {
				log.Errorw("failed to process a sftp event", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			}
		}

		return nil
	}

	pollIntervalDuration := time.Second * 10
	if d, err := time.ParseDuration(sftpEventSource.PollIntervalDuration); err != nil {
		pollIntervalDuration = d
	} else {
		log.Errorw("failed parsing poll interval duration.. falling back to %s: %w", pollIntervalDuration.String(), err)
	}

	log.Info("listening to sftp notifications... polling interval %s", pollIntervalDuration.String())
	for {
		select {
		case <-time.After(pollIntervalDuration):

			files, err := sftpNonDirFiles(sftpClient, sftpEventSource.WatchPathConfig.Directory)
			if err != nil {
				return fmt.Errorf("failed to read directory %s for %s, %w", sftpEventSource.WatchPathConfig.Directory, el.GetEventName(), err)
			}

			fileDiff := diffFiles(startingFiles, files)
			if fileDiff.isEmpty() {
				continue
			}

			log.Infof("found %d new files and %d removed files", len(fileDiff.new), len(fileDiff.removed))

			for _, fi := range fileDiff.removed {
				if err = maybeProcess(fi, fsnotify.Remove); err != nil {
					log.Errorw("failed to process a file event", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				}
			}
			for _, fi := range fileDiff.new {
				if err = maybeProcess(fi, fsnotify.Create); err != nil {
					log.Errorw("failed to process a file event", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				}
			}

			// TODO: errors processing files will result in dropped events
			// adjusting the logic for overwriting startingFiles could enable the next tick
			// to capture the event
			startingFiles = files

		case <-ctx.Done():
			log.Info("event source has been stopped")
			return nil
		}
	}
}

func sftpNonDirFiles(sftpClient *sftp.Client, dir string) ([]fs.FileInfo, error) {
	var files []fs.FileInfo
	err := sharedutil.DoWithRetry(nil, func() error {
		var err error
		files, err = sftpClient.ReadDir(dir)
		return err
	})
	if err != nil {
		return nil, err
	}
	var nonDirFiles []fs.FileInfo
	for _, f := range files {
		if !f.IsDir() {
			nonDirFiles = append(nonDirFiles, f)
		}
	}

	files = nonDirFiles
	return files, nil
}

type fileDiff struct {
	new     []fs.FileInfo
	removed []fs.FileInfo
}

func (f fileDiff) isEmpty() bool {
	return (len(f.new) + len(f.removed)) == 0
}

func diffFiles(startingFiles, currentFiles []fs.FileInfo) fileDiff {
	fileMap := make(map[string]fs.FileInfo)
	for _, file := range currentFiles {
		fileMap[file.Name()] = file
	}

	var diff fileDiff

	for _, startingFile := range startingFiles {
		name := startingFile.Name()

		if newFile, ok := fileMap[name]; !ok {
			diff.removed = append(diff.removed, startingFile)
		} else {
			delete(fileMap, name)

			if newFile.Size() != startingFile.Size() || newFile.ModTime() != startingFile.ModTime() {
				diff.new = append(diff.new, newFile)
			}
		}
	}

	for _, newFile := range fileMap {
		diff.new = append(diff.new, newFile)
	}

	return diff
}
