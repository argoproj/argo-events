package store

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestURLReader(t *testing.T) {
	urlArtifact := v1alpha1.URLArtifact{Path: "https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml"}
	urlReader, err := NewURLReader(&urlArtifact)
	assert.NotNil(t, urlReader)
	assert.Nil(t, err)
	data, err := urlReader.Read()
	assert.NotNil(t, data)
	assert.Nil(t, err)
}
