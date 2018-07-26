package store

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// URLReader implements the ArtifactReader interface for urls
type URLReader struct {
	urlArtifact *v1alpha1.URLArtifact
}

// NewURLReader creates a new ArtifactReader for workflows at URL endpoints.
func NewURLReader(urlArtifact *v1alpha1.URLArtifact) (ArtifactReader, error) {
	if urlArtifact == nil {
		panic("URLArtifact cannot be empty")
	}
	return &URLReader{urlArtifact}, nil
}

func (reader *URLReader) Read() ([]byte, error) {
	insecureSkipVerify := !reader.urlArtifact.VerifyCert
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(reader.urlArtifact.Path)
	if err != nil {
		log.Printf("Failed to read url %s. Err: %s\n", reader.urlArtifact.Path, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to read %s. Status code: %d\n", reader.urlArtifact.Path, resp.StatusCode)
		return nil, errors.New("Status code " + string(resp.StatusCode))
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read url body for %s. Err %s\n", reader.urlArtifact.Path, err)
		return nil, err
	}
	return content, nil
}
