package store

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestURLReader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	urlArtifact := v1alpha1.URLArtifact{Path: ts.URL}
	assert.False(t, urlArtifact.VerifyCert)
	urlReader, err := NewURLReader(&urlArtifact)
	assert.NotNil(t, urlReader)
	assert.Nil(t, err)
	data, err := urlReader.Read()
	assert.NotNil(t, data)
	assert.Nil(t, err)
}
