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

package artifact

import (
	"encoding/json"
	"time"

	"github.com/argoproj/argo-events/job"

	"github.com/minio/minio-go"
)

type event struct {
	job.AbstractEvent
	artifact       *artifact
	s3Notification *minio.NotificationEvent
}

func (e *event) GetID() string {
	return e.artifact.AbstractSignal.GetID()
}

func (e *event) GetBody() []byte {
	// let's just encode the s3Notification and ignore any possible errors
	b, _ := json.Marshal(e.s3Notification)
	return b
}

func (e *event) GetSource() string {
	if e.s3Notification != nil {
		return e.s3Notification.S3.Bucket.Name + "/" + e.s3Notification.S3.Object.Key
	}
	return "unknown"
}

func (e *event) GetTimestamp() time.Time {
	if e.s3Notification != nil {
		t, err := time.Parse(time.RFC3339, e.s3Notification.EventTime)
		if err != nil {
			// is it okay to just return the time now?
			return time.Now().UTC()
		}
		return t
	}
	return time.Now().UTC()
}

func (e *event) GetSignal() job.Signal {
	return e.artifact
}
