package store

import (
	"io/ioutil"
	"log"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// FileReader implements the ArtifactReader interface for file artifacts
type FileReader struct {
	fileArtifact *v1alpha1.FileArtifact
}

// NewFileReader creates a new ArtifactReader for inline
func NewFileReader(fileArtifact *v1alpha1.FileArtifact) (ArtifactReader, error) {
	// This should never happen!
	if fileArtifact == nil {
		panic("FileArtifact cannot be empty!")
	}
	log.Printf("Creating fileReader from %s!\n", fileArtifact.Path)
	return &FileReader{fileArtifact}, nil
}

func (reader *FileReader) Read() ([]byte, error) {
	content, err := ioutil.ReadFile(reader.fileArtifact.Path)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Reading fileArtifact from %s!\n", reader.fileArtifact.Path)
	return content, nil
}
