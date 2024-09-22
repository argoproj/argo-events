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

package aws

import (
	"context"
	"os"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAWS(t *testing.T) {
	client := fake.NewSimpleClientset()
	namespace := "test"
	secretName := "test-secret"
	accessKey := "YWNjZXNz"
	secretKey := "c2VjcmV0"
	LabelAccessKey := "access"
	LabelSecretKey := "secret"

	convey.Convey("Given credentials are in ENV by using envFrom, create AWS credential", t, func() {
		os.Setenv(secretName+"_"+LabelAccessKey, accessKey)
		os.Setenv(secretName+"_"+LabelSecretKey, secretKey)

		creds, err := GetAWSCredFromEnvironment(&corev1.SecretKeySelector{
			Key: LabelAccessKey,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName,
			},
		}, &corev1.SecretKeySelector{
			Key: LabelSecretKey,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName,
			},
		})

		convey.So(err, convey.ShouldBeNil)
		convey.So(creds, convey.ShouldNotBeNil)

		value, err := creds.Get()
		convey.So(err, convey.ShouldBeNil)
		convey.So(value.AccessKeyID, convey.ShouldEqual, accessKey)
		convey.So(value.SecretAccessKey, convey.ShouldEqual, secretKey)

		convey.Convey("Get a new aws session", func() {
			session, err := GetAWSSession(creds, "mock-region")
			convey.So(err, convey.ShouldBeNil)
			convey.So(session, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given kubernetes secret that hold credentials, create AWS credential", t, func() {
		secret, err := client.CoreV1().Secrets(namespace).Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				LabelAccessKey: []byte(accessKey),
				LabelSecretKey: []byte(secretKey),
			},
		}, metav1.CreateOptions{})

		convey.So(err, convey.ShouldBeNil)
		convey.So(secret, convey.ShouldNotBeNil)
	})

	convey.Convey("create AWS credential using already present config/IAM role", t, func() {
		convey.Convey("Get a new aws session", func() {
			session, err := GetAWSSessionWithoutCreds("mock-region")
			convey.So(err, convey.ShouldBeNil)
			convey.So(session, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("create AWS credential using assume roleARN", t, func() {
		convey.Convey("Get a new aws session", func() {
			session, err := GetAWSAssumeRoleCreds("moke-roleARN", "mock-region")
			convey.So(err, convey.ShouldBeNil)
			convey.So(session, convey.ShouldNotBeNil)
		})
	})
}
