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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestFileReader(t *testing.T) {
	content := []byte("temp content")
	tmpfile, err := os.CreateTemp("", "argo-events-temp")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	fileArtifact := v1alpha1.FileArtifact{Path: tmpfile.Name()}
	fileReader, err := NewFileReader(&fileArtifact)
	assert.NotNil(t, fileReader)
	assert.Nil(t, err)
	data, err := fileReader.Read()
	assert.NotNil(t, data)
	assert.Nil(t, err)
	assert.Equal(t, content, data)
}
