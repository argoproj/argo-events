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
	"testing"
)

// Todo: implement the tests

func TestExtractAndCreateStreamSignal(t *testing.T) {
}

func TestSignal(t *testing.T) {
}

var notificationInfo = `
{
  "EventType": "s3:ObjectCreated:Put",
  "Key": "images/myphoto.jpg",
  "Records": [
    {
    "eventVersion": "2.0",
    "eventSource": "minio:s3",
    "awsRegion": "",
    "eventTime": "2017-07-07T18:46:37Z",
    "eventName": "s3:ObjectCreated:Put",
    "userIdentity": {
      "principalId": "minio"
    },
    "requestParameters": {
      "sourceIPAddress": "192.168.1.80:55328"
    },
    "responseElements": {
      "x-amz-request-id": "14CF20BD1EFD5B93",
      "x-minio-origin-endpoint": "http://127.0.0.1:9000"
    },
    "s3": {
      "s3SchemaVersion": "1.0",
      "configurationId": "Config",
      "bucket": {
      "name": "images",
      "ownerIdentity": {
        "principalId": "minio"
      },
      "arn": "arn:aws:s3:::images"
      },
      "object": {
      "key": "myphoto.jpg",
      "size": 248682,
      "eTag": "f1671feacb8bbf7b0397c6e9364e8c92",
      "contentType": "image/jpeg",
      "userDefined": {
        "content-type": "image/jpeg"
      },
      "versionId": "1",
      "sequencer": "14CF20BD1EFD5B93"
      }
    },
    "source": {
      "host": "192.168.1.80",
      "port": "55328",
      "userAgent": "Minio (linux; amd64) minio-go/2.0.4 mc/DEVELOPMENT.GOGET"
    }
    }
  ],
  "level": "info",
  "msg": "",
  "time": "2017-07-07T11:46:37-07:00"
}
`
