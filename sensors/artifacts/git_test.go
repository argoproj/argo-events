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

package artifacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var gar = &GitArtifactReader{
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
	t.Run("Given configuration, get new git reader", func(t *testing.T) {
		reader, err := NewGitReader(&v1alpha1.GitArtifact{})
		assert.NoError(t, err)
		assert.NotNil(t, reader)
	})

	t.Run("bad clone dir", func(t *testing.T) {
		_, err := NewGitReader(&v1alpha1.GitArtifact{CloneDirectory: "/abc/../opt"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("bad file path", func(t *testing.T) {
		_, err := NewGitReader(&v1alpha1.GitArtifact{FilePath: "abc/efg/../../../root"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})
}

func TestGetRemote(t *testing.T) {
	t.Run("Test git remote", func(t *testing.T) {
		remote := gar.getRemote()
		assert.Equal(t, DefaultRemote, remote)
	})
}

func TestGetBranchOrTag(t *testing.T) {
	t.Run("Given a git minio, get the branch or tag", func(t *testing.T) {
		br := gar.getBranchOrTag()
		assert.Equal(t, "refs/heads/master", br.Branch.String())
		gar.artifact.Branch = "br"
		br = gar.getBranchOrTag()
		assert.NotEqual(t, "refs/heads/master", br.Branch.String())
		gar.artifact.Tag = "t"
		tag := gar.getBranchOrTag()
		assert.NotEqual(t, "refs/heads/master", tag.Branch.String())
	})

	t.Run("Given a git minio with a specific ref, get the ref", func(t *testing.T) {
		gar.artifact.Ref = "refs/something/weird/or/specific"
		br := gar.getBranchOrTag()
		assert.Equal(t, "refs/something/weird/or/specific", br.Branch.String())
	})
}
