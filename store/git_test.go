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

package store

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var gar = &GitArtifactReader{
	kubeClientset: fake.NewSimpleClientset(),
	namespace:     "fake",
	artifact: &v1alpha1.GitArtifact{
		URL: "fake",
		Creds: &v1alpha1.GitCreds{
			Username: &corev1.SecretKeySelector{
				Key: "username",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "git-secret",
				},
			},
			Password: &corev1.SecretKeySelector{
				Key: "password",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "git-secret",
				},
			},
		},
	},
}

func TestNewGitReader(t *testing.T) {
	convey.Convey("Given configuration, get new git reader", t, func() {
		reader, err := NewGitReader(fake.NewSimpleClientset(), "fake-ns", &v1alpha1.GitArtifact{})
		convey.So(err, convey.ShouldBeNil)
		convey.So(reader, convey.ShouldNotBeNil)
	})
}

func TestGetRemote(t *testing.T) {
	convey.Convey("Test git remote", t, func() {
		remote := gar.getRemote()
		convey.So(remote, convey.ShouldEqual, DefaultRemote)
	})
}

func TestGetGitAuth(t *testing.T) {
	convey.Convey("Given auth secret, get git auth tokens", t, func() {
		secret, err := gar.kubeClientset.CoreV1().Secrets("fake").Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gar.artifact.Creds.Username.Name,
				Namespace: "fake",
			},
			Data: map[string][]byte{
				"username": []byte("username"),
				"password": []byte("password"),
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(secret, convey.ShouldNotBeNil)

		auth, err := gar.getGitAuth()
		convey.So(err, convey.ShouldBeNil)
		convey.So(auth, convey.ShouldNotBeNil)
	})
}

func TestGetBranchOrTag(t *testing.T) {
	convey.Convey("Given a git minio, get the branch or tag", t, func() {
		br := gar.getBranchOrTag()
		convey.So(br.Branch, convey.ShouldEqual, "refs/heads/master")
		gar.artifact.Branch = "br"
		br = gar.getBranchOrTag()
		convey.So(br.Branch, convey.ShouldNotEqual, "refs/heads/master")
		gar.artifact.Tag = "t"
		tag := gar.getBranchOrTag()
		convey.So(tag.Branch, convey.ShouldNotEqual, "refs/heads/master")
	})

	convey.Convey("Given a git minio with a specific ref, get the ref", t, func() {
		gar.artifact.Ref = "refs/something/weird/or/specific"
		br := gar.getBranchOrTag()
		convey.So(br.Branch, convey.ShouldEqual, "refs/something/weird/or/specific")
	})
}
