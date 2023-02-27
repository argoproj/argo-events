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
	"strings"
	"testing"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
	assert.Equal(t, "test-url/fake", FormattedURL("test-url", "fake"))
}

type statusVal int
type foo struct {
	Name string
	Ye   int
	SS   *corev1.SecretKeySelector
	CM01 *corev1.ConfigMapKeySelector
}

type haha struct {
	Nani string
	S    *corev1.SecretKeySelector
	C    *corev1.ConfigMapKeySelector
}

type bar struct {
	Status statusVal
	FSlice []foo
	Lili   *string
	Hss    string
	Dd     int
	dd     int
	M1     map[string]haha
	M2     map[string]*haha
	ABC    *corev1.SecretKeySelector
	ABCD   *corev1.SecretKeySelector
	EFG    corev1.SecretKeySelector
	hello  *corev1.SecretKeySelector

	CM01 *corev1.ConfigMapKeySelector
}

var (
	testXObj = bar{
		dd:     3,
		Status: 5,
		FSlice: []foo{
			{
				Name: "asdb",
				Ye:   23,
				SS:   &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s01"}, Key: "key1"},
				CM01: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c01"}, Key: "ckey1"},
			},
			{
				Name: "sss",
				SS:   &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s02"}, Key: "key2"},
				CM01: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c02"}, Key: "ckey2"},
			},
		},
		M1: map[string]haha{
			"a": {
				Nani: "nani333",
				S:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s03"}, Key: "key3"},
				C:    &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c03"}, Key: "ckey3"},
			},
			"b": {Nani: "nani444"},
			"c": {
				Nani: "nani555",
				S:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s04"}, Key: "key4"},
				C:    &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c04"}, Key: "ckey4"},
			},
		},
		M2: map[string]*haha{
			"a": {
				Nani: "nani333",
				S:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s05"}, Key: "key5"},
				C:    &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c05"}, Key: "ckey5"},
			},
			"b": {Nani: "nani444"},
			"c": {
				Nani: "nani555",
				S:    &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s06"}, Key: "key6"},
				C:    &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c06"}, Key: "ckey6"},
			},
		},
		ABC:   &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s06"}, Key: "key7"},     // same name
		EFG:   corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s08"}, Key: "key8"},      // does not count
		hello: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s09"}, Key: "key9"},     // does not count
		CM01:  &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "c06"}, Key: "ckey7"}, // same name
	}
)

func TestFindTypeValues(t *testing.T) {
	t.Run("test find secretKeySelectors", func(t *testing.T) {
		values := findTypeValues(testXObj, SecretKeySelectorType)
		assert.Equal(t, len(values), 7)
		values = findTypeValues(&testXObj, SecretKeySelectorType)
		assert.Equal(t, len(values), 7)
	})

	t.Run("test find configMapKeySelectors", func(t *testing.T) {
		values := findTypeValues(testXObj, ConfigMapKeySelectorType)
		assert.Equal(t, len(values), 7)
		values = findTypeValues(&testXObj, ConfigMapKeySelectorType)
		assert.Equal(t, len(values), 7)
	})
}

func TestVolumesFromSecretsOrConfigMaps(t *testing.T) {
	t.Run("test secret volumes", func(t *testing.T) {
		vols, mounts := VolumesFromSecretsOrConfigMaps(&testXObj, SecretKeySelectorType)
		assert.Equal(t, len(vols), 6)
		assert.Equal(t, len(mounts), 6)
	})

	t.Run("test configmap volumes", func(t *testing.T) {
		vols, mounts := VolumesFromSecretsOrConfigMaps(&testXObj, ConfigMapKeySelectorType)
		assert.Equal(t, len(vols), 6)
		assert.Equal(t, len(mounts), 6)
	})
}

func fakeTLSConfig(t *testing.T, insecureSkipVerify bool) *apicommon.TLSConfig {
	t.Helper()
	if insecureSkipVerify == true {
		return &apicommon.TLSConfig{
			InsecureSkipVerify: true,
		}
	} else {
		return &apicommon.TLSConfig{
			CACertSecret: &corev1.SecretKeySelector{
				Key: "fake-key1",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake-name1",
				},
			},
			ClientCertSecret: &corev1.SecretKeySelector{
				Key: "fake-key2",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake-name2",
				},
			},
			ClientKeySecret: &corev1.SecretKeySelector{
				Key: "fake-key3",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fake-name3",
				},
			},
		}
	}
}

func TestGetTLSConfig(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		c := &apicommon.TLSConfig{}
		_, err := GetTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "neither of caCertSecret, clientCertSecret and clientKeySecret is configured"))
	})

	t.Run("test clientKeySecret is set, clientCertSecret is empty", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.CACertSecret = nil
		c.ClientCertSecret = nil
		_, err := GetTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "both of clientCertSecret and clientKeySecret need to be configured"))
	})

	t.Run("test only caCertSecret is set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.ClientCertSecret = nil
		c.ClientKeySecret = nil
		_, err := GetTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to read ca cert file"))
	})

	t.Run("test clientCertSecret and clientKeySecret are set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.CACertSecret = nil
		_, err := GetTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to load client cert key pair"))
	})

	t.Run("test all of 3 are set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		_, err := GetTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to read ca cert file"))
	})
}

func TestElementsMatch(t *testing.T) {
	assert.True(t, ElementsMatch(nil, nil))
	assert.True(t, ElementsMatch([]string{"hello"}, []string{"hello"}))
	assert.True(t, ElementsMatch([]string{"hello", "world"}, []string{"hello", "world"}))
	assert.True(t, ElementsMatch([]string{}, []string{}))

	assert.False(t, ElementsMatch([]string{"hello"}, nil))
	assert.False(t, ElementsMatch([]string{"hello"}, []string{}))
	assert.False(t, ElementsMatch([]string{}, []string{"hello"}))
	assert.False(t, ElementsMatch([]string{"hello"}, []string{"hello", "world"}))
	assert.False(t, ElementsMatch([]string{"hello", "world"}, []string{"hello"}))
	assert.False(t, ElementsMatch([]string{"hello", "world"}, []string{"hello", "moon"}))
	assert.True(t, ElementsMatch([]string{"hello", "world"}, []string{"world", "hello"}))
	assert.True(t, ElementsMatch([]string{"hello", "world", "hello"}, []string{"hello", "hello", "world", "world"}))
	assert.True(t, ElementsMatch([]string{"world", "hello"}, []string{"hello", "hello", "world", "world"}))
	assert.True(t, ElementsMatch([]string{"hello", "hello", "world", "world"}, []string{"world", "hello"}))
	assert.False(t, ElementsMatch([]string{"hello"}, []string{"*", "hello"}))
	assert.False(t, ElementsMatch([]string{"hello", "*"}, []string{"hello"}))
	assert.False(t, ElementsMatch([]string{"*", "hello", "*"}, []string{"hello"}))
	assert.False(t, ElementsMatch([]string{"hello"}, []string{"world", "world"}))
	assert.False(t, ElementsMatch([]string{"hello", "hello"}, []string{"world", "world"}))
}

func TestSliceContains(t *testing.T) {
	assert.True(t, SliceContains([]string{"hello", "*"}, "*"))
	assert.True(t, SliceContains([]string{"*", "world"}, "*"))
	assert.True(t, SliceContains([]string{"*", "world"}, "world"))
	assert.True(t, SliceContains([]string{"*", "hello", "*"}, "*"))
	assert.False(t, SliceContains([]string{"hello", "world"}, "*"))
}
