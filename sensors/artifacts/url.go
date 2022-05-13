package artifacts

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// URLReader implements the ArtifactReader interface for urls
type URLReader struct {
	urlArtifact *v1alpha1.URLArtifact
}

// NewURLReader creates a new ArtifactReader for workflows at URL endpoints.
func NewURLReader(urlArtifact *v1alpha1.URLArtifact) (ArtifactReader, error) {
	if urlArtifact == nil {
		return nil, errors.New("URLArtifact cannot be empty")
	}
	return &URLReader{urlArtifact}, nil
}

func (reader *URLReader) Read() ([]byte, error) {
	log := logging.NewArgoEventsLogger()
	log.Debugf("reading urlArtifact from %s", reader.urlArtifact.Path)
	insecureSkipVerify := !reader.urlArtifact.VerifyCert
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}
	resp, err := client.Get(reader.urlArtifact.Path)
	if err != nil {
		log.Warnf("failed to read url %s: %s", reader.urlArtifact.Path, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("failed to read %s. status code: %d", reader.urlArtifact.Path, resp.StatusCode)
		return nil, errors.Errorf("status code %v", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		log.Warnf("failed to read url body for %s: %s", reader.urlArtifact.Path, err)
		return nil, err
	}
	return content, nil
}
