package common

import (
	strings "strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func fakeTLSConfig(t *testing.T) *TLSConfig {
	t.Helper()
	return &TLSConfig{
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

func TestValidateTLSConfig(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		c := &TLSConfig{}
		err := ValidateTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "please configure either caCertSecret, or clientCertSecret and clientKeySecret, or both"))
	})

	t.Run("test clientKeySecret is set, clientCertSecret is empty", func(t *testing.T) {
		c := fakeTLSConfig(t)
		c.CACertSecret = nil
		c.ClientCertSecret = nil
		err := ValidateTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "both clientCertSecret and clientKeySecret need to be configured"))
	})

	t.Run("test only caCertSecret is set", func(t *testing.T) {
		c := fakeTLSConfig(t)
		c.ClientCertSecret = nil
		c.ClientKeySecret = nil
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test clientCertSecret and clientKeySecret are set", func(t *testing.T) {
		c := fakeTLSConfig(t)
		c.CACertSecret = nil
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test all of 3 are set", func(t *testing.T) {
		c := fakeTLSConfig(t)
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})
}
