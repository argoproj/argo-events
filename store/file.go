package store

import (
	"errors"
	"io/ioutil"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// FileReader implements the ArtifactReader interface for file artifacts
type FileReader struct {
	fileArtifact *v1alpha1.FileArtifact
}

// NewFileReader creates a new ArtifactReader for inline
func NewFileReader(fileArtifact *v1alpha1.FileArtifact) (ArtifactReader, error) {
	// This should never happen!
	if fileArtifact == nil {
		return nil, errors.New("FileArtifact cannot be empty")
	}
	return &FileReader{fileArtifact}, nil
}

func (reader *FileReader) Read() ([]byte, error) {
	content, err := ioutil.ReadFile(reader.fileArtifact.Path)
	if err != nil {
		return nil, err
	}
	log.Debugf("reading fileArtifact from %s", reader.fileArtifact.Path)
	return content, nil
}
