package store

import (
	"errors"

	log "github.com/sirupsen/logrus"
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
	log.Debug("reading fileArtifact from inline")
	return []byte(*reader.inlineArtifact), nil
}
