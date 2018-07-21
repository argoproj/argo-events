package store

// InlineReader implements the ArtifactReader interface for inlined artifacts
type InlineReader struct {
	inlineArtifact string
}

// NewInlineReader creates a new ArtifactReader for inline
func NewInlineReader(inlineArtifact string) (ArtifactReader, error) {
	// TODO(shri): Add logging here
	return &InlineReader{inlineArtifact}, nil
}

func (reader *InlineReader) Read() ([]byte, error) {
	return []byte(reader.inlineArtifact), nil
}
