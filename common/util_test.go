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

package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeHttpWriter struct {
	header  int
	payload []byte
}

func (f *fakeHttpWriter) Header() http.Header {
	return http.Header{}
}

func (f *fakeHttpWriter) Write(body []byte) (int, error) {
	f.payload = body
	return len(body), nil
}

func (f *fakeHttpWriter) WriteHeader(status int) {
	f.header = status
}

func TestHTTPMethods(t *testing.T) {
	f := &fakeHttpWriter{}
	SendSuccessResponse(f, "hello")
	assert.Equal(t, "hello", string(f.payload))
	assert.Equal(t, http.StatusOK, f.header)

	SendErrorResponse(f, "failure")
	assert.Equal(t, "failure", string(f.payload))
	assert.Equal(t, http.StatusBadRequest, f.header)
}

func TestFormatEndpoint(t *testing.T) {
	assert.Equal(t, "/hello", FormatEndpoint("hello"))
}

func TestFormattedURL(t *testing.T) {
	assert.Equal(t, "/test-url/fake", FormattedURL("test-url", "fake"))
}
