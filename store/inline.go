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
	"errors"

	"github.com/argoproj/argo-events/common/logging"
)

// InlineReader implements the ArtifactReader interface for inlined artifacts
type InlineReader struct {
	inlineArtifact *string
}

// NewInlineReader creates a new ArtifactReader for inline
func NewInlineReader(inlineArtifact *string) (ArtifactReader, error) {
	// This should never happen!
	if inlineArtifact == nil {
		return nil, errors.New("InlineArtifact does not exist")
	}
	return &InlineReader{inlineArtifact}, nil
}

func (reader *InlineReader) Read() ([]byte, error) {
	log := logging.NewArgoEventsLogger()
	log.Debug("reading fileArtifact from inline")
	return []byte(*reader.inlineArtifact), nil
}
