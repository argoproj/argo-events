package common

import (
	strings "strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func fakeTLSConfig(t *testing.T, insecureSkipVerify bool) *TLSConfig {
	t.Helper()
	if insecureSkipVerify == true {
		return &TLSConfig{InsecureSkipVerify: true}
	} else {
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
}

func fakeSASLConfig(t *testing.T) *SASLConfig {
	t.Helper()
	return &SASLConfig{
		Mechanism: "PLAIN",

		UserSecret: &corev1.SecretKeySelector{
			Key: "fake-key1",
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "user",
			},
		},
		PasswordSecret: &corev1.SecretKeySelector{
			Key: "fake-key2",
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "password",
			},
		},
	}
}

func TestValidateTLSConfig(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		c := &TLSConfig{}
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test insecureSkipVerify true", func(t *testing.T) {
		c := &TLSConfig{InsecureSkipVerify: true}
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test clientKeySecret is set, clientCertSecret is empty", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.CACertSecret = nil
		c.ClientCertSecret = nil
		err := ValidateTLSConfig(c)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "both clientCertSecret and clientKeySecret need to be configured"))
	})

	t.Run("test only caCertSecret is set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.ClientCertSecret = nil
		c.ClientKeySecret = nil
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test clientCertSecret and clientKeySecret are set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		c.CACertSecret = nil
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})

	t.Run("test all of 3 are set", func(t *testing.T) {
		c := fakeTLSConfig(t, false)
		err := ValidateTLSConfig(c)
		assert.Nil(t, err)
	})
}

func TestValidateSASLConfig(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		s := &SASLConfig{}
		err := ValidateSASLConfig(s)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid sasl config, both userSecret and passwordSecret must be defined"))
	})

	t.Run("test invalid Mechanism is set", func(t *testing.T) {
		s := fakeSASLConfig(t)
		s.Mechanism = "INVALIDSTRING"
		err := ValidateSASLConfig(s)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid sasl config. Possible values for SASL Mechanism are `OAUTHBEARER`, `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512` and `GSSAPI`"))
	})

	t.Run("test only User is set", func(t *testing.T) {
		s := fakeSASLConfig(t)
		s.PasswordSecret = nil
		err := ValidateSASLConfig(s)
		assert.NotNil(t, err)
	})

	t.Run("test only Password is set", func(t *testing.T) {
		s := fakeSASLConfig(t)
		s.UserSecret = nil
		err := ValidateSASLConfig(s)
		assert.NotNil(t, err)
	})

	t.Run("test all of 3 are set", func(t *testing.T) {
		s := fakeSASLConfig(t)
		err := ValidateSASLConfig(s)
		assert.Nil(t, err)
	})
}
