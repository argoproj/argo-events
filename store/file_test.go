package store

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFileReader(t *testing.T) {
	content := []byte("temp content")
	tmpfile, err := ioutil.TempFile("", "argo-events-temp")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
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
